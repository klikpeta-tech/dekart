package dekart

import (
	"context"
	"database/sql"
	"dekart/src/proto"
	"dekart/src/server/conn"
	"dekart/src/server/user"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	guardWorkspaceA   = "aaaaaaaa-1111-0000-0000-000000000001"
	guardWorkspaceB   = "bbbbbbbb-1111-0000-0000-000000000002"
	guardReportA      = "aaaaaaaa-2222-0000-0000-000000000001"
	guardDatasetA     = "aaaaaaaa-2222-0000-0000-000000000002"
	guardConnectionA  = "aaaaaaaa-3333-0000-0000-000000000001"
	guardConnectionB  = "bbbbbbbb-3333-0000-0000-000000000002"
	guardEditorEmail  = "editor@client-a.example.com"
	guardMissingConID = "cccccccc-3333-0000-0000-000000000003"
)

// newConnectionGuardFixture builds a real SQLite database (all production migrations) with two workspaces:
// workspace A (a client) with an editor, a report and an empty dataset, and workspace B (staff) that owns a Postgres connection.
func newConnectionGuardFixture(t *testing.T) (Server, context.Context) {
	t.Helper()
	t.Setenv("DEKART_CLOUD", "")
	clearMetadataEnv(t)
	t.Setenv("DEKART_LICENSE_KEY", "")
	t.Setenv("DEKART_SQLITE_DB_PATH", "")

	db := newMigratedSqliteDB(t)
	exec := func(query string, args ...any) {
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	exec(`insert into workspaces (id, name) values ($1, 'Client A'), ($2, 'Staff B')`, guardWorkspaceA, guardWorkspaceB)
	exec(`insert into reports (id, author_email, workspace_id, title) values ($1, $2, $3, 'Map A')`,
		guardReportA, guardEditorEmail, guardWorkspaceA)
	exec(`insert into datasets (id, report_id, name) values ($1, $2, 'Dataset A')`, guardDatasetA, guardReportA)
	insertConnection := func(id, workspaceID, name string) {
		exec(`insert into connections (id, connection_name, workspace_id, connection_type, postgres_host, postgres_username, postgres_database, postgres_port)
			values ($1, $2, $3, $4, 'library.example.internal', 'lib_owner', 'library', 5432)`,
			id, name, workspaceID, int(proto.ConnectionType_CONNECTION_TYPE_POSTGRES))
	}
	insertConnection(guardConnectionA, guardWorkspaceA, "Client A warehouse")
	insertConnection(guardConnectionB, guardWorkspaceB, "Staff library")

	server := NewServer(db, nil, nil)
	ctx := context.WithValue(context.Background(), user.ContextKey, &user.Claims{Email: guardEditorEmail})
	ctx = user.SetWorkspaceCtx(ctx, user.WorkspaceInfo{
		ID:       guardWorkspaceA,
		Name:     "Client A",
		UserRole: proto.UserRole_ROLE_EDITOR,
	})
	return *server, ctx
}

func datasetConnectionID(t *testing.T, db *sql.DB, datasetID string) sql.NullString {
	t.Helper()
	var connectionID sql.NullString
	require.NoError(t, db.QueryRow(`select connection_id from datasets where id=$1`, datasetID).Scan(&connectionID))
	return connectionID
}

func requireCrossWorkspaceRefusal(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(err), "got: %v", err)
}

// Probe 1 of the ENG-2098 technical plan: an editor in workspace A binds a dataset of A to workspace B's connection.
func TestCreateQuery_RefusesConnectionOfAnotherWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	_, err := server.CreateQuery(ctx, &proto.CreateQueryRequest{
		DatasetId:       guardDatasetA,
		ConnectionId:    guardConnectionB,
		ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
	})

	requireCrossWorkspaceRefusal(t, err)
	require.False(t, datasetConnectionID(t, server.db, guardDatasetA).Valid, "dataset must stay unbound")
	var queries int
	require.NoError(t, server.db.QueryRow(`select count(*) from queries`).Scan(&queries))
	require.Zero(t, queries, "no query row may be written")
}

func TestCreateQuery_RefusesUnknownConnection(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	_, err := server.CreateQuery(ctx, &proto.CreateQueryRequest{
		DatasetId:       guardDatasetA,
		ConnectionId:    guardMissingConID,
		ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
	})

	requireCrossWorkspaceRefusal(t, err)
	require.False(t, datasetConnectionID(t, server.db, guardDatasetA).Valid)
}

