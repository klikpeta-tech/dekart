package dekart

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"dekart/src/proto"
	"dekart/src/server/dark/libraryquery"
	"dekart/src/server/library"
	"dekart/src/server/user"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	libWorkspaceA   = "aaaaaaaa-5555-0000-0000-000000000001"
	libWorkspaceB   = "bbbbbbbb-5555-0000-0000-000000000001"
	libReportA      = "aaaaaaaa-5555-0000-0000-000000000002"
	libDatasetA     = "aaaaaaaa-5555-0000-0000-000000000003"
	libDatasetA2    = "aaaaaaaa-5555-0000-0000-000000000004"
	libReportB      = "bbbbbbbb-5555-0000-0000-000000000002"
	libDatasetB     = "bbbbbbbb-5555-0000-0000-000000000003"
	libConnection   = "cccccccc-5555-0000-0000-000000000001"
	libStaffEmail   = "staff@klikpeta.example.com"
	libClientAdmin  = "boss@client-a.example.com"
	libEditorEmail  = "editor@client-a.example.com"
	libViewerEmail  = "viewer@client-a.example.com"
	libFarFuture    = "2099-12-31"
	libLongPast     = "2020-01-01"
	itemBoundaries  = "kecamatan_bps_2024"
	itemPlaces      = "tempat_usaha"
	itemUnpublished = "not_in_catalogue"
)

// testCatalogue is an in-memory Catalogue holding two items; it records the statements it is asked to run.
type testCatalogue struct {
	items      []library.Item
	taxonomy   library.Taxonomy
	areas      map[string]library.Area
	estimate   library.Estimate
	statements []string
}

func newTestCatalogue() *testCatalogue {
	return &testCatalogue{
		items: []library.Item{
			{
				ID: itemBoundaries, Title: "Statistik Kecamatan BPS 2024", RowCount: 900, Shape: library.Shape{Kind: "areas", AdminLevel: "kecamatan"},
				Columns: []string{"kode", "nama"}, AreaLevels: []string{"kabkota"},
				Table: libraryquery.Table{
					Schema: "lib", Name: "kecamatan_bps_2024", Columns: []string{"kode", "nama"},
					AreaColumns: []libraryquery.AreaColumn{{Length: 4, Column: "kode_kabkota"}},
				},
			},
			{
				ID: itemPlaces, Title: "Tempat & Usaha", HasTaxonomy: true, MaxRows: 5000, RowCount: 70,
				Table: libraryquery.Table{
					Schema: "lib", Name: "tempat_usaha", Columns: []string{"nama", "lat", "lon"},
					AreaColumns:    []libraryquery.AreaColumn{{Length: 4, Column: "kode_kabkota"}},
					CategoryColumn: "category_id", BrandColumn: "brand_id",
				},
			},
		},
		taxonomy: library.Taxonomy{
			Groups:     []library.CategoryGroup{{ID: "fnb", Title: "Makanan"}, {ID: "kesehatan", Title: "Kesehatan"}},
			Categories: []library.Category{{ID: "kafe", GroupID: "fnb", Title: "Kafe"}, {ID: "bar", GroupID: "fnb", Title: "Bar"}, {ID: "apotek", GroupID: "kesehatan", Title: "Apotek"}},
			Brands:     []library.Brand{{ID: "starbucks", CategoryID: "kafe", Title: "Starbucks"}, {ID: "guardian", CategoryID: "apotek", Title: "Guardian"}},
		},
		areas:    map[string]library.Area{"3273": {Code: "3273", Level: "kabkota", Name: "Kota Bandung"}},
		estimate: library.Estimate{RowCount: 1200},
	}
}

func (c *testCatalogue) ListItems(ctx context.Context) ([]library.Item, error) { return c.items, nil }
func (c *testCatalogue) GetItem(ctx context.Context, id string) (library.Item, error) {
	for _, item := range c.items {
		if item.ID == id {
			return item, nil
		}
	}
	return library.Item{}, library.ErrItemNotFound
}
func (c *testCatalogue) Taxonomy(ctx context.Context, itemID string) (library.Taxonomy, error) {
	return c.taxonomy, nil
}

