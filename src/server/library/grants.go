package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dekart/src/server/dark/libraryquery"
)

// Grant statuses as stored in library_grant_log.
const (
	statusGranted = 1
	statusRevoked = 2
)

// State is where a workspace stands for one item.
type State int

// The states a workspace can be in for one item.
const (
	// StateNotGranted means the workspace never had access.
	StateNotGranted State = iota
	// StateActive means access is granted and its end date has not passed.
	StateActive
	// StateEnded means the end date passed.
	StateEnded
	// StateRevoked means an admin revoked the grant.
	StateRevoked
)

var (
	// ErrNotGranted is returned when a workspace has no active grant for an item.
	ErrNotGranted = errors.New("library: item is not granted to this workspace")
	// ErrAccessEnded is returned when a workspace's grant ended or was revoked.
	ErrAccessEnded = errors.New("library: access to this item has ended")
	// ErrEndDateInPast is returned when a grant would already be over.
	ErrEndDateInPast = errors.New("library: end date is in the past")
	// ErrCategoriesRequired is returned when an item with a taxonomy would be granted without categories.
	ErrCategoriesRequired = errors.New("library: at least one category is required")
	// ErrCategoriesUnexpected is returned when categories are given for an item without a taxonomy.
	ErrCategoriesUnexpected = errors.New("library: item has no categories")
	// ErrUnknownCategory is returned when a category id is not in the item's taxonomy.
	ErrUnknownCategory = errors.New("library: unknown category")
)

// Grant is the latest recorded decision for a workspace and an item.
type Grant struct {
	WorkspaceID string
	ItemID      string
	// Revoked is true when the latest entry is a revocation.
	Revoked bool
	// EndsOn is the last day of access, YYYY-MM-DD in Asia/Jakarta.
	EndsOn      string
	CategoryIDs []string
	AuthoredBy  string
}

// StateAt returns the grant's state at now.
func (g Grant) StateAt(now time.Time) State {
	if g.Revoked {
		return StateRevoked
	}
	open, err := AccessOpenAt(g.EndsOn, now)
	if err != nil || !open {
		return StateEnded
	}
	return StateActive
}

// AccessEndsAt returns the last second of access, or the zero time when the grant has no valid end date.
func (g Grant) AccessEndsAt() time.Time {
	end, err := AccessEndsAt(g.EndsOn)
	if err != nil {
		return time.Time{}
	}
	return end
}

// Grants stores grant decisions.
type Grants struct {
	DB *sql.DB
}

// scanGrant reads one grant row.
func scanGrant(row interface{ Scan(...any) error }) (Grant, error) {
	var grant Grant
	var statusValue int
	var categories string
	if err := row.Scan(&grant.WorkspaceID, &grant.ItemID, &statusValue, &grant.EndsOn, &categories, &grant.AuthoredBy); err != nil {
		return Grant{}, err
	}
	grant.Revoked = statusValue == statusRevoked
	if err := json.Unmarshal([]byte(categories), &grant.CategoryIDs); err != nil {
		return Grant{}, fmt.Errorf("library: stored categories of %s: %w", grant.ItemID, err)
	}
	return grant, nil
}

const grantColumns = `workspace_id, item_id, status, ends_on, category_ids, authored_by`

// queryer is what both *sql.DB and *sql.Tx offer.
type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// latest returns the most recent entry for a workspace and an item. The highest seq wins, never the timestamp.
func latest(ctx context.Context, q queryer, workspaceID, itemID string) (Grant, bool, error) {
	row := q.QueryRowContext(ctx,
		`select `+grantColumns+` from library_grant_log
		where workspace_id=$1 and item_id=$2
		order by seq desc limit 1`, workspaceID, itemID)
	grant, err := scanGrant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Grant{}, false, nil
	}
	if err != nil {
		return Grant{}, false, err
	}
	return grant, true, nil
}

// Latest returns the latest decision for a workspace and an item; ok is false when there is none.
func (g Grants) Latest(ctx context.Context, workspaceID, itemID string) (Grant, bool, error) {
	return latest(ctx, g.DB, workspaceID, itemID)
}

