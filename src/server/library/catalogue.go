// Package library holds the Data Library domain: grants, map bindings and the catalogue contract.
//
// Grants and bindings are stored in the fork-only tables created by sqlite/klikpeta-migrations. The catalogue
// itself lives in the library database and is read through the Catalogue interface.
package library

import (
	"context"
	"errors"
	"os"
	"time"

	"dekart/src/server/dark/libraryquery"
)

// ErrItemNotFound is returned by a Catalogue when an item or area does not exist or is not published.
var ErrItemNotFound = errors.New("library: item not found")

// ConnectionID returns the id of the staff-owned library connection, or "" when none is configured.
func ConnectionID() string {
	return os.Getenv("DEKART_LIBRARY_CONNECTION_ID")
}

// Shape says what one row of an item is.
type Shape struct {
	// Kind is one of points, lines, areas or h3.
	Kind string
	// AdminLevel is the area level for areas: provinsi, kabkota or kecamatan.
	AdminLevel string
	// H3Resolution is the hexagon resolution for h3.
	H3Resolution int
}

// Item is one published catalogue entry.
type Item struct {
	ID              string
	Title           string
	DescriptionMD   string
	Category        string
	Provider        string
	Attribution     string
	CoverageSummary string
	Shape           Shape
	RowCount        int64
	SourceUpdatedAt time.Time
	Cadence         string
	Columns         []string
	// HasTaxonomy is true when access and selection are per category.
	HasTaxonomy bool
	// MaxRows is the largest selection that may be added to a map, 0 when unlimited.
	MaxRows        int64
	LayerStyleJSON string
	// AreaLevels lists the area levels the item can be filtered at: provinsi, kabkota, kecamatan.
	AreaLevels []string
	// Table tells the query builder where the rows are.
	Table libraryquery.Table
}

// Category is a taxonomy category of an item.
type Category struct {
	ID        string
	GroupID   string
	Title     string
	SortOrder int
	RowCount  int64
}

// CategoryGroup groups categories.
type CategoryGroup struct {
	ID        string
	Title     string
	SortOrder int
}

// Brand is a brand within a category; brands filter rows but never grant access.
type Brand struct {
	ID         string
	CategoryID string
	Title      string
}

// Taxonomy is the category tree of an item.
type Taxonomy struct {
	Groups     []CategoryGroup
	Categories []Category
	Brands     []Brand
}

// Area is a region with a BPS code.
type Area struct {
	Code       string
	Level      string
	Name       string
	ParentCode string
	// RowCount is the number of rows of the requested item in the area, 0 when no item was requested.
	RowCount int64
}

// Count is a row count for a category or brand id.
type Count struct {
	ID       string
	RowCount int64
}

// Estimate is the expected size of a selection.
type Estimate struct {
	RowCount   int64
	ByCategory []Count
	ByBrand    []Count
}

// Sample is a small preview of the rows of a selection.
type Sample struct {
	Columns []string
	Rows    [][]string
}

// Catalogue reads the curated catalogue from the library database. Implementations must cache for a short time
// and return only published items. Methods return ErrItemNotFound for unknown ids.
type Catalogue interface {
	// ListItems returns every published item.
	ListItems(ctx context.Context) ([]Item, error)
	// GetItem returns one published item.
	GetItem(ctx context.Context, itemID string) (Item, error)
	// Taxonomy returns the whole category tree of an item with nationwide counts.
	Taxonomy(ctx context.Context, itemID string) (Taxonomy, error)
	// ListAreas returns the children of parentCode, or the provinces for "". With itemID set it fills RowCount,
	// counting only rows of the allowed categories when the item has a taxonomy.
	ListAreas(ctx context.Context, parentCode string, itemID string, allowed []string) ([]Area, error)
	// GetArea returns one area by code.
	GetArea(ctx context.Context, code string) (Area, error)
	// EstimateSelection returns the expected row count of a selection, restricted to allowed categories.
	EstimateSelection(ctx context.Context, item Item, selection libraryquery.Selection, allowed []string) (Estimate, error)
	// SampleRows runs a statement built by libraryquery on the library database and returns its rows.
	SampleRows(ctx context.Context, statement string) (Sample, error)
}
