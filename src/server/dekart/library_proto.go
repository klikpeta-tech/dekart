package dekart

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"dekart/src/proto"
	"dekart/src/server/dark/libraryquery"
	"dekart/src/server/errtype"
	"dekart/src/server/library"
	"dekart/src/server/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SetLibraryCatalogue wires the Data Library catalogue reader. Until it is set, every library command answers Unavailable.
func (s *Server) SetLibraryCatalogue(catalogue library.Catalogue) {
	s.libraryCatalogue = catalogue
}

// refuseLibraryConnection refuses the staff-owned library connection for anything but the library's own commands:
// user-chosen SQL must never run on it, because it can read every library table.
func refuseLibraryConnection(connectionID string) error {
	libraryConnectionID := library.ConnectionID()
	if libraryConnectionID != "" && connectionID == libraryConnectionID {
		return status.Error(codes.PermissionDenied, "this connection is reserved for the Data Library")
	}
	return nil
}

// libraryCatalogueOrUnavailable returns the catalogue reader, or Unavailable when the library is not wired in.
func (s Server) libraryCatalogueOrUnavailable() (library.Catalogue, error) {
	if s.libraryCatalogue == nil {
		return nil, status.Error(codes.Unavailable, "the Data Library is not available")
	}
	return s.libraryCatalogue, nil
}

func (s Server) libraryGrants() library.Grants {
	return library.Grants{DB: s.db}
}

// libraryStatus maps a library domain error to a gRPC status. Unknown errors are logged and answered as Internal.
func libraryStatus(err error) error {
	switch {
	case err == nil:
		return nil
	case status.Code(err) != codes.Unknown:
		return err
	// Same answer for an unknown item and one the workspace is not granted, so ids cannot be probed.
	case errors.Is(err, library.ErrItemNotFound), errors.Is(err, library.ErrNotGranted), errors.Is(err, library.ErrBindingNotFound):
		return status.Error(codes.NotFound, "library item not found")
	case errors.Is(err, library.ErrAccessEnded):
		return status.Error(codes.FailedPrecondition, "library access has ended")
	case errors.Is(err, library.ErrMaxRowsExceeded):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, library.ErrCategoryNotGranted):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, library.ErrVersionConflict):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, library.ErrInvalidEndDate), errors.Is(err, library.ErrEndDateInPast),
		errors.Is(err, library.ErrCategoriesRequired), errors.Is(err, library.ErrCategoriesUnexpected), errors.Is(err, library.ErrUnknownCategory),
		errors.Is(err, libraryquery.ErrInvalidIdentifier), errors.Is(err, libraryquery.ErrNoColumns), errors.Is(err, libraryquery.ErrInvalidArea),
		errors.Is(err, libraryquery.ErrAreaLevelUnsupported), errors.Is(err, libraryquery.ErrInvalidID), errors.Is(err, libraryquery.ErrCategoryRequired),
		errors.Is(err, libraryquery.ErrNoAllowedCategory), errors.Is(err, libraryquery.ErrCategoriesUnsupported),
		errors.Is(err, libraryquery.ErrBrandsUnsupported), errors.Is(err, libraryquery.ErrTooManyValues), errors.Is(err, libraryquery.ErrInvalidLimit):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		errtype.LogError(err, "Data Library command failed")
		return status.Error(codes.Internal, "internal error")
	}
}

// libraryReadStatus is libraryStatus for commands that read the library: an item whose grant ended has left
// the library, so it answers like an item the workspace never had. Commands on a map keep the "ended" answer.
func libraryReadStatus(err error) error {
	if errors.Is(err, library.ErrAccessEnded) {
		return libraryStatus(library.ErrNotGranted)
	}
	return libraryStatus(err)
}

// libraryCaller returns the signed-in user's email and workspace id for a client command.
func libraryCaller(ctx context.Context) (email string, workspaceID string, err error) {
	claims := user.GetClaims(ctx)
	if claims == nil {
		return "", "", Unauthenticated
	}
	workspaceID = checkWorkspace(ctx).ID
	if workspaceID == "" {
		return "", "", status.Error(codes.FailedPrecondition, "no workspace selected")
	}
	return claims.Email, workspaceID, nil
}

// reportWorkspaceID returns the workspace that owns a report; grants are always checked against it, not the caller's workspace.
func (s Server) reportWorkspaceID(ctx context.Context, reportID string) (string, error) {
	var workspaceID sql.NullString
	if err := s.db.QueryRowContext(ctx, `select workspace_id from reports where id=$1`, reportID).Scan(&workspaceID); err != nil {
		return "", err
	}
	if !workspaceID.Valid || workspaceID.String == "" {
		return "", status.Error(codes.FailedPrecondition, "the Data Library needs a map in a workspace")
	}
	return workspaceID.String, nil
}