func TestCreateQuery_AllowsConnectionOfSameWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	res, err := server.CreateQuery(ctx, &proto.CreateQueryRequest{
		DatasetId:       guardDatasetA,
		ConnectionId:    guardConnectionA,
		ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
	})

	require.NoError(t, err)
	require.NotEmpty(t, res.QueryId)
	require.Equal(t, guardConnectionA, datasetConnectionID(t, server.db, guardDatasetA).String)
}

func TestCreateQuery_AllowsSystemConnection(t *testing.T) {
	for _, connectionID := range []string{conn.SystemConnectionID, "default", ""} {
		t.Run("connection_id="+connectionID, func(t *testing.T) {
			server, ctx := newConnectionGuardFixture(t)

			res, err := server.CreateQuery(ctx, &proto.CreateQueryRequest{
				DatasetId:       guardDatasetA,
				ConnectionId:    connectionID,
				ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
			})

			require.NoError(t, err)
			require.NotEmpty(t, res.QueryId)
		})
	}
}

// The MCP create_query tool shares createQuery with its own validator; the workspace check must still apply.
func TestMCPCreateQuery_RefusesConnectionOfAnotherWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	_, err := server.createQuery(ctx, &proto.CreateQueryRequest{
		DatasetId:       guardDatasetA,
		ConnectionId:    guardConnectionB,
		ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
	}, server.requireMCPCreateQueryConnection)

	requireCrossWorkspaceRefusal(t, err)
	require.False(t, datasetConnectionID(t, server.db, guardDatasetA).Valid)
}

func TestMCPCreateQuery_AnswersMissingConnectionLikeAnotherWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	_, err := server.createQuery(ctx, &proto.CreateQueryRequest{
		DatasetId:       guardDatasetA,
		ConnectionId:    guardMissingConID,
		ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
	}, server.requireMCPCreateQueryConnection)

	requireCrossWorkspaceRefusal(t, err)
}

func TestMCPCreateQuery_AllowsConnectionOfSameWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	_, err := server.createQuery(ctx, &proto.CreateQueryRequest{
		DatasetId:       guardDatasetA,
		ConnectionId:    guardConnectionA,
		ExecutionEngine: proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION,
	}, server.requireMCPCreateQueryConnection)

	// Postgres needs no BigQuery credential, so the only gate left is the workspace check.
	require.NoError(t, err)
	require.Equal(t, guardConnectionA, datasetConnectionID(t, server.db, guardDatasetA).String)
}

func TestCreateFile_RefusesConnectionOfAnotherWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	_, err := server.CreateFile(ctx, &proto.CreateFileRequest{
		DatasetId:    guardDatasetA,
		ConnectionId: guardConnectionB,
	})

	requireCrossWorkspaceRefusal(t, err)
	require.False(t, datasetConnectionID(t, server.db, guardDatasetA).Valid, "dataset must stay unbound")
	var files int
	require.NoError(t, server.db.QueryRow(`select count(*) from files`).Scan(&files))
	require.Zero(t, files, "no file row may be written")
}

func TestCreateFile_AllowsConnectionOfSameWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	res, err := server.CreateFile(ctx, &proto.CreateFileRequest{
		DatasetId:    guardDatasetA,
		ConnectionId: guardConnectionA,
	})

	require.NoError(t, err)
	require.NotEmpty(t, res.GetFileId())
	require.Equal(t, guardConnectionA, datasetConnectionID(t, server.db, guardDatasetA).String)
}

func TestReplaceFile_RefusesConnectionOfAnotherWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)
	created, err := server.CreateFile(ctx, &proto.CreateFileRequest{
		DatasetId:    guardDatasetA,
		ConnectionId: conn.SystemConnectionID,
	})
	require.NoError(t, err)

	_, err = server.ReplaceFile(ctx, &proto.ReplaceFileRequest{
		DatasetId:    guardDatasetA,
		ConnectionId: guardConnectionB,
	})

	requireCrossWorkspaceRefusal(t, err)
	var fileID string
	require.NoError(t, server.db.QueryRow(`select file_id from datasets where id=$1`, guardDatasetA).Scan(&fileID))
	require.Equal(t, created.GetFileId(), fileID, "dataset must keep its file")
	require.NotEqual(t, guardConnectionB, datasetConnectionID(t, server.db, guardDatasetA).String, "dataset must not be rebound")
}

func TestReplaceFile_AllowsConnectionOfSameWorkspace(t *testing.T) {
	server, ctx := newConnectionGuardFixture(t)

	res, err := server.ReplaceFile(ctx, &proto.ReplaceFileRequest{
		DatasetId:    guardDatasetA,
		ConnectionId: guardConnectionA,
	})

	require.NoError(t, err)
	require.NotEmpty(t, res.GetFileId())
	require.Equal(t, guardConnectionA, datasetConnectionID(t, server.db, guardDatasetA).String)
}