// testAreaRowsByCategory is how many Places rows Kota Bandung holds per category.
var testAreaRowsByCategory = map[string]int64{"kafe": 10, "bar": 20, "apotek": 40}

func (c *testCatalogue) ListAreas(ctx context.Context, parentCode string, itemID string, allowed []string) ([]library.Area, error) {
	area := c.areas["3273"]
	if itemID == itemPlaces {
		for _, id := range allowed {
			area.RowCount += testAreaRowsByCategory[id]
		}
	}
	return []library.Area{area}, nil
}
func (c *testCatalogue) GetArea(ctx context.Context, code string) (library.Area, error) {
	area, ok := c.areas[code]
	if !ok {
		return library.Area{}, library.ErrItemNotFound
	}
	return area, nil
}
func (c *testCatalogue) EstimateSelection(ctx context.Context, item library.Item, selection libraryquery.Selection, allowed []string) (library.Estimate, error) {
	return c.estimate, nil
}
func (c *testCatalogue) SampleRows(ctx context.Context, statement string) (library.Sample, error) {
	c.statements = append(c.statements, statement)
	return library.Sample{Columns: []string{"nama"}, Rows: [][]string{{"Kopi A"}}}, nil
}

type libraryFixture struct {
	server    Server
	catalogue *testCatalogue
	jobs      *fakeJobStore
}

// asUser returns a context for a signed-in user with a role in workspace A.
func (f libraryFixture) asUser(email string, role proto.UserRole) context.Context {
	ctx := context.WithValue(context.Background(), user.ContextKey, &user.Claims{Email: email})
	return user.SetWorkspaceCtx(ctx, user.WorkspaceInfo{ID: libWorkspaceA, Name: "Client A", UserRole: role})
}

func (f libraryFixture) staff() context.Context {
	return f.asUser(libStaffEmail, proto.UserRole_ROLE_ADMIN)
}
func (f libraryFixture) editor() context.Context {
	return f.asUser(libEditorEmail, proto.UserRole_ROLE_EDITOR)
}

// grant writes a grant directly, bypassing the commands, so tests can set up dates in the past.
func (f libraryFixture) grant(t *testing.T, workspaceID, itemID, endsOn, categories string) {
	t.Helper()
	_, err := f.server.db.Exec(`insert into library_grant_log (workspace_id, item_id, status, ends_on, category_ids, authored_by) values ($1, $2, 1, $3, $4, 'test')`,
		workspaceID, itemID, endsOn, categories)
	require.NoError(t, err)
}

// newLibraryFixture builds a database with both migration tracks, a client workspace A with a map and two empty
// datasets, another client workspace B with its own map, and the library connection configured.
func newLibraryFixture(t *testing.T) libraryFixture {
	t.Helper()
	t.Setenv("DEKART_CLOUD", "")
	clearMetadataEnv(t)
	t.Setenv("DEKART_LICENSE_KEY", "")
	t.Setenv("DEKART_SQLITE_DB_PATH", "")
	t.Setenv("DEKART_DEFAULT_WORKSPACE_ADMIN", libStaffEmail)
	t.Setenv("DEKART_LIBRARY_CONNECTION_ID", libConnection)

	db := newMigratedSqliteDB(t)
	forkDir, err := filepath.Abs("../../../sqlite/klikpeta-migrations")
	require.NoError(t, err)
	driver, err := migratesqlite3.WithInstance(db, &migratesqlite3.Config{MigrationsTable: "klikpeta_schema_migrations"})
	require.NoError(t, err)
	m, err := migrate.NewWithDatabaseInstance("file://"+forkDir, "sqlite3", driver)
	require.NoError(t, err)
	require.NoError(t, m.Up())

	exec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	exec(`insert into workspaces (id, name) values ($1, 'Client A'), ($2, 'Client B')`, libWorkspaceA, libWorkspaceB)
	exec(`insert into reports (id, author_email, workspace_id, title) values ($1, $2, $3, 'Map A'), ($4, 'someone@client-b.example.com', $5, 'Map B')`,
		libReportA, libEditorEmail, libWorkspaceA, libReportB, libWorkspaceB)
	exec(`insert into datasets (id, report_id, name) values ($1, $2, 'Empty'), ($3, $2, 'Empty 2'), ($4, $5, 'Empty B')`,
		libDatasetA, libReportA, libDatasetA2, libDatasetB, libReportB)
	// The library connection lives in workspace A on purpose: the cross-workspace guard alone would let it through.
	exec(`insert into connections (id, connection_name, workspace_id, connection_type, postgres_host, postgres_username, postgres_database, postgres_port)
		values ($1, 'Library', $2, $3, 'library.example.internal', 'reader', 'library', 5432)`,
		libConnection, libWorkspaceA, int(proto.ConnectionType_CONNECTION_TYPE_POSTGRES))

	catalogue := newTestCatalogue()
	jobs := &fakeJobStore{}
	server := NewServer(db, fakeStorage{}, jobs)
	server.SetLibraryCatalogue(catalogue)
	return libraryFixture{server: *server, catalogue: catalogue, jobs: jobs}
}

func requireCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, code, status.Code(err), "got: %v", err)
}

func (f libraryFixture) grantRows(t *testing.T) int {
	t.Helper()
	var rows int
	require.NoError(t, f.server.db.QueryRow(`select count(*) from library_grant_log`).Scan(&rows))
	return rows
}

// Admin commands: workspace Admin and Klikpeta staff are both required.
func TestLibraryAdminCommandsNeedAdminRoleAndStaffIdentity(t *testing.T) {
	f := newLibraryFixture(t)
	callers := []struct {
		name string
		ctx  context.Context
		code codes.Code
	}{
		{"anonymous", context.Background(), codes.Unauthenticated},
		{"viewer", f.asUser(libViewerEmail, proto.UserRole_ROLE_VIEWER), codes.PermissionDenied},
		{"editor", f.editor(), codes.PermissionDenied},
		{"staff identity but only editor role", f.asUser(libStaffEmail, proto.UserRole_ROLE_EDITOR), codes.PermissionDenied},
		{"client admin without staff identity", f.asUser(libClientAdmin, proto.UserRole_ROLE_ADMIN), codes.PermissionDenied},
	}
	for _, caller := range callers {
		t.Run(caller.name, func(t *testing.T) {
			_, err := f.server.SetLibraryGrant(caller.ctx, &proto.SetLibraryGrantRequest{ItemId: itemBoundaries, EndsOn: libFarFuture})
			requireCode(t, err, caller.code)
			_, err = f.server.RevokeLibraryGrant(caller.ctx, &proto.RevokeLibraryGrantRequest{ItemId: itemBoundaries})
			requireCode(t, err, caller.code)
			_, err = f.server.SetLibraryGrantCategories(caller.ctx, &proto.SetLibraryGrantCategoriesRequest{ItemId: itemPlaces, CategoryIds: []string{"kafe"}})
			requireCode(t, err, caller.code)
			_, err = f.server.ListLibraryGrants(caller.ctx, &proto.ListLibraryGrantsRequest{})
			requireCode(t, err, caller.code)
			_, err = f.server.GetLibraryTaxonomy(caller.ctx, &proto.GetLibraryTaxonomyRequest{ItemId: itemPlaces})
			requireCode(t, err, caller.code)
			require.Zero(t, f.grantRows(t), "a refused command must leave no grant")
		})
	}
}

// Without DEKART_DEFAULT_WORKSPACE_ADMIN nobody is staff, even a caller whose email is empty.
func TestLibraryAdminCommandsRefuseEveryoneWhenNoStaffIsConfigured(t *testing.T) {
	f := newLibraryFixture(t)
	t.Setenv("DEKART_DEFAULT_WORKSPACE_ADMIN", "")
	for _, email := range []string{libStaffEmail, ""} {
		_, err := f.server.SetLibraryGrant(f.asUser(email, proto.UserRole_ROLE_ADMIN), &proto.SetLibraryGrantRequest{ItemId: itemBoundaries, EndsOn: libFarFuture})
		requireCode(t, err, codes.PermissionDenied)
	}
	require.Zero(t, f.grantRows(t))
}

