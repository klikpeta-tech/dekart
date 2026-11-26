package dekart

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"dekart/src/server/user"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

const (
	sharedMapWorkspaceID = "00000000-0000-0000-0000-0000000000b7"
	otherWorkspaceID     = "00000000-0000-0000-0000-0000000000c1"
	sharedMapReportID    = "00000000-0000-0000-0000-0000000000d1"
	sharedMapDatasetID   = "00000000-0000-0000-0000-0000000000e1"
	sharedMapAuthorEmail = "author@example.com"
	sharedMapMemberEmail = "member@example.com"
)

// A member of the map's workspace who joined another workspace later opens a discoverable map:
// the dataset download resolves access in the workspace named by X-Dekart-Workspace-Id, and
// falls back to the newest workspace (where the map is invisible) when the header is missing.
func TestServeDatasetSource_DiscoverableMapNeedsSelectedWorkspaceHeader(t *testing.T) {
	t.Setenv("DEKART_CLOUD", "")
	clearMetadataEnv(t)
	t.Setenv("DEKART_LICENSE_KEY", "")
	t.Setenv("DEKART_SQLITE_DB_PATH", "")

	db := newMigratedSqliteDB(t)
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	exec(`insert into workspaces (id, name) values ($1, 'Bintang7'), ($2, 'Other')`, sharedMapWorkspaceID, otherWorkspaceID)
	for _, workspaceID := range []string{sharedMapWorkspaceID, otherWorkspaceID} {
		exec(`insert into subscription_log (workspace_id, authored_by, plan_type) values ($1, 'system', 5)`, workspaceID)
	}
	insertMember := func(id, workspaceID, email, joinedAt string) {
		exec(`insert into workspace_log (id, workspace_id, email, authored_by, status, role, created_at) values ($1, $2, $3, $3, 1, 2, $4)`,
			id, workspaceID, email, joinedAt)
	}
	insertMember("log-author", sharedMapWorkspaceID, sharedMapAuthorEmail, "2026-01-01 00:00:00")
	insertMember("log-member-b7", sharedMapWorkspaceID, sharedMapMemberEmail, "2026-01-02 00:00:00")
	insertMember("log-member-other", otherWorkspaceID, sharedMapMemberEmail, "2026-01-03 00:00:00") // newest
	exec(`insert into reports (id, author_email, workspace_id, title, discoverable) values ($1, $2, $3, 'Bintang 7', 1)`,
		sharedMapReportID, sharedMapAuthorEmail, sharedMapWorkspaceID)
	exec(`insert into datasets (id, report_id, name) values ($1, $2, 'Dataset')`, sharedMapDatasetID, sharedMapReportID)

	server := NewServer(db, nil, nil)
	// workspaceContext resolves the request context the way the HTTP middleware does for a dataset download.
	workspaceContext := func(workspaceHeader string) (context.Context, *http.Request) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/dataset-source/"+sharedMapDatasetID+"/source.csv", nil)
		if workspaceHeader != "" {
			request.Header.Set("X-Dekart-Workspace-Id", workspaceHeader)
		}
		ctx := context.WithValue(request.Context(), user.ContextKey, &user.Claims{Email: sharedMapMemberEmail})
		ctx = server.SetWorkspaceContext(ctx, request)
		request = mux.SetURLVars(request.WithContext(ctx), map[string]string{"dataset": sharedMapDatasetID, "source": "source", "extension": "csv"})
		return ctx, request
	}

	// Without the header: the member lands in the newest workspace and the download answers 404.
	ctx, request := workspaceContext("")
	require.Equal(t, otherWorkspaceID, user.CheckWorkspaceCtx(ctx).ID)
	recorder := httptest.NewRecorder()
	server.ServeDatasetSource(recorder, request)
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), "dataset not found")

	// With the header: the map's workspace is selected and the dataset passes the access check
	// (ServeDatasetSource then reads storage, which this fixture does not configure).
	ctx, _ = workspaceContext(sharedMapWorkspaceID)
	require.Equal(t, sharedMapWorkspaceID, user.CheckWorkspaceCtx(ctx).ID)
	reportID, err := server.getReportID(ctx, sharedMapDatasetID, false)
	require.NoError(t, err)
	require.NotNil(t, reportID, "the discoverable map is visible in its own workspace")
	require.Equal(t, sharedMapReportID, *reportID)
}
