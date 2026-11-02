package dekart

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"dekart/src/proto"
	"dekart/src/server/job"
	"dekart/src/server/library"
	"dekart/src/server/storage"
	"dekart/src/server/user"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// fakeJobStore accepts every job and records the SQL it was asked to run.
type fakeJobStore struct {
	mu   sync.Mutex
	sqls []string
}

type fakeJob struct{ job.BasicJob }

func (j *fakeJob) Run(storage.StorageObject, *proto.Connection) error { return nil }

func (f *fakeJobStore) Create(reportID, queryID, queryText string, userCtx context.Context) (job.Job, chan int32, error) {
	f.mu.Lock()
	f.sqls = append(f.sqls, queryText)
	f.mu.Unlock()
	j := &fakeJob{job.BasicJob{ReportID: reportID, QueryID: queryID, QueryText: queryText}}
	j.Init(userCtx)
	return j, j.Status(), nil
}
func (f *fakeJobStore) TestConnection(context.Context, *proto.TestConnectionRequest) (*proto.TestConnectionResponse, error) {
	return &proto.TestConnectionResponse{}, nil
}
func (f *fakeJobStore) Cancel(string) bool        { return false }
func (f *fakeJobStore) CancelAll(context.Context) {}

func (f *fakeJobStore) statements() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sqls...)
}

// fakeStorage hands out no result files; fake jobs never write one.
type fakeStorage struct{ storage.Storage }

func (fakeStorage) GetObject(context.Context, string, string) storage.StorageObject { return nil }

// captureStream is a report stream that keeps what the server sends.
type captureStream struct {
	grpc.ServerStream
	ctx  context.Context
	sent []*proto.ReportStreamResponse
}

func (c *captureStream) Context() context.Context { return c.ctx }
func (c *captureStream) Send(response *proto.ReportStreamResponse) error {
	c.sent = append(c.sent, response)
	return nil
}

// addPlaces binds the first empty dataset of map A to Places (Kota Bandung, cafes) and returns its query id.
func (f libraryFixture) addPlaces(t *testing.T) string {
	t.Helper()
	res, err := f.server.AddLibraryDataset(f.editor(), &proto.AddLibraryDatasetRequest{
		DatasetId: libDatasetA, ItemId: itemPlaces,
		Selection: &proto.LibrarySelection{AreaCode: "3273", CategoryIds: []string{"kafe"}},
	})
	require.NoError(t, err)
	return res.QueryId
}

const placesKafeBandungSQL = "SELECT \"nama\", \"lat\", \"lon\"\nFROM \"lib\".\"tempat_usaha\"\nWHERE \"kode_kabkota\" = '3273'\n  AND \"category_id\" IN ('kafe')"

func TestLibraryRunsUseServerBuiltSQLOnly(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe","bar"]`)
	queryID := f.addPlaces(t)

	// Adding the dataset starts its first run.
	require.Equal(t, []string{placesKafeBandungSQL}, f.jobs.statements())

	// A refresh with supplied text builds the same SQL: supplied and stored text are ignored.
	runResponse, err := f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: queryID, QueryText: "select * from pg_shadow"})
	require.NoError(t, err)
	require.Equal(t, []string{placesKafeBandungSQL, placesKafeBandungSQL}, f.jobs.statements())
	require.NotEmpty(t, runResponse.QueryJob.Id)
	require.Empty(t, runResponse.QueryJob.QueryText, "the built SQL must not come back in the run response")

	// Polling the job over MCP does not reveal the SQL either.
	checkArgs, err := json.Marshal(map[string]string{"job_id": runResponse.QueryJob.Id})
	require.NoError(t, err)
	checked, err := f.server.callCheckJobStatusTool(f.editor(), checkArgs)
	require.NoError(t, err)
	require.NotContains(t, string(checked), "tempat_usaha")
	var queryText string
	require.NoError(t, f.server.db.QueryRow(`select query_text from queries where id=$1`, queryID).Scan(&queryText))
	require.Empty(t, queryText)

	// Run All refreshes it the same way.
	_, err = f.server.RunAllQueries(f.editor(), &proto.RunAllQueriesRequest{ReportId: libReportA})
	require.NoError(t, err)
	require.Len(t, f.jobs.statements(), 3)
	require.Equal(t, placesKafeBandungSQL, f.jobs.statements()[2])

	// Changing the selection runs the new selection.
	_, err = f.server.UpdateLibraryDatasetSelection(f.editor(), &proto.UpdateLibraryDatasetSelectionRequest{
		DatasetId: libDatasetA, ExpectedVersion: 1, Selection: &proto.LibrarySelection{AreaCode: "3273", CategoryIds: []string{"bar"}},
	})
	require.NoError(t, err)
	statements := f.jobs.statements()
	require.Contains(t, statements[len(statements)-1], `AND "category_id" IN ('bar')`)
}

