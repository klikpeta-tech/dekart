package library_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dekart/src/server/dark/libraryquery"
	"dekart/src/server/library"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

const (
	workspaceA = "aaaaaaaa-0000-0000-0000-000000000001"
	workspaceB = "bbbbbbbb-0000-0000-0000-000000000001"
)

var jakarta = time.FixedZone("WIB", 7*60*60)

// newDB returns a SQLite database with both migration tracks applied, as the server starts it.
func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "dekart.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, track := range []struct{ dir, table string }{
		{"../../../sqlite/migrations", ""},
		{"../../../sqlite/klikpeta-migrations", "klikpeta_schema_migrations"},
	} {
		driver, err := sqlite3.WithInstance(db, &sqlite3.Config{MigrationsTable: track.table})
		if err != nil {
			t.Fatal(err)
		}
		dir, err := filepath.Abs(track.dir)
		if err != nil {
			t.Fatal(err)
		}
		m, err := migrate.NewWithDatabaseInstance("file://"+dir, "sqlite3", driver)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Up(); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

var (
	boundaries = library.Item{
		ID: "kecamatan_bps_2024", Title: "Statistik Kecamatan BPS 2024",
		Table: libraryquery.Table{
			Schema: "lib", Name: "kecamatan_bps_2024", Columns: []string{"kode", "nama"},
			AreaColumns: []libraryquery.AreaColumn{{Length: 4, Column: "kode_kabkota"}},
		},
	}
	places = library.Item{
		ID: "tempat_usaha", Title: "Tempat & Usaha", HasTaxonomy: true, MaxRows: 100000,
		Table: libraryquery.Table{
			Schema: "lib", Name: "tempat_usaha", Columns: []string{"nama"},
			AreaColumns:    []libraryquery.AreaColumn{{Length: 4, Column: "kode_kabkota"}},
			CategoryColumn: "category_id", BrandColumn: "brand_id",
		},
	}
	taxonomy = library.Taxonomy{Categories: []library.Category{{ID: "kafe"}, {ID: "bar"}, {ID: "apotek"}}}
)

func must[T any](t *testing.T) func(T, error) T {
	return func(value T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
}

func TestGrantLatestEntryWinsEvenWithinTheSameSecond(t *testing.T) {
	grants := library.Grants{DB: newDB(t)}
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, jakarta)
	// grant, revoke and re-grant with a new date, all inside one second
	must[library.Grant](t)(grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2026-12-31", nil, "staff@klikpeta.com"))
	must[library.Grant](t)(grants.Revoke(ctx, workspaceA, boundaries.ID, "staff@klikpeta.com"))
	revoked, ok := must2(t)(grants.Latest(ctx, workspaceA, boundaries.ID))
	if !ok || !revoked.Revoked {
		t.Fatalf("revocation must be the latest entry, got %+v", revoked)
	}
	must[library.Grant](t)(grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2027-01-31", nil, "staff@klikpeta.com"))

	latest, ok := must2(t)(grants.Latest(ctx, workspaceA, boundaries.ID))
	if !ok || latest.Revoked || latest.EndsOn != "2027-01-31" {
		t.Fatalf("re-grant must be the latest entry, got %+v", latest)
	}
}

func must2(t *testing.T) func(library.Grant, bool, error) (library.Grant, bool) {
	return func(grant library.Grant, ok bool, err error) (library.Grant, bool) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return grant, ok
	}
}

func TestAuthorizeStates(t *testing.T) {
	grants := library.Grants{DB: newDB(t)}
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, jakarta)

	if _, err := grants.Authorize(ctx, workspaceA, boundaries.ID, now); !errors.Is(err, library.ErrNotGranted) {
		t.Fatalf("never granted: got %v", err)
	}
	must[library.Grant](t)(grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2026-11-10", nil, "staff"))
	if _, err := grants.Authorize(ctx, workspaceA, boundaries.ID, now); err != nil {
		t.Fatalf("granted: got %v", err)
	}
	if _, err := grants.Authorize(ctx, workspaceB, boundaries.ID, now); !errors.Is(err, library.ErrNotGranted) {
		t.Fatalf("another workspace must not be granted, got %v", err)
	}
	if _, err := grants.Authorize(ctx, workspaceA, places.ID, now); !errors.Is(err, library.ErrNotGranted) {
		t.Fatalf("another item must not be granted, got %v", err)
	}
	must[library.Grant](t)(grants.Revoke(ctx, workspaceA, boundaries.ID, "staff"))
	grant, err := grants.Authorize(ctx, workspaceA, boundaries.ID, now)
	if !errors.Is(err, library.ErrAccessEnded) || grant.EndsOn != "2026-11-10" {
		t.Fatalf("revoked: got %v %+v", err, grant)
	}
}

// The end date is the last day of access, read in Asia/Jakarta: open through 23:59:59, closed from midnight.
func TestAccessEndsAtEndOfDayInJakarta(t *testing.T) {
	grants := library.Grants{DB: newDB(t)}
	ctx := context.Background()
	setAt := time.Date(2026, 11, 2, 9, 0, 0, 0, jakarta)
	must[library.Grant](t)(grants.Set(ctx, setAt, workspaceA, boundaries, taxonomy, "2026-11-10", nil, "staff"))

	cases := []struct {
		name string
		now  time.Time
		open bool
	}{
		{"morning of the end date", time.Date(2026, 11, 10, 8, 0, 0, 0, jakarta), true},
		{"23:59:59 Jakarta", time.Date(2026, 11, 10, 23, 59, 59, 0, jakarta), true},
		{"23:59:59.9 Jakarta", time.Date(2026, 11, 10, 23, 59, 59, 900_000_000, jakarta), true},
		{"midnight Jakarta", time.Date(2026, 11, 11, 0, 0, 0, 0, jakarta), false},
		{"00:00:01 Jakarta", time.Date(2026, 11, 11, 0, 0, 1, 0, jakarta), false},
		{"16:59:59 UTC is still the end date in Jakarta", time.Date(2026, 11, 10, 16, 59, 59, 0, time.UTC), true},
		{"17:00:00 UTC is already the next day in Jakarta", time.Date(2026, 11, 10, 17, 0, 0, 0, time.UTC), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := grants.Authorize(ctx, workspaceA, boundaries.ID, c.now)
			if c.open && err != nil {
				t.Fatalf("want open, got %v", err)
			}
			if !c.open && !errors.Is(err, library.ErrAccessEnded) {
				t.Fatalf("want ended, got %v", err)
			}
		})
	}
}

func TestAccessEndsAtIsTheLastSecondOfTheDay(t *testing.T) {
	got, err := library.AccessEndsAt("2026-11-10")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 11, 10, 23, 59, 59, 0, jakarta); !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestSetRejectsBadOrPastEndDates(t *testing.T) {
	grants := library.Grants{DB: newDB(t)}
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, jakarta)
	for _, endsOn := range []string{"", "2026-02-30", "2026-1-5", "10/11/2026", "2026-11-10T00:00:00Z", "2026-11-10 ", "tomorrow"} {
		if _, err := grants.Set(ctx, now, workspaceA, boundaries, taxonomy, endsOn, nil, "staff"); !errors.Is(err, library.ErrInvalidEndDate) {
			t.Errorf("%q: got %v, want ErrInvalidEndDate", endsOn, err)
		}
	}
	if _, err := grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2026-11-01", nil, "staff"); !errors.Is(err, library.ErrEndDateInPast) {
		t.Errorf("yesterday: got %v, want ErrEndDateInPast", err)
	}
	if _, err := grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2026-11-02", nil, "staff"); err != nil {
		t.Errorf("today in Jakarta must be allowed: %v", err)
	}
	if _, ok := must2(t)(grants.Latest(ctx, workspaceB, boundaries.ID)); ok {
		t.Error("a refused grant must leave no record")
	}
}