func TestLibraryStaffAdminManagesGrants(t *testing.T) {
	f := newLibraryFixture(t)
	ctx := f.staff()

	set, err := f.server.SetLibraryGrant(ctx, &proto.SetLibraryGrantRequest{ItemId: itemBoundaries, EndsOn: libFarFuture})
	require.NoError(t, err)
	require.Equal(t, proto.LibraryGrant_STATE_ACTIVE, set.Grant.State)
	require.Equal(t, libFarFuture, set.Grant.EndsOn)
	require.Equal(t, libStaffEmail, set.Grant.AuthoredBy)

	// Places needs categories, and they must exist
	_, err = f.server.SetLibraryGrant(ctx, &proto.SetLibraryGrantRequest{ItemId: itemPlaces, EndsOn: libFarFuture})
	requireCode(t, err, codes.InvalidArgument)
	_, err = f.server.SetLibraryGrant(ctx, &proto.SetLibraryGrantRequest{ItemId: itemPlaces, EndsOn: libFarFuture, CategoryIds: []string{"nope"}})
	requireCode(t, err, codes.InvalidArgument)
	_, err = f.server.SetLibraryGrant(ctx, &proto.SetLibraryGrantRequest{ItemId: itemBoundaries, EndsOn: libLongPast})
	requireCode(t, err, codes.InvalidArgument)
	_, err = f.server.SetLibraryGrant(ctx, &proto.SetLibraryGrantRequest{ItemId: itemUnpublished, EndsOn: libFarFuture})
	requireCode(t, err, codes.NotFound)

	places, err := f.server.SetLibraryGrant(ctx, &proto.SetLibraryGrantRequest{ItemId: itemPlaces, EndsOn: libFarFuture, CategoryIds: []string{"kafe", "bar"}})
	require.NoError(t, err)
	require.Equal(t, []string{"bar", "kafe"}, places.Grant.CategoryIds)
	changed, err := f.server.SetLibraryGrantCategories(ctx, &proto.SetLibraryGrantCategoriesRequest{ItemId: itemPlaces, CategoryIds: []string{"apotek"}})
	require.NoError(t, err)
	require.Equal(t, []string{"apotek"}, changed.Grant.CategoryIds)
	require.Equal(t, libFarFuture, changed.Grant.EndsOn)
	_, err = f.server.SetLibraryGrantCategories(ctx, &proto.SetLibraryGrantCategoriesRequest{ItemId: itemPlaces})
	requireCode(t, err, codes.InvalidArgument)

	list, err := f.server.ListLibraryGrants(ctx, &proto.ListLibraryGrantsRequest{})
	require.NoError(t, err)
	require.Len(t, list.Rows, 2, "the whole catalogue is listed")
	for _, row := range list.Rows {
		require.Equal(t, proto.LibraryGrant_STATE_ACTIVE, row.Grant.State, row.Item.Id)
	}

	revoked, err := f.server.RevokeLibraryGrant(ctx, &proto.RevokeLibraryGrantRequest{ItemId: itemBoundaries})
	require.NoError(t, err)
	require.Equal(t, proto.LibraryGrant_STATE_REVOKED, revoked.Grant.State)
	require.Equal(t, libFarFuture, revoked.Grant.EndsOn, "revoking keeps the date")
}

func TestLibraryGrantsApplyOnlyToTheWorkspaceTheyWereMadeIn(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceB, itemBoundaries, libFarFuture, "[]")

	list, err := f.server.ListLibraryItems(f.editor(), &proto.ListLibraryItemsRequest{})
	require.NoError(t, err)
	require.Empty(t, list.Items, "a grant of workspace B must not show in workspace A")
	require.EqualValues(t, 2, list.CatalogueSize)
}

