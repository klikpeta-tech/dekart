package dekart

import (
	"context"
	"database/sql"
	"time"

	"dekart/src/proto"
	"dekart/src/server/dark/libraryquery"
	"dekart/src/server/library"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// libraryItemAccess loads an item and the active grant of a workspace for it.
// An unknown item, a missing grant and an ended grant all fail the same way, as libraryStatus maps them.
func (s Server) libraryItemAccess(ctx context.Context, catalogue library.Catalogue, workspaceID, itemID string) (library.Item, library.Grant, error) {
	item, err := catalogue.GetItem(ctx, itemID)
	if err != nil {
		return library.Item{}, library.Grant{}, err
	}
	grant, err := s.libraryGrants().Authorize(ctx, workspaceID, item.ID, time.Now())
	if err != nil {
		return library.Item{}, library.Grant{}, err
	}
	return item, grant, nil
}

// ListLibraryItems returns the catalogue items granted to the current workspace.
func (s Server) ListLibraryItems(ctx context.Context, req *proto.ListLibraryItemsRequest) (*proto.ListLibraryItemsResponse, error) {
	_, workspaceID, err := libraryCaller(ctx)
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	items, err := catalogue.ListItems(ctx)
	if err != nil {
		return nil, libraryStatus(err)
	}
	grants, err := s.libraryGrants().ForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, libraryStatus(err)
	}
	now := time.Now()
	active := map[string]library.Grant{}
	for _, grant := range grants {
		if grant.StateAt(now) == library.StateActive {
			active[grant.ItemID] = grant
		}
	}
	response := &proto.ListLibraryItemsResponse{CatalogueSize: int32(len(items))}
	for _, item := range items {
		if grant, ok := active[item.ID]; ok {
			grant := grant
			response.Items = append(response.Items, itemToProto(item, &grant))
		}
	}
	return response, nil
}

// GetLibraryItem returns one granted item with the part of its taxonomy that the workspace is granted.
func (s Server) GetLibraryItem(ctx context.Context, req *proto.GetLibraryItemRequest) (*proto.GetLibraryItemResponse, error) {
	_, workspaceID, err := libraryCaller(ctx)
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	item, grant, err := s.libraryItemAccess(ctx, catalogue, workspaceID, req.GetItemId())
	if err != nil {
		return nil, libraryReadStatus(err)
	}
	response := &proto.GetLibraryItemResponse{Item: itemToProto(item, &grant)}
	if item.HasTaxonomy {
		taxonomy, err := catalogue.Taxonomy(ctx, item.ID)
		if err != nil {
			return nil, libraryStatus(err)
		}
		response.Taxonomy = taxonomyToProto(taxonomy, grant.CategoryIDs)
	}
	return response, nil
}

// ListLibraryAreas lists the areas under a parent code. With an item it also returns the item's row count per area,
// which needs an active grant for that item.
func (s Server) ListLibraryAreas(ctx context.Context, req *proto.ListLibraryAreasRequest) (*proto.ListLibraryAreasResponse, error) {
	_, workspaceID, err := libraryCaller(ctx)
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	if !libraryquery.ValidArea(req.GetParentCode()) {
		return nil, libraryStatus(libraryquery.ErrInvalidArea)
	}
	if req.GetItemId() != "" {
		if _, _, err := s.libraryItemAccess(ctx, catalogue, workspaceID, req.GetItemId()); err != nil {
			return nil, libraryReadStatus(err)
		}
	}
	areas, err := catalogue.ListAreas(ctx, req.GetParentCode(), req.GetItemId())
	if err != nil {
		return nil, libraryStatus(err)
	}
	response := &proto.ListLibraryAreasResponse{}
	for _, area := range areas {
		response.Areas = append(response.Areas, &proto.LibraryArea{
			Code: area.Code, Level: area.Level, Name: area.Name, ParentCode: area.ParentCode, RowCount: area.RowCount,
		})
	}
	return response, nil
}

// validatedSelection loads the item, the workspace's active grant and a checked selection for a pick that is not yet stored.
func (s Server) validatedSelection(ctx context.Context, catalogue library.Catalogue, workspaceID, itemID string, selection *proto.LibrarySelection) (library.Item, library.Grant, libraryquery.Selection, error) {
	item, grant, err := s.libraryItemAccess(ctx, catalogue, workspaceID, itemID)
	if err != nil {
		return library.Item{}, library.Grant{}, libraryquery.Selection{}, err
	}
	picked := selectionFromProto(selection)
	if err := library.ValidateSelection(item, grant, picked); err != nil {
		return library.Item{}, library.Grant{}, libraryquery.Selection{}, err
	}
	return item, grant, picked, nil
}

