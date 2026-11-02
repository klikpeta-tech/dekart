package library

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"dekart/src/server/dark/libraryquery"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

var itemColumns = []string{"id", "title", "description_md", "category", "provider", "attribution", "shape_kind", "admin_level", "h3_resolution",
	"coverage_summary", "row_count", "source_updated_on", "cadence", "columns", "view_name", "area_levels", "has_taxonomy", "max_rows", "layer_style"}

// newMockCatalogue returns a reader over a mock database and a clock the test can move.
func newMockCatalogue(t *testing.T) (*PGCatalogue, sqlmock.Sqlmock, *time.Time) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	clock := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	catalogue := NewPGCatalogue(func(context.Context) (*sql.DB, error) { return db, nil })
	catalogue.now = func() time.Time { return clock }
	return catalogue, mock, &clock
}

func placesAndBoundariesRows() *sqlmock.Rows {
	return sqlmock.NewRows(itemColumns).
		AddRow("places", "Tempat", "", "", "", "", "points", nil, nil, "Nasional", nil, nil, "quarterly",
			`["nama","lat","lon"]`, "places_view", "{kabkota,kecamatan}", true, 5000, nil).
		AddRow("kec", "Kecamatan", "", "", "", "", "areas", "kecamatan", nil, "Nasional", 7000, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), "",
			`["kode","nama"]`, "kec_view", "{provinsi}", false, nil, `{"opacity":1}`)
}