func TestListLibraryItemsShowsOnlyActiveGrants(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemBoundaries, libFarFuture, "[]")
	f.grant(t, libWorkspaceA, itemPlaces, libLongPast, `["kafe"]`)

	list, err := f.server.ListLibraryItems(f.asUser(libViewerEmail, proto.UserRole_ROLE_VIEWER), &proto.ListLibraryItemsRequest{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1, "an ended grant removes the item from the library")
	require.Equal(t, itemBoundaries, list.Items[0].Id)
	require.NotZero(t, list.Items[0].AccessEndsAt)
}

func TestGetLibraryItemAnswersTheSameForUngrantedAndUnknown(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemBoundaries, libFarFuture, "[]")
	f.grant(t, libWorkspaceA, itemPlaces, libLongPast, `["kafe"]`)

	for _, itemID := range []string{itemPlaces, itemUnpublished} {
		_, err := f.server.GetLibraryItem(f.editor(), &proto.GetLibraryItemRequest{ItemId: itemID})
		requireCode(t, err, codes.NotFound)
	}
	_, err := f.server.GetLibraryItem(f.editor(), &proto.GetLibraryItemRequest{ItemId: itemBoundaries})
	require.NoError(t, err)
}

func TestGetLibraryItemShowsOnlyGrantedCategories(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)

	res, err := f.server.GetLibraryItem(f.editor(), &proto.GetLibraryItemRequest{ItemId: itemPlaces})
	require.NoError(t, err)
	require.Len(t, res.Taxonomy.Categories, 1)
	require.Equal(t, "kafe", res.Taxonomy.Categories[0].Id)
	require.Len(t, res.Taxonomy.Groups, 1)
	require.Equal(t, "fnb", res.Taxonomy.Groups[0].Id)
	require.Len(t, res.Taxonomy.Brands, 1, "brands of other categories must not leak")
	require.Equal(t, "starbucks", res.Taxonomy.Brands[0].Id)
}

func TestLibraryCommandsAreUnavailableWithoutACatalogue(t *testing.T) {
	f := newLibraryFixture(t)
	f.server.libraryCatalogue = nil
	_, err := f.server.ListLibraryItems(f.editor(), &proto.ListLibraryItemsRequest{})
	requireCode(t, err, codes.Unavailable)
	_, err = f.server.ListLibraryGrants(f.staff(), &proto.ListLibraryGrantsRequest{})
	requireCode(t, err, codes.Unavailable)
	_, err = f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemBoundaries})
	requireCode(t, err, codes.Unavailable)
}

func TestLibraryClientCommandsNeedSignIn(t *testing.T) {
	f := newLibraryFixture(t)
	_, err := f.server.ListLibraryItems(context.Background(), &proto.ListLibraryItemsRequest{})
	requireCode(t, err, codes.Unauthenticated)
	_, err = f.server.AddLibraryDataset(context.Background(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemBoundaries})
	requireCode(t, err, codes.Unauthenticated)
}

// The sample statement handed to the library database holds only categories that are both picked and granted.
func TestGetLibrarySampleUsesTheIntersectionOfPickedAndGrantedCategories(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)

	_, err := f.server.GetLibrarySample(f.editor(), &proto.GetLibrarySampleRequest{
		ItemId: itemPlaces, Selection: &proto.LibrarySelection{AreaCode: "3273", CategoryIds: []string{"kafe"}},
	})
	require.NoError(t, err)
	require.Len(t, f.catalogue.statements, 1)
	require.Contains(t, f.catalogue.statements[0], `"category_id" IN ('kafe')`)
	require.True(t, strings.HasSuffix(f.catalogue.statements[0], "LIMIT 20"))

	// picking an ungranted category is refused before anything reaches the library database
	_, err = f.server.GetLibrarySample(f.editor(), &proto.GetLibrarySampleRequest{
		ItemId: itemPlaces, Selection: &proto.LibrarySelection{CategoryIds: []string{"kafe", "apotek"}},
	})
	requireCode(t, err, codes.PermissionDenied)
	require.Len(t, f.catalogue.statements, 1)
}