// CountLibrarySelection estimates the size of a selection and its counts per category and brand.
func (s Server) CountLibrarySelection(ctx context.Context, req *proto.CountLibrarySelectionRequest) (*proto.CountLibrarySelectionResponse, error) {
	_, workspaceID, err := libraryCaller(ctx)
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	item, grant, selection, err := s.validatedSelection(ctx, catalogue, workspaceID, req.GetItemId(), req.GetSelection())
	if err != nil {
		return nil, libraryReadStatus(err)
	}
	estimate, err := catalogue.EstimateSelection(ctx, item, selection, grant.CategoryIDs)
	if err != nil {
		return nil, libraryStatus(err)
	}
	response := &proto.CountLibrarySelectionResponse{
		RowCount:       estimate.RowCount,
		ExceedsMaxRows: library.CheckSize(item, estimate) != nil,
	}
	for _, count := range estimate.ByCategory {
		response.ByCategory = append(response.ByCategory, &proto.LibraryCount{Id: count.ID, RowCount: count.RowCount})
	}
	for _, count := range estimate.ByBrand {
		response.ByBrand = append(response.ByBrand, &proto.LibraryCount{Id: count.ID, RowCount: count.RowCount})
	}
	return response, nil
}

// GetLibrarySample returns a few rows of a selection for the preview.
func (s Server) GetLibrarySample(ctx context.Context, req *proto.GetLibrarySampleRequest) (*proto.GetLibrarySampleResponse, error) {
	_, workspaceID, err := libraryCaller(ctx)
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	item, grant, selection, err := s.validatedSelection(ctx, catalogue, workspaceID, req.GetItemId(), req.GetSelection())
	if err != nil {
		return nil, libraryReadStatus(err)
	}
	statement, err := libraryquery.BuildSample(item.Table, selection, grant.CategoryIDs, library.SampleLimit)
	if err != nil {
		return nil, libraryStatus(err)
	}
	sample, err := catalogue.SampleRows(ctx, statement)
	if err != nil {
		return nil, libraryStatus(err)
	}
	response := &proto.GetLibrarySampleResponse{Columns: nonNil(sample.Columns)}
	for _, row := range sample.Rows {
		response.Rows = append(response.Rows, &proto.LibrarySampleRow{Values: row})
	}
	return response, nil
}

// pickedTitles returns the titles used in a tab name: the picked brands if any, else the picked categories.
func pickedTitles(taxonomy library.Taxonomy, selection libraryquery.Selection) []string {
	titles := map[string]string{}
	for _, category := range taxonomy.Categories {
		titles[category.ID] = category.Title
	}
	for _, brand := range taxonomy.Brands {
		titles[brand.ID] = brand.Title
	}
	ids := selection.Categories
	if len(selection.Brands) > 0 {
		ids = selection.Brands
	}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if title, ok := titles[id]; ok {
			result = append(result, title)
		} else {
			result = append(result, id)
		}
	}
	return result
}

// libraryDatasetName builds the tab name of a library dataset from the catalogue's titles.
func (s Server) libraryDatasetName(ctx context.Context, catalogue library.Catalogue, item library.Item, selection libraryquery.Selection) (string, error) {
	var taxonomy library.Taxonomy
	if item.HasTaxonomy {
		var err error
		if taxonomy, err = catalogue.Taxonomy(ctx, item.ID); err != nil {
			return "", err
		}
	}
	areaName := "Indonesia"
	if selection.Area != "" {
		area, err := catalogue.GetArea(ctx, selection.Area)
		if err != nil {
			return "", err
		}
		areaName = area.Name
	}
	return library.DatasetName(item.Title, pickedTitles(taxonomy, selection), areaName), nil
}

// libraryDatasetReport resolves a dataset the caller can write to its report, or answers NotFound.
func (s Server) libraryDatasetReport(ctx context.Context, datasetID string) (string, error) {
	if err := validateUUIDField(datasetID, "dataset_id"); err != nil {
		return "", err
	}
	reportID, err := s.getReportID(ctx, datasetID, true)
	if err != nil {
		return "", status.Error(codes.Internal, err.Error())
	}
	if reportID == nil {
		return "", status.Error(codes.NotFound, "dataset not found or permission not granted")
	}
	if err := s.requireReportWorkspaceWrite(ctx, *reportID); err != nil {
		return "", err
	}
	return *reportID, nil
}