func selectionFromProto(selection *proto.LibrarySelection) libraryquery.Selection {
	return libraryquery.Selection{
		Area:       selection.GetAreaCode(),
		Categories: selection.GetCategoryIds(),
		Brands:     selection.GetBrandIds(),
	}
}

func selectionToProto(selection libraryquery.Selection) *proto.LibrarySelection {
	return &proto.LibrarySelection{
		AreaCode:    selection.Area,
		CategoryIds: nonNil(selection.Categories),
		BrandIds:    nonNil(selection.Brands),
	}
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func shapeKindToProto(kind string) proto.LibraryShape_Kind {
	switch kind {
	case "points":
		return proto.LibraryShape_KIND_POINTS
	case "lines":
		return proto.LibraryShape_KIND_LINES
	case "areas":
		return proto.LibraryShape_KIND_AREAS
	case "h3":
		return proto.LibraryShape_KIND_H3
	}
	return proto.LibraryShape_KIND_UNSPECIFIED
}

// itemToProto maps a catalogue item; grant is nil when the item is shown without one (admin rows of an ungranted item).
func itemToProto(item library.Item, grant *library.Grant) *proto.LibraryItem {
	result := &proto.LibraryItem{
		Id:              item.ID,
		Title:           item.Title,
		DescriptionMd:   item.DescriptionMD,
		Category:        item.Category,
		Provider:        item.Provider,
		Attribution:     item.Attribution,
		CoverageSummary: item.CoverageSummary,
		Shape: &proto.LibraryShape{
			Kind:         shapeKindToProto(item.Shape.Kind),
			AdminLevel:   item.Shape.AdminLevel,
			H3Resolution: int32(item.Shape.H3Resolution),
		},
		SourceUpdatedAt: unixOrZero(item.SourceUpdatedAt),
		Cadence:         item.Cadence,
		Columns:         nonNil(item.Columns),
		HasTaxonomy:     item.HasTaxonomy,
		MaxRows:         item.MaxRows,
		LayerStyleJson:  item.LayerStyleJSON,
		AreaLevels:      nonNil(item.AreaLevels),
	}
	// The item total would count categories the workspace is not granted, so taxonomy items leave it out.
	if !item.HasTaxonomy {
		result.RowCount = item.RowCount
	}
	if grant != nil {
		result.AccessEndsAt = unixOrZero(grant.AccessEndsAt())
		result.GrantedCategoryCount = int32(len(grant.CategoryIDs))
	}
	return result
}

func grantToProto(itemID string, grant *library.Grant, now time.Time) *proto.LibraryGrant {
	if grant == nil {
		return &proto.LibraryGrant{ItemId: itemID, State: proto.LibraryGrant_STATE_NOT_GRANTED, CategoryIds: []string{}}
	}
	state := proto.LibraryGrant_STATE_ACTIVE
	switch grant.StateAt(now) {
	case library.StateEnded:
		state = proto.LibraryGrant_STATE_ENDED
	case library.StateRevoked:
		state = proto.LibraryGrant_STATE_REVOKED
	}
	return &proto.LibraryGrant{
		ItemId:       itemID,
		State:        state,
		EndsOn:       grant.EndsOn,
		AccessEndsAt: unixOrZero(grant.AccessEndsAt()),
		CategoryIds:  nonNil(grant.CategoryIDs),
		AuthoredBy:   grant.AuthoredBy,
	}
}

// taxonomyToProto maps a taxonomy. With onlyCategories set, it keeps those categories, their groups and their brands.
func taxonomyToProto(taxonomy library.Taxonomy, onlyCategories []string) *proto.LibraryTaxonomy {
	keep := func(string) bool { return true }
	if onlyCategories != nil {
		allowed := make(map[string]struct{}, len(onlyCategories))
		for _, id := range onlyCategories {
			allowed[id] = struct{}{}
		}
		keep = func(id string) bool { _, ok := allowed[id]; return ok }
	}
	result := &proto.LibraryTaxonomy{}
	usedGroups := map[string]struct{}{}
	for _, category := range taxonomy.Categories {
		if !keep(category.ID) {
			continue
		}
		usedGroups[category.GroupID] = struct{}{}
		result.Categories = append(result.Categories, &proto.LibraryCategory{
			Id: category.ID, GroupId: category.GroupID, Title: category.Title, SortOrder: int32(category.SortOrder), RowCount: category.RowCount,
		})
	}
	for _, group := range taxonomy.Groups {
		if _, ok := usedGroups[group.ID]; ok {
			result.Groups = append(result.Groups, &proto.LibraryCategoryGroup{Id: group.ID, Title: group.Title, SortOrder: int32(group.SortOrder)})
		}
	}
	for _, brand := range taxonomy.Brands {
		if keep(brand.CategoryID) {
			result.Brands = append(result.Brands, &proto.LibraryBrand{Id: brand.ID, CategoryId: brand.CategoryID, Title: brand.Title})
		}
	}
	return result
}