func TestLibrarySelectionsRejectHostileInputBeforeTheLibraryDatabase(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	f.grant(t, libWorkspaceA, itemBoundaries, libFarFuture, "[]")

	hostile := []struct {
		name      string
		itemID    string
		selection *proto.LibrarySelection
	}{
		{"quote in area", itemBoundaries, &proto.LibrarySelection{AreaCode: "3273' OR '1'='1"}},
		{"statement in area", itemBoundaries, &proto.LibrarySelection{AreaCode: "3273'; DROP TABLE lib.kecamatan_bps_2024; --"}},
		{"area level the item lacks", itemBoundaries, &proto.LibrarySelection{AreaCode: "32"}},
		{"brands on a plain item", itemBoundaries, &proto.LibrarySelection{BrandIds: []string{"starbucks"}}},
		{"quote in brand", itemPlaces, &proto.LibrarySelection{CategoryIds: []string{"kafe"}, BrandIds: []string{"x' OR '1'='1"}}},
		{"no category", itemPlaces, &proto.LibrarySelection{}},
	}
	for _, c := range hostile {
		t.Run(c.name, func(t *testing.T) {
			_, err := f.server.GetLibrarySample(f.editor(), &proto.GetLibrarySampleRequest{ItemId: c.itemID, Selection: c.selection})
			require.Error(t, err)
			require.Contains(t, []codes.Code{codes.InvalidArgument, codes.PermissionDenied}, status.Code(err), "got: %v", err)
			_, err = f.server.CountLibrarySelection(f.editor(), &proto.CountLibrarySelectionRequest{ItemId: c.itemID, Selection: c.selection})
			require.Error(t, err)
			_, err = f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: c.itemID, Selection: c.selection})
			require.Error(t, err)
		})
	}
	require.Empty(t, f.catalogue.statements, "no hostile selection may reach the library database")
	var bindings int
	require.NoError(t, f.server.db.QueryRow(`select count(*) from library_dataset_bindings`).Scan(&bindings))
	require.Zero(t, bindings)
}

func TestCountLibrarySelectionFlagsSelectionsOverTheItemLimit(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	selection := &proto.LibrarySelection{CategoryIds: []string{"kafe"}}

	res, err := f.server.CountLibrarySelection(f.editor(), &proto.CountLibrarySelectionRequest{ItemId: itemPlaces, Selection: selection})
	require.NoError(t, err)
	require.False(t, res.ExceedsMaxRows)

	f.catalogue.estimate.RowCount = 5001
	res, err = f.server.CountLibrarySelection(f.editor(), &proto.CountLibrarySelectionRequest{ItemId: itemPlaces, Selection: selection})
	require.NoError(t, err)
	require.True(t, res.ExceedsMaxRows)
}

func TestAddLibraryDatasetBindsTheDatasetAndStoresNoSQL(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe","bar"]`)

	res, err := f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{
		DatasetId: libDatasetA, ItemId: itemPlaces,
		Selection: &proto.LibrarySelection{AreaCode: "3273", CategoryIds: []string{"kafe", "bar"}, BrandIds: []string{"starbucks"}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.QueryId)

	var connectionID, name, queryText string
	require.NoError(t, f.server.db.QueryRow(`select d.connection_id, d.name, q.query_text from datasets d join queries q on q.id=d.query_id where d.id=$1`, libDatasetA).
		Scan(&connectionID, &name, &queryText))
	require.Equal(t, libConnection, connectionID)
	require.Equal(t, "Tempat & Usaha · Starbucks · Kota Bandung", name)
	require.Empty(t, queryText, "the SQL is built by the server on every run, never stored from a request")

	binding, err := (library.Bindings{}).Get(context.Background(), f.server.db, libDatasetA)
	require.NoError(t, err)
	require.Equal(t, libReportA, binding.ReportID)
	require.Equal(t, libEditorEmail, binding.CreatedBy)
	require.Equal(t, "3273", binding.Selection.Area)
	require.EqualValues(t, 1, binding.Version)

	// the dataset is no longer empty
	_, err = f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemPlaces, Selection: &proto.LibrarySelection{CategoryIds: []string{"kafe"}}})
	requireCode(t, err, codes.FailedPrecondition)
}

