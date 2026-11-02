package dekart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"dekart/src/proto"
	"dekart/src/server/library"
	"dekart/src/server/user"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isLibraryConnection reports whether connectionID is the staff-owned library connection.
func isLibraryConnection(connectionID string) bool {
	libraryConnectionID := library.ConnectionID()
	return libraryConnectionID != "" && connectionID == libraryConnectionID
}

// libraryAccessRefusal is a run refused because the map's workspace holds no active grant for the item.
// The last stored result stays; only a new run is refused.
type libraryAccessRefusal struct{ message string }

func (e libraryAccessRefusal) Error() string { return e.message }

// libraryRunStatus maps an error of libraryQueryText to a gRPC status.
func libraryRunStatus(err error) error {
	var refusal libraryAccessRefusal
	if errors.As(err, &refusal) {
		return status.Error(codes.FailedPrecondition, refusal.message)
	}
	return libraryStatus(err)
}

// libraryQueryText builds the SQL of the library dataset that owns queryID.
//
// It is the one place that turns a binding into SQL: the grant of the map's own workspace is checked first, then
// the statement is built from the catalogue item and the stored selection. Stored and supplied SQL never take part.
// A missing binding or an ended or missing grant returns a [libraryAccessRefusal].
func (s Server) libraryQueryText(ctx context.Context, reportID, queryID string) (string, error) {
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return "", err
	}
	var datasetID string
	err = s.db.QueryRowContext(ctx, `select id from datasets where query_id=$1 and report_id=$2 limit 1`, queryID, reportID).Scan(&datasetID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", libraryAccessRefusal{"library dataset not found"}
	}
	if err != nil {
		return "", err
	}
	binding, err := (library.Bindings{}).Get(ctx, s.db, datasetID)
	if errors.Is(err, library.ErrBindingNotFound) {
		return "", libraryAccessRefusal{"library dataset has no selection"}
	}
	if err != nil {
		return "", err
	}
	workspaceID, err := s.reportWorkspaceID(ctx, reportID)
	if err != nil {
		return "", err
	}
	grant, err := s.libraryGrants().Authorize(ctx, workspaceID, binding.ItemID, time.Now())
	switch {
	case errors.Is(err, library.ErrAccessEnded):
		return "", libraryAccessRefusal{fmt.Sprintf("library access ended %s", grant.EndsOn)}
	case errors.Is(err, library.ErrNotGranted):
		return "", libraryAccessRefusal{"this workspace has no access to the library item"}
	case err != nil:
		return "", err
	}
	item, err := catalogue.GetItem(ctx, binding.ItemID)
	if errors.Is(err, library.ErrItemNotFound) {
		return "", libraryAccessRefusal{"the library item is no longer available"}
	}
	if err != nil {
		return "", err
	}
	statement, err := library.BuildSQL(item, binding.Selection, grant)
	if err != nil {
		// Categories revoked since the selection was stored leave nothing to read.
		return "", libraryAccessRefusal{err.Error()}
	}
	return statement, nil
}

// runLibraryQuery starts the job of a library dataset right after its binding was stored or changed.
// A failure is logged and not returned: the binding is saved, and a refresh runs the dataset again.
func (s Server) runLibraryQuery(ctx context.Context, reportID, queryID string) {
	err := s.startLibraryQuery(ctx, reportID, queryID)
	if err != nil {
		log.Warn().Err(libraryRunStatus(err)).Str("report_id", reportID).Str("query_id", queryID).Msg("Data Library dataset was saved but its first run failed")
	}
}

