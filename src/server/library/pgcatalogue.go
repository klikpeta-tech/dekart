package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"dekart/src/server/dark/libraryquery"
	"dekart/src/server/pgutils"

	"github.com/lib/pq"
)

// CacheTTL is how long the catalogue reader keeps what it read from the library database.
const CacheTTL = 60 * time.Second

// areaColumns are the lib view columns that hold the area code of each area level.
var areaColumns = map[string]libraryquery.AreaColumn{
	"provinsi":  {Length: 2, Column: "kode_provinsi"},
	"kabkota":   {Length: 4, Column: "kode_kabkota"},
	"kecamatan": {Length: 7, Column: "kode_kecamatan"},
}

// Taxonomy items filter rows by these lib view columns.
const (
	categoryColumn = "category_id"
	brandColumn    = "brand_id"
)

// PGCatalogue is a [Catalogue] that reads lib_catalog and lib in the library database.
//
// Reads of items, taxonomy and areas are cached for [CacheTTL]; counts and samples always go to the database.
type PGCatalogue struct {
	open func(ctx context.Context) (*sql.DB, error)
	now  func() time.Time

	mu    sync.Mutex
	db    *sql.DB
	cache map[string]cacheEntry
}

type cacheEntry struct {
	value   any
	expires time.Time
}

// NewPGCatalogue returns a catalogue reader. open returns the library database handle; it is called until it
// succeeds once, so the reader can start before the library connection is reachable.
func NewPGCatalogue(open func(ctx context.Context) (*sql.DB, error)) *PGCatalogue {
	return &PGCatalogue{open: open, now: time.Now, cache: map[string]cacheEntry{}}
}

// handle returns the library database, opening it on first use.
func (c *PGCatalogue) handle(ctx context.Context) (*sql.DB, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db != nil {
		return c.db, nil
	}
	db, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	c.db = db
	return db, nil
}

// cached returns the value stored under key while it is fresh, or loads and stores it.
func cached[T any](c *PGCatalogue, key string, load func() (T, error)) (T, error) {
	c.mu.Lock()
	entry, ok := c.cache[key]
	c.mu.Unlock()
	if ok && c.now().Before(entry.expires) {
		return entry.value.(T), nil
	}
	value, err := load()
	if err != nil {
		var zero T
		return zero, err
	}
	c.mu.Lock()
	c.cache[key] = cacheEntry{value: value, expires: c.now().Add(CacheTTL)}
	c.mu.Unlock()
	return value, nil
}

// itemTable builds the query builder's view of an item. The lib view is named by the catalogue and the
// area, category and brand columns follow from the item's area levels and taxonomy flag.
func itemTable(item Item, viewName string) libraryquery.Table {
	table := libraryquery.Table{Schema: "lib", Name: viewName, Columns: item.Columns}
	for _, level := range item.AreaLevels {
		if column, ok := areaColumns[level]; ok {
			table.AreaColumns = append(table.AreaColumns, column)
		}
	}
	if item.HasTaxonomy {
		table.CategoryColumn = categoryColumn
		table.BrandColumn = brandColumn
	}
	return table
}

// ListItems returns every published item.
func (c *PGCatalogue) ListItems(ctx context.Context) ([]Item, error) {
	return cached(c, "items", func() ([]Item, error) { return c.loadItems(ctx) })
}