func TestLibraryRunsDropCategoriesNoLongerGranted(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe","bar"]`)
	queryID := f.addPlaces(t)
	_, err := f.server.UpdateLibraryDatasetSelection(f.editor(), &proto.UpdateLibraryDatasetSelectionRequest{
		DatasetId: libDatasetA, ExpectedVersion: 1, Selection: &proto.LibrarySelection{CategoryIds: []string{"kafe", "bar"}},
	})
	require.NoError(t, err)

	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["bar"]`)
	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: queryID})
	require.NoError(t, err)
	statements := f.jobs.statements()
	last := statements[len(statements)-1]
	require.Contains(t, last, `IN ('bar')`)
	require.NotContains(t, last, "kafe")

	// With nothing left of the selection there is nothing to read.
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["apotek"]`)
	before := len(f.jobs.statements())
	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: queryID})
	requireCode(t, err, codes.FailedPrecondition)
	require.Len(t, f.jobs.statements(), before)
}

func TestLibraryRefreshIsRefusedAfterTheGrantEndsAndLastDataStays(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	queryID := f.addPlaces(t)
	before := len(f.jobs.statements())

	// The grant ends (revoked here; an end date in the past takes the same path).
	_, err := f.server.RevokeLibraryGrant(f.staff(), &proto.RevokeLibraryGrantRequest{ItemId: itemPlaces})
	require.NoError(t, err)
	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: queryID})
	requireCode(t, err, codes.FailedPrecondition)
	require.Contains(t, err.Error(), "library access ended")
	require.Len(t, f.jobs.statements(), before, "a refused refresh must not start a job")

	// Run All skips the ended dataset instead of failing the whole map.
	_, err = f.server.RunAllQueries(f.editor(), &proto.RunAllQueriesRequest{ReportId: libReportA})
	require.NoError(t, err)
	require.Len(t, f.jobs.statements(), before)

	// The stored result stays, and the stream says access ended.
	var jobs int
	require.NoError(t, f.server.db.QueryRow(`select count(*) from query_jobs where query_id=$1`, queryID).Scan(&jobs))
	require.Equal(t, 1, jobs)
	stream := &captureStream{ctx: f.editor()}
	require.NoError(t, f.server.sendReportMessage(libReportA, stream, 1))
	binding := stream.sent[0].Datasets[0].Library
	require.True(t, binding.AccessEnded)
	require.Len(t, stream.sent[0].QueryJobs, 1)

	// A new grant lifts the refusal.
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: queryID})
	require.NoError(t, err)
}

func TestLibraryRunRefusesDatasetWithoutBindingAndUnavailableCatalogue(t *testing.T) {
	f := newLibraryFixture(t)
	// A dataset that points at the library connection by some other route has no binding to build SQL from.
	_, err := f.server.db.Exec(`insert into queries (id, query_text, execution_engine) values ($1, 'select 1', $2)`,
		"dddddddd-5555-0000-0000-000000000001", int(proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION))
	require.NoError(t, err)
	_, err = f.server.db.Exec(`update datasets set query_id=$1, connection_id=$2 where id=$3`, "dddddddd-5555-0000-0000-000000000001", libConnection, libDatasetA)
	require.NoError(t, err)

	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: "dddddddd-5555-0000-0000-000000000001"})
	requireCode(t, err, codes.FailedPrecondition)
	_, err = f.server.RunAllQueries(f.editor(), &proto.RunAllQueriesRequest{ReportId: libReportA})
	require.NoError(t, err, "Run All skips a dataset that cannot run")
	require.Empty(t, f.jobs.statements())

	f.server.libraryCatalogue = nil
	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: "dddddddd-5555-0000-0000-000000000001"})
	requireCode(t, err, codes.Unavailable)
}

func TestStreamSendsTheBindingInsteadOfConnectionAndSQL(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	queryID := f.addPlaces(t)

	stream := &captureStream{ctx: f.editor()}
	require.NoError(t, f.server.sendReportMessage(libReportA, stream, 1))
	response := stream.sent[0]

	var dataset, plain *proto.Dataset
	for _, candidate := range response.Datasets {
		switch candidate.Id {
		case libDatasetA:
			dataset = candidate
		case libDatasetA2:
			plain = candidate
		}
	}
	require.NotNil(t, dataset)
	require.NotNil(t, plain)
	require.Empty(t, dataset.ConnectionId, "the library connection id must not reach the browser")
	require.Equal(t, itemPlaces, dataset.Library.ItemId)
	require.Equal(t, "3273", dataset.Library.Selection.AreaCode)
	require.Equal(t, []string{"kafe"}, dataset.Library.Selection.CategoryIds)
	require.EqualValues(t, 1, dataset.Library.Version)
	require.False(t, dataset.Library.AccessEnded)
	require.NotZero(t, dataset.Library.AccessEndsAt)
	require.Nil(t, plain.Library, "datasets that are not library datasets carry no binding")

	for _, query := range response.Queries {
		require.Empty(t, query.QueryText)
	}
	require.NotEmpty(t, response.QueryJobs)
	for _, queryJob := range response.QueryJobs {
		require.Equal(t, queryID, queryJob.QueryId)
		require.Empty(t, queryJob.QueryText, "the SQL must not reach the browser")
	}
	// The stored job keeps the SQL for the server's own use.
	var stored string
	require.NoError(t, f.server.db.QueryRow(`select query_text from query_jobs where query_id=$1`, queryID).Scan(&stored))
	require.Equal(t, placesKafeBandungSQL, stored)
}