func (s Server) startLibraryQuery(ctx context.Context, reportID, queryID string) error {
	claims := user.GetClaims(ctx)
	if claims == nil {
		return Unauthenticated
	}
	queryText, err := s.libraryQueryText(ctx, reportID, queryID)
	if err != nil {
		return err
	}
	report, err := s.getReport(ctx, reportID)
	if err != nil {
		return err
	}
	if report == nil {
		return status.Error(codes.NotFound, "report not found")
	}
	connection, err := s.getConnection(ctx, library.ConnectionID())
	if err != nil {
		return err
	}
	if connection == nil {
		return status.Error(codes.Unavailable, "the Data Library is not available")
	}
	_, err = s.runQuery(ctx, runQueryOptions{
		reportID:       reportID,
		queryID:        queryID,
		queryText:      queryText,
		connection:     connection,
		userBucketName: s.getBucketNameFromConnection(connection),
		isPublic:       report.IsPublic,
		changedBy:      claims.Email,
	})
	return err
}

// decorateLibraryDatasets keeps the library connection and SQL away from the browser.
//
// For every dataset on the library connection it sends the binding in place of them: the dataset's `library` field
// is set, and its connection id and the SQL of its query and jobs are blanked. The binding says whether the map's
// workspace still holds an active grant, so the client can show that access ended while the last result stays.
func (s Server) decorateLibraryDatasets(ctx context.Context, reportID string, datasets []*proto.Dataset, queries []*proto.Query, queryJobs []*proto.QueryJob) error {
	libraryQueryIDs := map[string]bool{}
	var workspaceID string
	var workspaceLoaded bool
	for _, dataset := range datasets {
		if !isLibraryConnection(dataset.ConnectionId) {
			continue
		}
		dataset.ConnectionId = ""
		if dataset.QueryId != "" {
			libraryQueryIDs[dataset.QueryId] = true
		}
		binding, err := (library.Bindings{}).Get(ctx, s.db, dataset.Id)
		if errors.Is(err, library.ErrBindingNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if !workspaceLoaded {
			var workspace sql.NullString
			if err := s.db.QueryRowContext(ctx, `select workspace_id from reports where id=$1`, reportID).Scan(&workspace); err != nil {
				return err
			}
			workspaceID, workspaceLoaded = workspace.String, true
		}
		// A map without a grant (never granted, for example a copy in another workspace) reads as ended.
		grant, found, err := s.libraryGrants().Latest(ctx, workspaceID, binding.ItemID)
		if err != nil {
			return err
		}
		dataset.Library = &proto.LibraryDatasetBinding{
			ItemId:      binding.ItemID,
			Selection:   selectionToProto(binding.Selection),
			Version:     binding.Version,
			AccessEnded: !found || grant.StateAt(time.Now()) != library.StateActive,
		}
		if found {
			dataset.Library.AccessEndsAt = unixOrZero(grant.AccessEndsAt())
		}
	}
	for _, query := range queries {
		if libraryQueryIDs[query.Id] {
			query.QueryText = ""
		}
	}
	for _, queryJob := range queryJobs {
		if libraryQueryIDs[queryJob.QueryId] {
			queryJob.QueryText = ""
		}
	}
	return nil
}

// libraryQueriesOfOtherWorkspace returns the query ids of the library datasets among datasets when a copy is made
// in a workspace other than the one that owns the source report. A source in no workspace counts as another one.
func (s Server) libraryQueriesOfOtherWorkspace(ctx context.Context, sourceReportID string, datasets []*proto.Dataset, targetWorkspaceID string) (map[string]bool, error) {
	queryIDs := map[string]bool{}
	var sourceWorkspace sql.NullString
	var loaded bool
	for _, dataset := range datasets {
		if !isLibraryConnection(dataset.ConnectionId) || dataset.QueryId == "" {
			continue
		}
		if !loaded {
			if err := s.db.QueryRowContext(ctx, `select workspace_id from reports where id=$1`, sourceReportID).Scan(&sourceWorkspace); err != nil {
				return nil, err
			}
			loaded = true
		}
		if !sourceWorkspace.Valid || sourceWorkspace.String != targetWorkspaceID {
			queryIDs[dataset.QueryId] = true
		}
	}
	return queryIDs, nil
}