func TestAddLibraryDatasetRefusals(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	f.grant(t, libWorkspaceA, itemBoundaries, libLongPast, "[]")
	f.grant(t, libWorkspaceB, itemBoundaries, libFarFuture, "[]")
	kafe := &proto.LibrarySelection{CategoryIds: []string{"kafe"}}

	cases := []struct {
		name string
		ctx  context.Context
		req  *proto.AddLibraryDatasetRequest
		code codes.Code
	}{
		{"viewer cannot write the map", f.asUser(libViewerEmail, proto.UserRole_ROLE_VIEWER), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemPlaces, Selection: kafe}, codes.NotFound},
		{"ungranted category", f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemPlaces, Selection: &proto.LibrarySelection{CategoryIds: []string{"apotek"}}}, codes.PermissionDenied},
		{"grant ended", f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemBoundaries, Selection: &proto.LibrarySelection{}}, codes.FailedPrecondition},
		{"never granted", f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemUnpublished, Selection: &proto.LibrarySelection{}}, codes.NotFound},
		{"dataset of another workspace", f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetB, ItemId: itemBoundaries, Selection: &proto.LibrarySelection{}}, codes.NotFound},
		{"malformed dataset id", f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: "x", ItemId: itemPlaces, Selection: kafe}, codes.InvalidArgument},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := f.server.AddLibraryDataset(c.ctx, c.req)
			requireCode(t, err, c.code)
		})
	}
	// the grant of workspace B must not serve a map of workspace A, and nothing was bound
	var bindings int
	require.NoError(t, f.server.db.QueryRow(`select count(*) from library_dataset_bindings`).Scan(&bindings))
	require.Zero(t, bindings)
	require.False(t, datasetConnectionID(t, f.server.db, libDatasetA).Valid, "the dataset must stay unbound")

	f.catalogue.estimate.RowCount = 5001
	_, err := f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemPlaces, Selection: kafe})
	requireCode(t, err, codes.FailedPrecondition)
	f.catalogue.estimate.RowCount = 10

	t.Setenv("DEKART_LIBRARY_CONNECTION_ID", "")
	_, err = f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemPlaces, Selection: kafe})
	requireCode(t, err, codes.Unavailable)
}

func TestUpdateLibraryDatasetSelectionIsCompareAndSet(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe","bar"]`)
	_, err := f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{
		DatasetId: libDatasetA, ItemId: itemPlaces, Selection: &proto.LibrarySelection{AreaCode: "3273", CategoryIds: []string{"kafe"}},
	})
	require.NoError(t, err)

	update := func(version int64, categories ...string) error {
		_, err := f.server.UpdateLibraryDatasetSelection(f.editor(), &proto.UpdateLibraryDatasetSelectionRequest{
			DatasetId: libDatasetA, ExpectedVersion: version, Selection: &proto.LibrarySelection{AreaCode: "3273", CategoryIds: categories},
		})
		return err
	}
	require.NoError(t, update(1, "bar"))
	requireCode(t, update(1, "kafe"), codes.Canceled)

	binding, err := (library.Bindings{}).Get(context.Background(), f.server.db, libDatasetA)
	require.NoError(t, err)
	require.EqualValues(t, 2, binding.Version)
	require.Equal(t, []string{"bar"}, binding.Selection.Categories, "a stale update must change nothing")

	var name string
	require.NoError(t, f.server.db.QueryRow(`select name from datasets where id=$1`, libDatasetA).Scan(&name))
	require.Equal(t, "Tempat & Usaha · Bar · Kota Bandung", name)

	// an ungranted category is refused, and so is any change after the grant ends
	requireCode(t, update(2, "apotek"), codes.PermissionDenied)
	_, err = f.server.db.Exec(`insert into library_grant_log (workspace_id, item_id, status, ends_on, category_ids, authored_by) values ($1, $2, 2, $3, '["kafe","bar"]', 'test')`, libWorkspaceA, itemPlaces, libFarFuture)
	require.NoError(t, err)
	requireCode(t, update(2, "kafe"), codes.FailedPrecondition)
	_, err = f.server.UpdateLibraryDatasetSelection(f.editor(), &proto.UpdateLibraryDatasetSelectionRequest{DatasetId: libDatasetA2, ExpectedVersion: 1, Selection: &proto.LibrarySelection{}})
	requireCode(t, err, codes.NotFound)
}

