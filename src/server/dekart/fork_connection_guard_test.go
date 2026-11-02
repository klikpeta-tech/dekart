package dekart

import (
	"context"
	"database/sql"
	"dekart/src/proto"
	"dekart/src/server/conn"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	forkSourceReportB  = "bbbbbbbb-4444-0000-0000-000000000001"
	forkSourceDatasetB = "bbbbbbbb-4444-0000-0000-000000000002"
	forkSourceQueryB   = "bbbbbbbb-4444-0000-0000-000000000003"
	forkSourceReportA  = "aaaaaaaa-4444-0000-0000-000000000001"
	forkSourceDatasetA = "aaaaaaaa-4444-0000-0000-000000000002"
	forkSourceQueryA   = "aaaaaaaa-4444-0000-0000-000000000003"
	forkStaffEmail     = "staff@klikpeta.example.com"
	forkQueryText      = "select 1 as marker"
)

// forkedDataset is the stored copy of a source dataset.
type forkedDataset struct {
	connectionID sql.NullString
	queryText    string
}

// seedForkSource inserts a source map with one query dataset bound to connectionID (empty means unbound) into workspaceID.
func seedForkSource(t *testing.T, db *sql.DB, reportID, datasetID, queryID, workspaceID, authorEmail, connectionID string, isPublic bool) {
	t.Helper()
	exec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	exec(`insert into reports (id, author_email, workspace_id, title, is_public) values ($1, $2, $3, 'Source map', $4)`,
		reportID, authorEmail, workspaceID, isPublic)
	exec(`insert into queries (id, query_text) values ($1, $2)`, queryID, forkQueryText)
	var connection any
	if connectionID != "" {
		connection = connectionID
	}
	exec(`insert into datasets (id, report_id, query_id, name, connection_id) values ($1, $2, $3, 'Source dataset', $4)`,
		datasetID, reportID, queryID, connection)
}

func forkAndReadDataset(t *testing.T, server Server, reportID string, ctx context.Context) forkedDataset {
	t.Helper()
	res, err := server.ForkReport(ctx, &proto.ForkReportRequest{ReportId: reportID})
	require.NoError(t, err)
	require.NotEmpty(t, res.ReportId)
	var copied forkedDataset
	require.NoError(t, server.db.QueryRow(`
		select d.connection_id, q.query_text
		from datasets d join queries q on q.id = d.query_id
		where d.report_id = $1`, res.ReportId).Scan(&copied.connectionID, &copied.queryText))
	return copied
}

// Second path of the cross-workspace hole: a map readable from another workspace (here shared by direct access)
// must not carry that workspace's connection into the copy, or a refresh would run with its credentials.
func TestForkReport_BlanksConnectionOfAnotherWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)
	seedForkSource(t, server.db, forkSourceReportB, forkSourceDatasetB, forkSourceQueryB, guardWorkspaceB, forkStaffEmail, guardConnectionB, false)
	// Direct access is honored only for sharable maps, i.e. when every connection has a result cache bucket.
	_, err := server.db.Exec(`update connections set cloud_storage_bucket = 'staff-bucket' where id = $1`, guardConnectionB)
	require.NoError(t, err)
	_, err = server.db.Exec(`insert into report_access_log (report_id, email, access_level, status, authored_by) values ($1, $2, 1, 1, $3)`,
		forkSourceReportB, guardEditorEmail, forkStaffEmail)
	require.NoError(t, err)

	copied := forkAndReadDataset(t, server, forkSourceReportB, ctx)

	require.False(t, copied.connectionID.Valid, "copy must not keep the connection of the source workspace, got %q", copied.connectionID.String)
	require.Equal(t, forkQueryText, copied.queryText, "query text must be kept")
}

// A public map of another workspace is readable by everyone; its copy must not point at that workspace's connection either.
func TestForkReport_PublicMapOfAnotherWorkspaceDoesNotKeepForeignConnection(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)
	seedForkSource(t, server.db, forkSourceReportB, forkSourceDatasetB, forkSourceQueryB, guardWorkspaceB, forkStaffEmail, guardConnectionB, true)

	res, err := server.ForkReport(ctx, &proto.ForkReportRequest{ReportId: forkSourceReportB})

	require.NoError(t, err)
	require.NotEqual(t, guardConnectionB, datasetConnectionIDForReport(t, server.db, res.ReportId).String,
		"copy must not keep the connection of the source workspace")
}

// The existing public-map remap to the copier's own connection of the same type must keep working.
func TestForkReport_PublicMapOfAnotherWorkspaceRemapsToOwnConnection(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)
	seedForkSource(t, server.db, forkSourceReportB, forkSourceDatasetB, forkSourceQueryB, guardWorkspaceB, forkStaffEmail, guardConnectionB, true)

	copied := forkAndReadDataset(t, server, forkSourceReportB, ctx)

	require.Equal(t, guardConnectionA, copied.connectionID.String)
	require.Equal(t, forkQueryText, copied.queryText)
}

// A report the editor authored but that lives in another workspace is also readable and must be treated the same way.
func TestForkReport_AuthoredMapOfAnotherWorkspaceBlanksConnection(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)
	seedForkSource(t, server.db, forkSourceReportB, forkSourceDatasetB, forkSourceQueryB, guardWorkspaceB, guardEditorEmail, guardConnectionB, false)

	copied := forkAndReadDataset(t, server, forkSourceReportB, ctx)

	require.False(t, copied.connectionID.Valid, "copy must not keep the connection of the source workspace, got %q", copied.connectionID.String)
	require.Equal(t, forkQueryText, copied.queryText)
}

func TestForkReport_KeepsConnectionOfSameWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)
	seedForkSource(t, server.db, forkSourceReportA, forkSourceDatasetA, forkSourceQueryA, guardWorkspaceA, guardEditorEmail, guardConnectionA, false)

	copied := forkAndReadDataset(t, server, forkSourceReportA, ctx)

	require.Equal(t, guardConnectionA, copied.connectionID.String)
	require.Equal(t, forkQueryText, copied.queryText)
}

func TestForkReport_KeepsSystemConnection(t *testing.T) {
	for _, connectionID := range []string{"", conn.SystemConnectionID} {
		t.Run("connection_id="+connectionID, func(t *testing.T) {
			server, ctx := newConnectionGuardFixture(t)
			seedForkSource(t, server.db, forkSourceReportA, forkSourceDatasetA, forkSourceQueryA, guardWorkspaceA, guardEditorEmail, connectionID, false)

			copied := forkAndReadDataset(t, server, forkSourceReportA, ctx)

			require.Equal(t, connectionID, copied.connectionID.String)
			require.Equal(t, forkQueryText, copied.queryText)
		})
	}
}

func datasetConnectionIDForReport(t *testing.T, db *sql.DB, reportID string) sql.NullString {
	t.Helper()
	var connectionID sql.NullString
	require.NoError(t, db.QueryRow(`select connection_id from datasets where report_id=$1`, reportID).Scan(&connectionID))
	return connectionID
}