func TestForkReportCopiesLibraryBindings(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	f.addPlaces(t)

	// Same workspace: the copy keeps the library connection and its own binding, and reuses the stored result.
	res, err := f.server.ForkReport(f.editor(), &proto.ForkReportRequest{ReportId: libReportA})
	require.NoError(t, err)
	var copiedDatasetID, copiedQueryID string
	require.NoError(t, f.server.db.QueryRow(`select id, query_id from datasets where report_id=$1 and connection_id=$2`, res.ReportId, libConnection).Scan(&copiedDatasetID, &copiedQueryID))
	binding, err := (library.Bindings{}).Get(context.Background(), f.server.db, copiedDatasetID)
	require.NoError(t, err)
	require.Equal(t, res.ReportId, binding.ReportID)
	require.Equal(t, itemPlaces, binding.ItemID)
	require.Equal(t, "3273", binding.Selection.Area)
	require.Equal(t, []string{"kafe"}, binding.Selection.Categories)
	require.EqualValues(t, 1, binding.Version)
	var jobs int
	require.NoError(t, f.server.db.QueryRow(`select count(*) from query_jobs where query_id=$1`, copiedQueryID).Scan(&jobs))
	require.Equal(t, 1, jobs)

	// The copy refreshes from its own binding.
	before := len(f.jobs.statements())
	_, err = f.server.RunQuery(f.editor(), &proto.RunQueryRequest{QueryId: copiedQueryID})
	require.NoError(t, err)
	require.Equal(t, placesKafeBandungSQL, f.jobs.statements()[before])
}

func TestForkReportOfAnotherWorkspaceKeepsBindingButNotResultsOrAccess(t *testing.T) {
	f := newLibraryFixture(t)
	f.grant(t, libWorkspaceA, itemPlaces, libFarFuture, `["kafe"]`)
	f.addPlaces(t)
	_, err := f.server.db.Exec(`update reports set is_public=true where id=$1`, libReportA)
	require.NoError(t, err)

	// Workspace B copies the public map. It has no grant of its own.
	ctxB := user.SetWorkspaceCtx(
		context.WithValue(context.Background(), user.ContextKey, &user.Claims{Email: "editor@client-b.example.com"}),
		user.WorkspaceInfo{ID: libWorkspaceB, Name: "Client B", UserRole: proto.UserRole_ROLE_EDITOR})
	res, err := f.server.ForkReport(ctxB, &proto.ForkReportRequest{ReportId: libReportA})
	require.NoError(t, err)

	var copiedDatasetID, copiedQueryID string
	require.NoError(t, f.server.db.QueryRow(`select id, query_id from datasets where report_id=$1 and connection_id=$2`, res.ReportId, libConnection).Scan(&copiedDatasetID, &copiedQueryID))
	_, err = (library.Bindings{}).Get(context.Background(), f.server.db, copiedDatasetID)
	require.NoError(t, err)
	var jobs int
	require.NoError(t, f.server.db.QueryRow(`select count(*) from query_jobs where query_id=$1`, copiedQueryID).Scan(&jobs))
	require.Zero(t, jobs, "results read under another workspace's grant are not copied")

	// A copy of a public map is public too; the test has no public storage, so it is made private to run.
	_, err = f.server.db.Exec(`update reports set is_public=false where id=$1`, res.ReportId)
	require.NoError(t, err)
	before := len(f.jobs.statements())
	_, err = f.server.RunQuery(ctxB, &proto.RunQueryRequest{QueryId: copiedQueryID})
	requireCode(t, err, codes.FailedPrecondition)
	require.Len(t, f.jobs.statements(), before)

	// Once the copy's workspace is granted, it runs.
	f.grant(t, libWorkspaceB, itemPlaces, libFarFuture, `["kafe"]`)
	_, err = f.server.RunQuery(ctxB, &proto.RunQueryRequest{QueryId: copiedQueryID})
	require.NoError(t, err)
	require.Equal(t, placesKafeBandungSQL, f.jobs.statements()[before])
}