// ForWorkspace returns the latest decision for every item a workspace has ever been given or refused.
func (g Grants) ForWorkspace(ctx context.Context, workspaceID string) ([]Grant, error) {
	rows, err := g.DB.QueryContext(ctx,
		`select `+grantColumns+` from library_grant_log latest_entry
		where workspace_id=$1
		and not exists (
			select 1 from library_grant_log newer
			where newer.workspace_id = latest_entry.workspace_id and newer.item_id = latest_entry.item_id and newer.seq > latest_entry.seq
		)
		order by item_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Grant
	for rows.Next() {
		grant, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, grant)
	}
	return result, rows.Err()
}

// Authorize returns the workspace's grant for an item when it is active at now.
// It returns ErrNotGranted when the workspace never had the item and ErrAccessEnded when the grant
// was revoked or its end date passed; in that case the grant is returned too, so callers can show the date.
func (g Grants) Authorize(ctx context.Context, workspaceID, itemID string, now time.Time) (Grant, error) {
	grant, ok, err := g.Latest(ctx, workspaceID, itemID)
	if err != nil {
		return Grant{}, err
	}
	if !ok {
		return Grant{}, ErrNotGranted
	}
	if grant.StateAt(now) != StateActive {
		return grant, ErrAccessEnded
	}
	return grant, nil
}

// normalizeCategories checks category ids against the item and returns them sorted and deduplicated.
// An item with a taxonomy needs at least one category from the taxonomy; an item without one takes none.
func normalizeCategories(item Item, taxonomy Taxonomy, ids []string) ([]string, error) {
	if !item.HasTaxonomy {
		if len(ids) > 0 {
			return nil, ErrCategoriesUnexpected
		}
		return []string{}, nil
	}
	known := make(map[string]struct{}, len(taxonomy.Categories))
	for _, category := range taxonomy.Categories {
		known[category.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := known[id]; !ok || !libraryquery.ValidID(id) {
			return nil, ErrUnknownCategory
		}
	}
	// Intersect sorts and deduplicates; ids are all known and valid here, so nothing is dropped.
	normalized := libraryquery.Intersect(ids, ids)
	if len(normalized) == 0 {
		return nil, ErrCategoriesRequired
	}
	return normalized, nil
}

// append adds one entry to the log and returns it as the new latest grant.
func appendEntry(ctx context.Context, q queryer, grant Grant, statusValue int) (Grant, error) {
	categories, err := json.Marshal(grant.CategoryIDs)
	if err != nil {
		return Grant{}, err
	}
	if _, err := q.ExecContext(ctx,
		`insert into library_grant_log (workspace_id, item_id, status, ends_on, category_ids, authored_by)
		values ($1, $2, $3, $4, $5, $6)`,
		grant.WorkspaceID, grant.ItemID, statusValue, grant.EndsOn, string(categories), grant.AuthoredBy); err != nil {
		return Grant{}, err
	}
	grant.Revoked = statusValue == statusRevoked
	return grant, nil
}

// inTx runs fn in a transaction.
func (g Grants) inTx(ctx context.Context, fn func(q queryer) (Grant, error)) (Grant, error) {
	tx, err := g.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return Grant{}, err
	}
	defer tx.Rollback()
	grant, err := fn(tx)
	if err != nil {
		return Grant{}, err
	}
	if err := tx.Commit(); err != nil {
		return Grant{}, err
	}
	return grant, nil
}

// Set grants an item to a workspace until endsOn, replacing any earlier decision, including a revocation.
//
// endsOn must be a real day in YYYY-MM-DD form and not before today in Asia/Jakarta. For an item with a taxonomy,
// categoryIDs must hold at least one category of the taxonomy; for an item without, it must be empty.
func (g Grants) Set(ctx context.Context, now time.Time, workspaceID string, item Item, taxonomy Taxonomy, endsOn string, categoryIDs []string, author string) (Grant, error) {
	open, err := AccessOpenAt(endsOn, now)
	if err != nil {
		return Grant{}, err
	}
	if !open {
		return Grant{}, ErrEndDateInPast
	}
	categories, err := normalizeCategories(item, taxonomy, categoryIDs)
	if err != nil {
		return Grant{}, err
	}
	return g.inTx(ctx, func(q queryer) (Grant, error) {
		return appendEntry(ctx, q, Grant{WorkspaceID: workspaceID, ItemID: item.ID, EndsOn: endsOn, CategoryIDs: categories, AuthoredBy: author}, statusGranted)
	})
}

// Revoke ends a workspace's access to an item now, keeping its end date and categories on record.
// It returns ErrNotGranted when the workspace never had the item.
func (g Grants) Revoke(ctx context.Context, workspaceID, itemID, author string) (Grant, error) {
	return g.inTx(ctx, func(q queryer) (Grant, error) {
		current, ok, err := latest(ctx, q, workspaceID, itemID)
		if err != nil {
			return Grant{}, err
		}
		if !ok {
			return Grant{}, ErrNotGranted
		}
		current.AuthoredBy = author
		return appendEntry(ctx, q, current, statusRevoked)
	})
}

// SetCategories replaces the granted categories of an item, keeping its end date.
// It returns ErrNotGranted when the grant is missing or revoked. categoryIDs follows the rules of Set.
func (g Grants) SetCategories(ctx context.Context, workspaceID string, item Item, taxonomy Taxonomy, categoryIDs []string, author string) (Grant, error) {
	categories, err := normalizeCategories(item, taxonomy, categoryIDs)
	if err != nil {
		return Grant{}, err
	}
	return g.inTx(ctx, func(q queryer) (Grant, error) {
		current, ok, err := latest(ctx, q, workspaceID, item.ID)
		if err != nil {
			return Grant{}, err
		}
		if !ok || current.Revoked {
			return Grant{}, ErrNotGranted
		}
		current.CategoryIDs = categories
		current.AuthoredBy = author
		return appendEntry(ctx, q, current, statusGranted)
	})
}