func TestPGCatalogueTurnsAnItemIntoAQueryTable(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	mock.ExpectQuery("lib_catalog.item").WillReturnRows(placesAndBoundariesRows())

	places, err := catalogue.GetItem(context.Background(), "places")
	require.NoError(t, err)
	require.Equal(t, libraryquery.Table{
		Schema: "lib", Name: "places_view", Columns: []string{"nama", "lat", "lon"},
		AreaColumns:    []libraryquery.AreaColumn{{Length: 4, Column: "kode_kabkota"}, {Length: 7, Column: "kode_kecamatan"}},
		CategoryColumn: "category_id", BrandColumn: "brand_id",
	}, places.Table)
	require.EqualValues(t, 5000, places.MaxRows)

	kec, err := catalogue.GetItem(context.Background(), "kec")
	require.NoError(t, err)
	require.Equal(t, libraryquery.Table{
		Schema: "lib", Name: "kec_view", Columns: []string{"kode", "nama"},
		AreaColumns: []libraryquery.AreaColumn{{Length: 2, Column: "kode_provinsi"}},
	}, kec.Table, "an item without a taxonomy has no category or brand column")
	require.Equal(t, `{"opacity":1}`, kec.LayerStyleJSON)
	require.EqualValues(t, 7000, kec.RowCount)
	require.Equal(t, 2024, kec.SourceUpdatedAt.Year())

	_, err = catalogue.GetItem(context.Background(), "retired_or_unknown")
	require.ErrorIs(t, err, ErrItemNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCatalogueCachesForSixtySeconds(t *testing.T) {
	catalogue, mock, clock := newMockCatalogue(t)
	mock.ExpectQuery("lib_catalog.item").WillReturnRows(placesAndBoundariesRows())

	// The mock allows one read; a second one within 60 s would fail the call.
	for i := 0; i < 3; i++ {
		_, err := catalogue.ListItems(context.Background())
		require.NoError(t, err)
	}
	*clock = clock.Add(59 * time.Second)
	_, err := catalogue.GetItem(context.Background(), "kec")
	require.NoError(t, err)

	*clock = clock.Add(2 * time.Second)
	mock.ExpectQuery("lib_catalog.item").WillReturnRows(placesAndBoundariesRows())
	_, err = catalogue.ListItems(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet(), "after 60 s the items are read again")
}

func TestPGCatalogueDoesNotCacheFailures(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	mock.ExpectQuery("lib_catalog.item").WillReturnError(errors.New("connection refused"))
	mock.ExpectQuery("lib_catalog.item").WillReturnRows(placesAndBoundariesRows())

	_, err := catalogue.ListItems(context.Background())
	require.Error(t, err)
	items, err := catalogue.ListItems(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestPGCatalogueOpensTheLibraryDatabaseUntilItSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	attempts := 0
	catalogue := NewPGCatalogue(func(context.Context) (*sql.DB, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("library connection is not ready")
		}
		return db, nil
	})
	mock.ExpectQuery("lib_catalog.item").WillReturnRows(placesAndBoundariesRows())

	_, err = catalogue.ListItems(context.Background())
	require.Error(t, err)
	_, err = catalogue.ListItems(context.Background())
	require.NoError(t, err)
	_, err = catalogue.GetItem(context.Background(), "kec")
	require.NoError(t, err)
	require.Equal(t, 2, attempts, "an opened database is kept")
}

func TestPGCatalogueAreaCountsDoNotLeakIntoTheCachedAreas(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	mock.ExpectQuery("lib_catalog.area").WillReturnRows(sqlmock.NewRows([]string{"kode", "level", "nama", "parent_kode"}).
		AddRow("3204", "kabkota", "Kab Bandung", "32").AddRow("3273", "kabkota", "Kota Bandung", "32"))
	mock.ExpectQuery("lib_catalog.item").WillReturnRows(placesAndBoundariesRows())
	mock.ExpectQuery("poi_count").WithArgs(4, "32%", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"left", "sum"}).AddRow("3273", 11))

	counted, err := catalogue.ListAreas(context.Background(), "32", "places", []string{"kafe"})
	require.NoError(t, err)
	require.EqualValues(t, 0, counted[0].RowCount)
	require.EqualValues(t, 11, counted[1].RowCount)

	plain, err := catalogue.ListAreas(context.Background(), "32", "", nil)
	require.NoError(t, err)
	require.EqualValues(t, 0, plain[1].RowCount, "counts for one workspace's categories must not be served to another")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCatalogueAreasRejectMalformedCodesBeforeTheDatabase(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	for _, code := range []string{"3", "327", "32'; drop table x;--", "３２"} {
		_, err := catalogue.ListAreas(context.Background(), code, "", nil)
		require.ErrorIs(t, err, libraryquery.ErrInvalidArea)
		_, err = catalogue.GetArea(context.Background(), code)
		require.ErrorIs(t, err, ErrItemNotFound)
	}
	_, err := catalogue.GetArea(context.Background(), "")
	require.ErrorIs(t, err, ErrItemNotFound)
	require.NoError(t, mock.ExpectationsWereMet(), "nothing was sent to the database")
}

func TestPGCatalogueUnknownAreaIsNotFound(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	mock.ExpectQuery("lib_catalog.area").WithArgs("9999").WillReturnError(sql.ErrNoRows)
	_, err := catalogue.GetArea(context.Background(), "9999")
	require.ErrorIs(t, err, ErrItemNotFound)
}

func TestPGCatalogueEstimatesWithoutATaxonomyFromAreaCounts(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	mock.ExpectQuery("item_area_count").WithArgs("kec", "3273").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(42))
	item := Item{ID: "kec", RowCount: 7000}

	whole, err := catalogue.EstimateSelection(context.Background(), item, libraryquery.Selection{}, nil)
	require.NoError(t, err)
	require.EqualValues(t, 7000, whole.RowCount, "whole coverage is the item's row count and needs no query")

	area, err := catalogue.EstimateSelection(context.Background(), item, libraryquery.Selection{Area: "3273"}, nil)
	require.NoError(t, err)
	require.EqualValues(t, 42, area.RowCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCatalogueEstimatesTaxonomyItemsOnlyOverGrantedCategories(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	item := Item{ID: "places", HasTaxonomy: true}
	mock.ExpectQuery("poi_count").WithArgs("3273%", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(5))
	mock.ExpectQuery("poi_count").WithArgs("3273%", sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"category_id", "sum"}).AddRow("bar", 20).AddRow("kafe", 11))
	mock.ExpectQuery("poi_count").WithArgs("3273%", sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"brand_id", "sum"}).AddRow("starbucks", 5))

	estimate, err := catalogue.EstimateSelection(context.Background(), item,
		libraryquery.Selection{Area: "3273", Categories: []string{"kafe", "apotek"}, Brands: []string{"starbucks"}}, []string{"kafe", "bar"})
	require.NoError(t, err)
	require.EqualValues(t, 5, estimate.RowCount)
	require.Equal(t, []Count{{ID: "bar", RowCount: 20}, {ID: "kafe", RowCount: 11}}, estimate.ByCategory)
	require.Equal(t, []Count{{ID: "starbucks", RowCount: 5}}, estimate.ByBrand)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCatalogueSampleRowsReturnsTextRows(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"nama", "lat"}).AddRow("Kopi A", 1.5).AddRow(nil, 2.5))

	sample, err := catalogue.SampleRows(context.Background(), "SELECT nama, lat FROM lib.places_view LIMIT 20")
	require.NoError(t, err)
	require.Equal(t, []string{"nama", "lat"}, sample.Columns)
	require.Equal(t, [][]string{{"Kopi A", "1.5"}, {"", "2.5"}}, sample.Rows)
}

func TestPGCatalogueTaxonomyOfAnItemWithoutOneIsEmpty(t *testing.T) {
	catalogue, mock, _ := newMockCatalogue(t)
	mock.ExpectQuery("lib_catalog.item").WillReturnRows(placesAndBoundariesRows())
	taxonomy, err := catalogue.Taxonomy(context.Background(), "kec")
	require.NoError(t, err)
	require.Empty(t, taxonomy.Categories)
	_, err = catalogue.Taxonomy(context.Background(), "unknown")
	require.ErrorIs(t, err, ErrItemNotFound)
}
