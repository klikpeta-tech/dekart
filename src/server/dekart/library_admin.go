package dekart

import (
	"context"
	"time"

	"dekart/src/proto"
	"dekart/src/server/library"
	"dekart/src/server/user"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// requireLibraryAdmin gates the grant commands: the caller must be a workspace Admin and also Klikpeta staff,
// so a client person who is made Admin of their own workspace cannot grant themselves catalogue items.
// It returns the caller's email and the workspace the grants apply to.
func (s Server) requireLibraryAdmin(ctx context.Context) (email string, workspaceID string, err error) {
	claims := user.GetClaims(ctx)
	if claims == nil {
		return "", "", Unauthenticated
	}
	workspaceInfo := checkWorkspace(ctx)
	if workspaceInfo.ID == "" {
		return "", "", status.Error(codes.FailedPrecondition, "no workspace selected")
	}
	if workspaceInfo.UserRole != proto.UserRole_ROLE_ADMIN {
		return "", "", status.Error(codes.PermissionDenied, "only workspace admins can manage Data Library access")
	}
	if !user.IsDefaultWorkspaceAdmin(claims.Email) {
		log.Warn().Str("workspace_id", workspaceInfo.ID).Msg("Data Library grant command refused: caller is not Klikpeta staff")
		return "", "", status.Error(codes.PermissionDenied, "only Klikpeta staff can manage Data Library access")
	}
	return claims.Email, workspaceInfo.ID, nil
}

// requireLibraryGrantWrite is requireLibraryAdmin for commands that change grants, which also need a writable workspace.
func (s Server) requireLibraryGrantWrite(ctx context.Context) (email string, workspaceID string, err error) {
	email, workspaceID, err = s.requireLibraryAdmin(ctx)
	if err != nil {
		return "", "", err
	}
	if err := s.requireWorkspaceIDWrite(ctx, workspaceID); err != nil {
		return "", "", err
	}
	return email, workspaceID, nil
}

// pingWorkspaceReports refreshes the open maps of a workspace so they pick up a changed grant without a reload.
func (s Server) pingWorkspaceReports(ctx context.Context, workspaceID string) {
	rows, err := s.db.QueryContext(ctx, `select id from reports where workspace_id=$1`, workspaceID)
	if err != nil {
		log.Warn().Err(err).Msg("could not list reports to ping after a Data Library grant change")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var reportID string
		if err := rows.Scan(&reportID); err != nil {
			return
		}
		s.reportStreams.Ping(reportID)
	}
}

// ListLibraryGrants returns the whole catalogue with the current workspace's grant state for each item.
func (s Server) ListLibraryGrants(ctx context.Context, req *proto.ListLibraryGrantsRequest) (*proto.ListLibraryGrantsResponse, error) {
	_, workspaceID, err := s.requireLibraryAdmin(ctx)
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
	byItem := map[string]library.Grant{}
	for _, grant := range grants {
		byItem[grant.ItemID] = grant
	}
	now := time.Now()
	response := &proto.ListLibraryGrantsResponse{}
	for _, item := range items {
		var grant *library.Grant
		if found, ok := byItem[item.ID]; ok {
			grant = &found
		}
		response.Rows = append(response.Rows, &proto.LibraryGrantRow{Item: itemToProto(item, grant), Grant: grantToProto(item.ID, grant, now)})
	}
	return response, nil
}

// GetLibraryTaxonomy returns the whole category tree of an item with nationwide counts, for choosing what to grant.
func (s Server) GetLibraryTaxonomy(ctx context.Context, req *proto.GetLibraryTaxonomyRequest) (*proto.GetLibraryTaxonomyResponse, error) {
	if _, _, err := s.requireLibraryAdmin(ctx); err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	item, err := catalogue.GetItem(ctx, req.GetItemId())
	if err != nil {
		return nil, libraryStatus(err)
	}
	taxonomy, err := catalogue.Taxonomy(ctx, item.ID)
	if err != nil {
		return nil, libraryStatus(err)
	}
	return &proto.GetLibraryTaxonomyResponse{Taxonomy: taxonomyToProto(taxonomy, nil)}, nil
}

// SetLibraryGrant grants an item to the current workspace until a required end date, or changes the end date.
func (s Server) SetLibraryGrant(ctx context.Context, req *proto.SetLibraryGrantRequest) (*proto.SetLibraryGrantResponse, error) {
	email, workspaceID, err := s.requireLibraryGrantWrite(ctx)
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	item, err := catalogue.GetItem(ctx, req.GetItemId())
	if err != nil {
		return nil, libraryStatus(err)
	}
	var taxonomy library.Taxonomy
	if item.HasTaxonomy {
		if taxonomy, err = catalogue.Taxonomy(ctx, item.ID); err != nil {
			return nil, libraryStatus(err)
		}
	}
	now := time.Now()
	grant, err := s.libraryGrants().Set(ctx, now, workspaceID, item, taxonomy, req.GetEndsOn(), req.GetCategoryIds(), email)
	if err != nil {
		return nil, libraryStatus(err)
	}
	s.pingWorkspaceReports(ctx, workspaceID)
	return &proto.SetLibraryGrantResponse{Grant: grantToProto(item.ID, &grant, now)}, nil
}

// RevokeLibraryGrant ends the current workspace's access to an item now.
func (s Server) RevokeLibraryGrant(ctx context.Context, req *proto.RevokeLibraryGrantRequest) (*proto.RevokeLibraryGrantResponse, error) {
	email, workspaceID, err := s.requireLibraryGrantWrite(ctx)
	if err != nil {
		return nil, err
	}
	grant, err := s.libraryGrants().Revoke(ctx, workspaceID, req.GetItemId(), email)
	if err != nil {
		return nil, libraryStatus(err)
	}
	s.pingWorkspaceReports(ctx, workspaceID)
	return &proto.RevokeLibraryGrantResponse{Grant: grantToProto(req.GetItemId(), &grant, time.Now())}, nil
}

// SetLibraryGrantCategories replaces the granted categories of an item, keeping its end date.
func (s Server) SetLibraryGrantCategories(ctx context.Context, req *proto.SetLibraryGrantCategoriesRequest) (*proto.SetLibraryGrantCategoriesResponse, error) {
	email, workspaceID, err := s.requireLibraryGrantWrite(ctx)
	if err != nil {
		return nil, err
	}
	catalogue, err := s.libraryCatalogueOrUnavailable()
	if err != nil {
		return nil, err
	}
	item, err := catalogue.GetItem(ctx, req.GetItemId())
	if err != nil {
		return nil, libraryStatus(err)
	}
	var taxonomy library.Taxonomy
	if item.HasTaxonomy {
		if taxonomy, err = catalogue.Taxonomy(ctx, item.ID); err != nil {
			return nil, libraryStatus(err)
		}
	}
	grant, err := s.libraryGrants().SetCategories(ctx, workspaceID, item, taxonomy, req.GetCategoryIds(), email)
	if err != nil {
		return nil, libraryStatus(err)
	}
	s.pingWorkspaceReports(ctx, workspaceID)
	return &proto.SetLibraryGrantCategoriesResponse{Grant: grantToProto(item.ID, &grant, time.Now())}, nil
}