// Guards: nobody can bind or edit a dataset on the library connection through the ordinary commands.
func TestOrdinaryCommandsRefuseTheLibraryConnection(t *testing.T) {
	f := newLibraryFixture(t)

	_, err := f.server.CreateQuery(f.editor(), &proto.CreateQueryRequest{
		DatasetId: libDatasetA, ConnectionId: libConnection, ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
	})
	requireCode(t, err, codes.PermissionDenied)
	require.False(t, datasetConnectionID(t, f.server.db, libDatasetA).Valid)

	_, err = f.server.CreateFile(f.editor(), &proto.CreateFileRequest{DatasetId: libDatasetA, ConnectionId: libConnection})
	requireCode(t, err, codes.PermissionDenied)

	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	added, err := f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{DatasetId: libDatasetA, ItemId: itemPlaces, Selection: &proto.LibrarySelection{CategoryIds: []string{"kafe"}}})
	require.NoError(t, err)
	_, err = f.server.UpdateQuery(f.editor(), &proto.UpdateQueryRequest{QueryId: added.QueryId, QueryText: "select * from pg_shadow"})
	requireCode(t, err, codes.PermissionDenied)
	var queryText string
	require.NoError(t, f.server.db.QueryRow(`select query_text from queries where id=$1`, added.QueryId).Scan(&queryText))
	require.Empty(t, queryText)

	// Running must not take user text for a library dataset either, from a map or from a single dataset.
	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: added.QueryId, QueryText: "select * from pg_shadow"})
	require.NoError(t, err)
	_, err = f.server.RunAllQueries(f.editor(), &proto.RunAllQueriesRequest{ReportId: libReportA})
	require.NoError(t, err)
	for _, statement := range f.jobs.statements() {
		require.NotContains(t, statement, "pg_shadow")
		require.Contains(t, statement, `FROM "lib"."tempat_usaha"`)
	}
	require.NoError(t, f.server.db.QueryRow(`select query_text from queries where id=$1`, added.QueryId).Scan(&queryText))
	require.Empty(t, queryText)
}

// Row counts must not reveal categories the workspace was not granted.
func TestLibraryRowCountsCoverOnlyGrantedCategories(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	f.grant(t, libWorkspaceA, itemBoundaries, libFarFuture, `[]`)

	areas, err := f.server.ListLibraryAreas(f.editor(), &proto.ListLibraryAreasRequest{ParentCode: "32", ItemId: itemPlaces})
	require.NoError(t, err)
	require.Len(t, areas.Areas, 1)
	require.EqualValues(t, 10, areas.Areas[0].RowCount)

	items, err := f.server.ListLibraryItems(f.editor(), &proto.ListLibraryItemsRequest{})
	require.NoError(t, err)
	counts := map[string]int64{}
	for _, item := range items.Items {
		counts[item.Id] = item.RowCount
	}
	require.EqualValues(t, map[string]int64{itemPlaces: 0, itemBoundaries: 900}, counts)

	got, err := f.server.GetLibraryItem(f.editor(), &proto.GetLibraryItemRequest{ItemId: itemPlaces})
	require.NoError(t, err)
	require.EqualValues(t, 0, got.Item.RowCount)
}

// Without a configured library connection the guard must not match an empty connection id of an unbound dataset.
func TestLibraryConnectionGuardIgnoresUnsetConfiguration(t *testing.T) {
	require.NoError(t, refuseLibraryConnection(""))
	t.Setenv("DEKART_LIBRARY_CONNECTION_ID", "")
	require.NoError(t, refuseLibraryConnection(""))
	require.NoError(t, refuseLibraryConnection(libConnection))
	t.Setenv("DEKART_LIBRARY_CONNECTION_ID", libConnection)
	requireCode(t, refuseLibraryConnection(libConnection), codes.PermissionDenied)
	require.NoError(t, refuseLibraryConnection(""))
}