func TestGrantCategories(t *testing.T) {
	grants := library.Grants{DB: newDB(t)}
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, jakarta)

	if _, err := grants.Set(ctx, now, workspaceA, places, taxonomy, "2026-12-31", nil, "staff"); !errors.Is(err, library.ErrCategoriesRequired) {
		t.Errorf("taxonomy item without categories: got %v", err)
	}
	if _, err := grants.Set(ctx, now, workspaceA, places, taxonomy, "2026-12-31", []string{"nonexistent"}, "staff"); !errors.Is(err, library.ErrUnknownCategory) {
		t.Errorf("unknown category: got %v", err)
	}
	if _, err := grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2026-12-31", []string{"kafe"}, "staff"); !errors.Is(err, library.ErrCategoriesUnexpected) {
		t.Errorf("categories on a plain item: got %v", err)
	}
	granted := must[library.Grant](t)(grants.Set(ctx, now, workspaceA, places, taxonomy, "2026-12-31", []string{"kafe", "bar", "kafe"}, "staff"))
	if len(granted.CategoryIDs) != 2 || granted.CategoryIDs[0] != "bar" || granted.CategoryIDs[1] != "kafe" {
		t.Errorf("categories must be sorted and deduplicated, got %v", granted.CategoryIDs)
	}

	// changing categories keeps the end date and cannot empty the set
	changed := must[library.Grant](t)(grants.SetCategories(ctx, workspaceA, places, taxonomy, []string{"apotek"}, "staff"))
	if changed.EndsOn != "2026-12-31" || len(changed.CategoryIDs) != 1 || changed.CategoryIDs[0] != "apotek" {
		t.Errorf("got %+v", changed)
	}
	if _, err := grants.SetCategories(ctx, workspaceA, places, taxonomy, nil, "staff"); !errors.Is(err, library.ErrCategoriesRequired) {
		t.Errorf("emptying categories: got %v", err)
	}
	latest, _ := must2(t)(grants.Latest(ctx, workspaceA, places.ID))
	if len(latest.CategoryIDs) != 1 || latest.CategoryIDs[0] != "apotek" {
		t.Errorf("a refused change must leave the categories as they were, got %v", latest.CategoryIDs)
	}

	must[library.Grant](t)(grants.Revoke(ctx, workspaceA, places.ID, "staff"))
	if _, err := grants.SetCategories(ctx, workspaceA, places, taxonomy, []string{"kafe"}, "staff"); !errors.Is(err, library.ErrNotGranted) {
		t.Errorf("categories of a revoked grant: got %v", err)
	}
}