// AddLibraryDataset binds an empty dataset of a map to a library item and a selection.
//
// The checks run in order: sign-in, write access to the map, workspace writable, the map's workspace holds an
// active grant, the selection is valid and within the item's row limit. The query is created on the library
// connection with empty text; its SQL is built by the server whenever it runs.
func (s Server) AddLibraryDataset(ctx context.Context, req *proto.AddLibraryDatasetRequest) (*proto.AddLibraryDatasetResponse, error) {
	email, _, err := libraryCaller(ctx)
	if err != nil {
		return nil, err
	}
	reportID, err := s.libraryDatasetReport(ctx, req.GetDatasetId())
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	connectionID := library.ConnectionID()
	if connectionID == "" {
		return nil, status.Error(codes.Unavailable, "the Data Library is not available")
	}
	workspaceID, err := s.reportWorkspaceID(ctx, reportID)
	if err != nil {
		return nil, libraryStatus(err)
	}
	item, grant, selection, err := s.validatedSelection(ctx, catalogue, workspaceID, req.GetItemId(), req.GetSelection())
	if err != nil {
		return nil, libraryStatus(err)
	}
	estimate, err := catalogue.EstimateSelection(ctx, item, selection, grant.CategoryIDs)
	if err != nil {
		return nil, libraryStatus(err)
	}
	if err := library.CheckSize(item, estimate); err != nil {
		return nil, libraryStatus(err)
	}
	name, err := s.libraryDatasetName(ctx, catalogue, item, selection)
	if err != nil {
		return nil, libraryStatus(err)
	}
	var queryID, fileID sql.NullString
	if err := s.db.QueryRowContext(ctx, `select query_id, file_id from datasets where id=$1`, req.GetDatasetId()).Scan(&queryID, &fileID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if queryID.Valid || fileID.Valid {
		return nil, status.Error(codes.FailedPrecondition, "dataset is not empty")
	}
	binding := library.Binding{DatasetID: req.GetDatasetId(), ReportID: reportID, ItemID: item.ID, Selection: selection, CreatedBy: email}
	newQueryID, _, err := s.createDatasetQueryRecord(ctx, reportID, req.GetDatasetId(), connectionID,
		proto.QueryExecutionEngine_QUERY_EXECUTION_ENGINE_CONNECTION, email,
		func(tx *sql.Tx) error {
			if err := (library.Bindings{}).Insert(ctx, tx, binding); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `update datasets set name=$1 where id=$2`, name, req.GetDatasetId())
			return err
		})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, libraryStatus(err)
	}
	s.reportStreams.Ping(reportID)
	return &proto.AddLibraryDatasetResponse{DatasetId: req.GetDatasetId(), QueryId: newQueryID}, nil
}

// UpdateLibraryDatasetSelection replaces the selection of a library dataset when the client's version is current.
//
// A grant that ended or was revoked refuses the change. The binding version is a compare-and-set token: a stale
// expected_version answers Canceled and the client reconciles from the report stream.
func (s Server) UpdateLibraryDatasetSelection(ctx context.Context, req *proto.UpdateLibraryDatasetSelectionRequest) (*proto.UpdateLibraryDatasetSelectionResponse, error) {
	email, _, err := libraryCaller(ctx)
	if err != nil {
		return nil, err
	}
	reportID, err := s.libraryDatasetReport(ctx, req.GetDatasetId())
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	binding, err := (library.Bindings{}).Get(ctx, s.db, req.GetDatasetId())
	if err != nil {
		return nil, libraryStatus(err)
	}
	workspaceID, err := s.reportWorkspaceID(ctx, reportID)
	if err != nil {
		return nil, libraryStatus(err)
	}
	item, grant, selection, err := s.validatedSelection(ctx, catalogue, workspaceID, binding.ItemID, req.GetSelection())
	if err != nil {
		return nil, libraryStatus(err)
	}
	estimate, err := catalogue.EstimateSelection(ctx, item, selection, grant.CategoryIDs)
	if err != nil {
		return nil, libraryStatus(err)
	}
	if err := library.CheckSize(item, estimate); err != nil {
		return nil, libraryStatus(err)
	}
	name, err := s.libraryDatasetName(ctx, catalogue, item, selection)
	if err != nil {
		return nil, libraryStatus(err)
	}
	version, err := s.commitLibrarySelection(ctx, reportID, req.GetDatasetId(), req.GetExpectedVersion(), selection, name, email)
	if err != nil {
		return nil, libraryStatus(err)
	}
	s.reportStreams.Ping(reportID)
	return &proto.UpdateLibraryDatasetSelectionResponse{DatasetId: req.GetDatasetId(), Version: version}, nil
}

// commitLibrarySelection stores a new selection, the tab name and a report snapshot in one transaction.
func (s Server) commitLibrarySelection(ctx context.Context, reportID, datasetID string, expectedVersion int64, selection libraryquery.Selection, name, changedBy string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := lockReportTx(ctx, tx, reportID); err != nil {
		return 0, err
	}
	version, err := (library.Bindings{}).UpdateSelection(ctx, tx, datasetID, expectedVersion, selection)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `update datasets set name=$1, updated_at=CURRENT_TIMESTAMP where id=$2`, name, datasetID); err != nil {
		return 0, err
	}
	versionID := newUUID()
	if _, err := tx.ExecContext(ctx, `update reports set version_id=$1 where id=$2`, versionID, reportID); err != nil {
		return 0, err
	}
	if err := s.createReportSnapshotWithVersionIDTx(ctx, tx, versionID, reportID, changedBy, proto.ReportSnapshot_TRIGGER_TYPE_QUERY_CHANGE); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return version, nil
}