// loadItems reads the published items in display order.
func (c *PGCatalogue) loadItems(ctx context.Context) ([]Item, error) {
	db, err := c.handle(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`select id, title, description_md, category, provider, attribution, shape_kind, admin_level, h3_resolution,
			coverage_summary, row_count, source_updated_on, cadence, columns, view_name, area_levels,
			has_taxonomy, max_rows, layer_style
		from lib_catalog.item where status = 'published' order by sort_order, title, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var item Item
		var adminLevel sql.NullString
		var h3Resolution, maxRows, rowCount sql.NullInt64
		var sourceUpdated sql.NullTime
		var columns, layerStyle []byte
		var viewName string
		var areaLevels pq.StringArray
		if err := rows.Scan(&item.ID, &item.Title, &item.DescriptionMD, &item.Category, &item.Provider, &item.Attribution,
			&item.Shape.Kind, &adminLevel, &h3Resolution, &item.CoverageSummary, &rowCount, &sourceUpdated, &item.Cadence,
			&columns, &viewName, &areaLevels, &item.HasTaxonomy, &maxRows, &layerStyle); err != nil {
			return nil, err
		}
		item.Shape.AdminLevel = adminLevel.String
		item.Shape.H3Resolution = int(h3Resolution.Int64)
		item.RowCount = rowCount.Int64
		item.MaxRows = maxRows.Int64
		item.SourceUpdatedAt = sourceUpdated.Time
		item.AreaLevels = []string(areaLevels)
		item.LayerStyleJSON = string(layerStyle)
		if err := json.Unmarshal(columns, &item.Columns); err != nil {
			return nil, fmt.Errorf("library: columns of item %s: %w", item.ID, err)
		}
		item.Table = itemTable(item, viewName)
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetItem returns one published item, or ErrItemNotFound.
func (c *PGCatalogue) GetItem(ctx context.Context, itemID string) (Item, error) {
	items, err := c.ListItems(ctx)
	if err != nil {
		return Item{}, err
	}
	for _, item := range items {
		if item.ID == itemID {
			return item, nil
		}
	}
	return Item{}, ErrItemNotFound
}

// Taxonomy returns the category tree of an item with nationwide counts per category.
func (c *PGCatalogue) Taxonomy(ctx context.Context, itemID string) (Taxonomy, error) {
	item, err := c.GetItem(ctx, itemID)
	if err != nil {
		return Taxonomy{}, err
	}
	if !item.HasTaxonomy {
		return Taxonomy{}, nil
	}
	return cached(c, "taxonomy", func() (Taxonomy, error) { return c.loadTaxonomy(ctx) })
}

// loadTaxonomy reads groups, categories with their nationwide counts, and brands.
func (c *PGCatalogue) loadTaxonomy(ctx context.Context) (Taxonomy, error) {
	db, err := c.handle(ctx)
	if err != nil {
		return Taxonomy{}, err
	}
	taxonomy := Taxonomy{}
	groups, err := db.QueryContext(ctx, `select id, title, sort_order from lib_catalog.category_group order by sort_order, title, id`)
	if err != nil {
		return Taxonomy{}, err
	}
	defer groups.Close()
	for groups.Next() {
		var group CategoryGroup
		if err := groups.Scan(&group.ID, &group.Title, &group.SortOrder); err != nil {
			return Taxonomy{}, err
		}
		taxonomy.Groups = append(taxonomy.Groups, group)
	}
	if err := groups.Err(); err != nil {
		return Taxonomy{}, err
	}
	categories, err := db.QueryContext(ctx,
		`select c.id, c.group_id, c.title, c.sort_order, coalesce(n.total, 0)
		from lib_catalog.category c
		left join (select category_id, sum(n) as total from lib_catalog.poi_count group by category_id) n on n.category_id = c.id
		order by c.sort_order, c.title, c.id`)
	if err != nil {
		return Taxonomy{}, err
	}
	defer categories.Close()
	for categories.Next() {
		var category Category
		if err := categories.Scan(&category.ID, &category.GroupID, &category.Title, &category.SortOrder, &category.RowCount); err != nil {
			return Taxonomy{}, err
		}
		taxonomy.Categories = append(taxonomy.Categories, category)
	}
	if err := categories.Err(); err != nil {
		return Taxonomy{}, err
	}
	brands, err := db.QueryContext(ctx, `select id, category_id, title from lib_catalog.brand order by category_id, title, id`)
	if err != nil {
		return Taxonomy{}, err
	}
	defer brands.Close()
	for brands.Next() {
		var brand Brand
		if err := brands.Scan(&brand.ID, &brand.CategoryID, &brand.Title); err != nil {
			return Taxonomy{}, err
		}
		taxonomy.Brands = append(taxonomy.Brands, brand)
	}
	return taxonomy, brands.Err()
}

// childLength is the code length of the areas under a parent code: provinces for "", then 4 and 7 digits.
func childLength(parentCode string) int {
	switch len(parentCode) {
	case 0:
		return 2
	case 2:
		return 4
	case 4:
		return 7
	}
	return 0
}

// ListAreas returns the children of parentCode, or the provinces for "". With itemID set it fills RowCount.
func (c *PGCatalogue) ListAreas(ctx context.Context, parentCode string, itemID string, allowed []string) ([]Area, error) {
	if !libraryquery.ValidArea(parentCode) {
		return nil, libraryquery.ErrInvalidArea
	}
	areas, err := cached(c, "areas:"+parentCode, func() ([]Area, error) { return c.loadAreas(ctx, parentCode) })
	if err != nil {
		return nil, err
	}
	if itemID == "" {
		return areas, nil
	}
	item, err := c.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	counts, err := c.areaCounts(ctx, item, parentCode, allowed)
	if err != nil {
		return nil, err
	}
	// The cached slice is shared, so counts go onto a copy.
	result := make([]Area, len(areas))
	for i, area := range areas {
		area.RowCount = counts[area.Code]
		result[i] = area
	}
	return result, nil
}

// loadAreas reads the areas directly under parentCode.
func (c *PGCatalogue) loadAreas(ctx context.Context, parentCode string) ([]Area, error) {
	db, err := c.handle(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx,
		`select kode, level, nama, coalesce(parent_kode, '') from lib_catalog.area
		where coalesce(parent_kode, '') = $1 order by nama, kode`, parentCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	areas := []Area{}
	for rows.Next() {
		var area Area
		if err := rows.Scan(&area.Code, &area.Level, &area.Name, &area.ParentCode); err != nil {
			return nil, err
		}
		areas = append(areas, area)
	}
	return areas, rows.Err()
}

// areaCounts returns the rows of an item per child area of parentCode. For a taxonomy item only rows of the
// allowed categories count, so the numbers never reveal categories the workspace is not granted.
func (c *PGCatalogue) areaCounts(ctx context.Context, item Item, parentCode string, allowed []string) (map[string]int64, error) {
	db, err := c.handle(ctx)
	if err != nil {
		return nil, err
	}
	length := childLength(parentCode)
	var rows *sql.Rows
	if item.HasTaxonomy {
		rows, err = db.QueryContext(ctx,
			`select left(kode_kecamatan, $1), sum(n) from lib_catalog.poi_count
			where kode_kecamatan like $2 and category_id = any($3) group by 1`,
			length, parentCode+"%", pq.Array(nonNilIDs(allowed)))
	} else {
		rows, err = db.QueryContext(ctx,
			`select kode, n from lib_catalog.item_area_count
			where item_id = $1 and length(kode) = $2 and kode like $3`,
			item.ID, length, parentCode+"%")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int64{}
	for rows.Next() {
		var code string
		var n int64
		if err := rows.Scan(&code, &n); err != nil {
			return nil, err
		}
		counts[code] = n
	}
	return counts, rows.Err()
}

func nonNilIDs(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

// GetArea returns one area by code, or ErrItemNotFound.
func (c *PGCatalogue) GetArea(ctx context.Context, code string) (Area, error) {
	if !libraryquery.ValidArea(code) || code == "" {
		return Area{}, ErrItemNotFound
	}
	return cached(c, "area:"+code, func() (Area, error) {
		db, err := c.handle(ctx)
		if err != nil {
			return Area{}, err
		}
		var area Area
		err = db.QueryRowContext(ctx,
			`select kode, level, nama, coalesce(parent_kode, '') from lib_catalog.area where kode = $1`, code).
			Scan(&area.Code, &area.Level, &area.Name, &area.ParentCode)
		if errors.Is(err, sql.ErrNoRows) {
			return Area{}, ErrItemNotFound
		}
		return area, err
	})
}

// EstimateSelection returns the expected rows of a selection from the count tables, never from the data views.
//
// Items without a taxonomy are counted per area in item_area_count, nationwide from the item's row count. For a
// taxonomy item the count is the sum of poi_count over the picked categories (restricted to allowed) and brands.
func (c *PGCatalogue) EstimateSelection(ctx context.Context, item Item, selection libraryquery.Selection, allowed []string) (Estimate, error) {
	db, err := c.handle(ctx)
	if err != nil {
		return Estimate{}, err
	}
	if !item.HasTaxonomy {
		if selection.Area == "" {
			return Estimate{RowCount: item.RowCount}, nil
		}
		var n int64
		err := db.QueryRowContext(ctx, `select coalesce(sum(n), 0) from lib_catalog.item_area_count where item_id = $1 and kode = $2`,
			item.ID, selection.Area).Scan(&n)
		return Estimate{RowCount: n}, err
	}
	picked := libraryquery.Intersect(selection.Categories, allowed)
	brands := nonNilIDs(selection.Brands)
	prefix := selection.Area + "%"
	estimate := Estimate{}
	if err := db.QueryRowContext(ctx,
		`select coalesce(sum(n), 0) from lib_catalog.poi_count
		where kode_kecamatan like $1 and category_id = any($2) and (cardinality($3::text[]) = 0 or brand_id = any($3))`,
		prefix, pq.Array(picked), pq.Array(brands)).Scan(&estimate.RowCount); err != nil {
		return Estimate{}, err
	}
	if estimate.ByCategory, err = c.countsBy(ctx, db, `category_id`, `category_id = any($2)`, prefix, allowed); err != nil {
		return Estimate{}, err
	}
	if estimate.ByBrand, err = c.countsBy(ctx, db, `brand_id`, `category_id = any($2)`, prefix, picked); err != nil {
		return Estimate{}, err
	}
	return estimate, nil
}

// countsBy sums poi_count per column over the rows of an area prefix whose category is in categories.
func (c *PGCatalogue) countsBy(ctx context.Context, db *sql.DB, column, condition, prefix string, categories []string) ([]Count, error) {
	// column and condition are constants of this package, never request data.
	rows, err := db.QueryContext(ctx,
		`select `+column+`, sum(n) from lib_catalog.poi_count where kode_kecamatan like $1 and `+condition+` group by 1 order by 1`,
		prefix, pq.Array(nonNilIDs(categories)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := []Count{}
	for rows.Next() {
		var count Count
		if err := rows.Scan(&count.ID, &count.RowCount); err != nil {
			return nil, err
		}
		counts = append(counts, count)
	}
	return counts, rows.Err()
}

// SampleRows runs a statement built by libraryquery and returns its column names and rows as text.
func (c *PGCatalogue) SampleRows(ctx context.Context, statement string) (Sample, error) {
	db, err := c.handle(ctx)
	if err != nil {
		return Sample{}, err
	}
	rows, err := db.QueryContext(ctx, statement)
	if err != nil {
		return Sample{}, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return Sample{}, err
	}
	sample := Sample{Columns: columns}
	for rows.Next() {
		row, err := pgutils.ScanRow(rows, len(columns))
		if err != nil {
			return Sample{}, err
		}
		sample.Rows = append(sample.Rows, row)
	}
	return sample, rows.Err()
}