func TestRevokeKeepsEndDateAndCategories(t *testing.T) {
	grants := library.Grants{DB: newDB(t)}
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, jakarta)
	if _, err := grants.Revoke(ctx, workspaceA, places.ID, "staff"); !errors.Is(err, library.ErrNotGranted) {
		t.Fatalf("revoking a never granted item: got %v", err)
	}
	must[library.Grant](t)(grants.Set(ctx, now, workspaceA, places, taxonomy, "2026-12-31", []string{"kafe"}, "staff"))
	revoked := must[library.Grant](t)(grants.Revoke(ctx, workspaceA, places.ID, "other-staff"))
	if !revoked.Revoked || revoked.EndsOn != "2026-12-31" || len(revoked.CategoryIDs) != 1 || revoked.AuthoredBy != "other-staff" {
		t.Fatalf("got %+v", revoked)
	}
	if revoked.StateAt(now) != library.StateRevoked {
		t.Fatalf("state: %v", revoked.StateAt(now))
	}
}

func TestForWorkspaceReturnsLatestPerItemOnlyForThatWorkspace(t *testing.T) {
	grants := library.Grants{DB: newDB(t)}
	ctx := context.Background()
	now := time.Date(2026, 11, 2, 9, 0, 0, 0, jakarta)
	must[library.Grant](t)(grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2026-12-31", nil, "staff"))
	must[library.Grant](t)(grants.Set(ctx, now, workspaceA, boundaries, taxonomy, "2027-03-31", nil, "staff"))
	must[library.Grant](t)(grants.Set(ctx, now, workspaceA, places, taxonomy, "2026-12-31", []string{"kafe"}, "staff"))
	must[library.Grant](t)(grants.Revoke(ctx, workspaceA, places.ID, "staff"))
	must[library.Grant](t)(grants.Set(ctx, now, workspaceB, boundaries, taxonomy, "2026-11-30", nil, "staff"))

	got, err := grants.ForWorkspace(ctx, workspaceA)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 items, got %+v", got)
	}
	byItem := map[string]library.Grant{}
	for _, grant := range got {
		byItem[grant.ItemID] = grant
	}
	if byItem[boundaries.ID].EndsOn != "2027-03-31" || byItem[boundaries.ID].Revoked {
		t.Errorf("boundaries: %+v", byItem[boundaries.ID])
	}
	if !byItem[places.ID].Revoked {
		t.Errorf("places must show the revocation: %+v", byItem[places.ID])
	}
	for _, grant := range got {
		if grant.WorkspaceID != workspaceA {
			t.Errorf("leaked a grant of another workspace: %+v", grant)
		}
	}
}

func TestBindingsRoundTripAndCompareAndSet(t *testing.T) {
	db := newDB(t)
	ctx := context.Background()
	bindings := library.Bindings{}
	binding := library.Binding{
		DatasetID: "dataset-1", ReportID: "report-1", ItemID: places.ID, CreatedBy: "editor@client.example",
		Selection: libraryquery.Selection{Area: "3273", Categories: []string{"kafe"}, Brands: []string{"starbucks"}},
	}
	if _, err := bindings.Get(ctx, db, "dataset-1"); !errors.Is(err, library.ErrBindingNotFound) {
		t.Fatalf("got %v", err)
	}
	if err := bindings.Insert(ctx, db, binding); err != nil {
		t.Fatal(err)
	}
	if err := bindings.Insert(ctx, db, binding); err == nil {
		t.Fatal("a dataset has at most one binding")
	}
	got, err := bindings.Get(ctx, db, "dataset-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || got.Selection.Area != "3273" || got.Selection.Categories[0] != "kafe" || got.Selection.Brands[0] != "starbucks" || got.ItemID != places.ID {
		t.Fatalf("got %+v", got)
	}

	// two updates in the same second must still be told apart: a counter, not a timestamp
	next := libraryquery.Selection{Area: "32", Categories: []string{"bar"}}
	version, err := bindings.UpdateSelection(ctx, db, "dataset-1", 1, next)
	if err != nil || version != 2 {
		t.Fatalf("got %d %v", version, err)
	}
	if _, err := bindings.UpdateSelection(ctx, db, "dataset-1", 1, libraryquery.Selection{Area: "31", Categories: []string{"kafe"}}); !errors.Is(err, library.ErrVersionConflict) {
		t.Fatalf("stale version must conflict, got %v", err)
	}
	got, _ = bindings.Get(ctx, db, "dataset-1")
	if got.Version != 2 || got.Selection.Area != "32" || len(got.Selection.Brands) != 0 {
		t.Fatalf("a conflicting update must change nothing, got %+v", got)
	}
	if _, err := bindings.UpdateSelection(ctx, db, "missing", 1, next); !errors.Is(err, library.ErrBindingNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildSQLAppliesOnlyTheCurrentlyGrantedCategories(t *testing.T) {
	selection := libraryquery.Selection{Area: "3273", Categories: []string{"kafe", "bar"}}
	both := library.Grant{CategoryIDs: []string{"bar", "kafe"}}
	narrowed := library.Grant{CategoryIDs: []string{"kafe"}}

	wide, err := library.BuildSQL(places, selection, both)
	if err != nil {
		t.Fatal(err)
	}
	narrow, err := library.BuildSQL(places, selection, narrowed)
	if err != nil {
		t.Fatal(err)
	}
	if wide == narrow {
		t.Fatal("revoking a category must change the SQL of an existing binding")
	}
	if want := `"category_id" IN ('kafe')`; !strings.Contains(narrow, want) || strings.Contains(narrow, "bar") {
		t.Fatalf("narrowed SQL:\n%s", narrow)
	}
	if _, err := library.BuildSQL(places, selection, library.Grant{CategoryIDs: []string{"apotek"}}); !errors.Is(err, libraryquery.ErrNoAllowedCategory) {
		t.Fatalf("a binding whose categories are all revoked must not build, got %v", err)
	}
}

func TestValidateSelection(t *testing.T) {
	grant := library.Grant{CategoryIDs: []string{"kafe", "bar"}}
	cases := []struct {
		name      string
		item      library.Item
		selection libraryquery.Selection
		want      error
	}{
		{"granted categories", places, libraryquery.Selection{Area: "3273", Categories: []string{"kafe"}}, nil},
		{"category outside the grant", places, libraryquery.Selection{Categories: []string{"apotek"}}, library.ErrCategoryNotGranted},
		{"one category outside the grant", places, libraryquery.Selection{Categories: []string{"kafe", "apotek"}}, library.ErrCategoryNotGranted},
		{"no category on a taxonomy item", places, libraryquery.Selection{}, libraryquery.ErrCategoryRequired},
		{"hostile category", places, libraryquery.Selection{Categories: []string{"kafe' OR '1'='1"}}, library.ErrCategoryNotGranted},
		{"hostile area", boundaries, libraryquery.Selection{Area: "3273' OR '1'='1"}, libraryquery.ErrInvalidArea},
		{"area level the item lacks", boundaries, libraryquery.Selection{Area: "32"}, libraryquery.ErrAreaLevelUnsupported},
		{"plain item", boundaries, libraryquery.Selection{Area: "3273"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := library.ValidateSelection(c.item, grant, c.selection); !errors.Is(err, c.want) && !(c.want == nil && err == nil) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}

func TestCheckSize(t *testing.T) {
	if err := library.CheckSize(places, library.Estimate{RowCount: 100000}); err != nil {
		t.Errorf("at the limit: %v", err)
	}
	if err := library.CheckSize(places, library.Estimate{RowCount: 100001}); !errors.Is(err, library.ErrMaxRowsExceeded) {
		t.Errorf("over the limit: %v", err)
	}
	if err := library.CheckSize(boundaries, library.Estimate{RowCount: 1 << 40}); err != nil {
		t.Errorf("an item without a limit: %v", err)
	}
}

func TestDatasetName(t *testing.T) {
	cases := []struct {
		picked []string
		area   string
		want   string
	}{
		{nil, "Kota Bandung", "Tempat · Kota Bandung"},
		{[]string{"Kafe"}, "Kota Bandung", "Tempat · Kafe · Kota Bandung"},
		{[]string{"Kafe", "Bar", "Apotek"}, "Kota Bandung", "Tempat · Kafe +2 · Kota Bandung"},
		{[]string{"Kafe"}, "", "Tempat · Kafe"},
	}
	for _, c := range cases {
		if got := library.DatasetName("Tempat", c.picked, c.area); got != c.want {
			t.Errorf("got %q want %q", got, c.want)
		}
	}
}
