package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dekart/src/server/dark/libraryquery"
)

var (
	// ErrBindingNotFound is returned when a dataset has no library binding.
	ErrBindingNotFound = errors.New("library: dataset has no library binding")
	// ErrVersionConflict is returned when a compare-and-set update saw a newer binding version.
	ErrVersionConflict = errors.New("library: binding changed since it was read")
	// ErrMaxRowsExceeded is returned when a selection is larger than the item allows on a map.
	ErrMaxRowsExceeded = errors.New("library: selection has too many rows, pick a smaller area")
	// ErrCategoryNotGranted is returned when a selection names a category the workspace is not granted.
	ErrCategoryNotGranted = errors.New("library: category is not granted to this workspace")
)

// Binding ties a map dataset to a catalogue item and a selection.
type Binding struct {
	DatasetID string
	ReportID  string
	ItemID    string
	Selection libraryquery.Selection
	// Version increases by one with every selection change and is the token of compare-and-set updates.
	Version   int64
	CreatedBy string
}

// Bindings stores dataset bindings. Methods take a transaction or database so callers can bind within their own transaction.
type Bindings struct{}

// encodeIDs stores an id list as JSON, using [] for nil.
func encodeIDs(ids []string) (string, error) {
	if ids == nil {
		ids = []string{}
	}
	encoded, err := json.Marshal(ids)
	return string(encoded), err
}

// Insert stores a new binding at version 1. It fails when the dataset already has one.
func (Bindings) Insert(ctx context.Context, q queryer, binding Binding) error {
	categories, err := encodeIDs(binding.Selection.Categories)
	if err != nil {
		return err
	}
	brands, err := encodeIDs(binding.Selection.Brands)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx,
		`insert into library_dataset_bindings (dataset_id, report_id, item_id, area_code, category_ids, brand_ids, created_by)
		values ($1, $2, $3, $4, $5, $6, $7)`,
		binding.DatasetID, binding.ReportID, binding.ItemID, binding.Selection.Area, categories, brands, binding.CreatedBy)
	return err
}

// Get returns the binding of a dataset, or ErrBindingNotFound.
func (Bindings) Get(ctx context.Context, q queryer, datasetID string) (Binding, error) {
	var binding Binding
	var categories, brands string
	err := q.QueryRowContext(ctx,
		`select dataset_id, report_id, item_id, area_code, category_ids, brand_ids, version, created_by
		from library_dataset_bindings where dataset_id=$1`, datasetID).
		Scan(&binding.DatasetID, &binding.ReportID, &binding.ItemID, &binding.Selection.Area, &categories, &brands, &binding.Version, &binding.CreatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return Binding{}, ErrBindingNotFound
	}
	if err != nil {
		return Binding{}, err
	}
	if err := json.Unmarshal([]byte(categories), &binding.Selection.Categories); err != nil {
		return Binding{}, fmt.Errorf("library: stored categories of dataset %s: %w", datasetID, err)
	}
	if err := json.Unmarshal([]byte(brands), &binding.Selection.Brands); err != nil {
		return Binding{}, fmt.Errorf("library: stored brands of dataset %s: %w", datasetID, err)
	}
	return binding, nil
}

// UpdateSelection replaces the selection of a dataset's binding only when its version is still expectedVersion,
// and returns the new version. It returns ErrVersionConflict when the binding moved on and ErrBindingNotFound when
// there is none.
func (b Bindings) UpdateSelection(ctx context.Context, q queryer, datasetID string, expectedVersion int64, selection libraryquery.Selection) (int64, error) {
	categories, err := encodeIDs(selection.Categories)
	if err != nil {
		return 0, err
	}
	brands, err := encodeIDs(selection.Brands)
	if err != nil {
		return 0, err
	}
	result, err := q.ExecContext(ctx,
		`update library_dataset_bindings
		set area_code=$1, category_ids=$2, brand_ids=$3, version=version+1, updated_at=CURRENT_TIMESTAMP
		where dataset_id=$4 and version=$5`,
		selection.Area, categories, brands, datasetID, expectedVersion)
	if err != nil {
		return 0, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if changed == 0 {
		if _, err := b.Get(ctx, q, datasetID); err != nil {
			return 0, err
		}
		return 0, ErrVersionConflict
	}
	return expectedVersion + 1, nil
}

// BuildSQL returns the statement that loads a binding's rows for a grant. The grant must be active: the
// caller checks that with [Grants.Authorize]. Categories that are no longer granted drop out of the filter.
func BuildSQL(item Item, selection libraryquery.Selection, grant Grant) (string, error) {
	return libraryquery.Build(item.Table, selection, grant.CategoryIDs)
}

// ValidateSelection checks a selection against an item and a grant before it is stored: every requested category
// must be granted and the selection must build (area level, id patterns, required categories).
func ValidateSelection(item Item, grant Grant, selection libraryquery.Selection) error {
	if item.HasTaxonomy {
		granted := make(map[string]struct{}, len(grant.CategoryIDs))
		for _, id := range grant.CategoryIDs {
			granted[id] = struct{}{}
		}
		for _, id := range selection.Categories {
			if _, ok := granted[id]; !ok {
				return ErrCategoryNotGranted
			}
		}
	}
	_, err := BuildSQL(item, selection, grant)
	return err
}

// CheckSize returns ErrMaxRowsExceeded when an estimate is larger than the item allows on a map.
func CheckSize(item Item, estimate Estimate) error {
	if item.MaxRows > 0 && estimate.RowCount > item.MaxRows {
		return ErrMaxRowsExceeded
	}
	return nil
}

// DatasetName returns the tab name for a selection: `<Title> · <Category or brand> +n · <Area>`.
// pickedTitles are the titles of the picked categories or brands in display order, areaName the picked area.
func DatasetName(title string, pickedTitles []string, areaName string) string {
	parts := []string{title}
	switch len(pickedTitles) {
	case 0:
	case 1:
		parts = append(parts, pickedTitles[0])
	default:
		parts = append(parts, fmt.Sprintf("%s +%d", pickedTitles[0], len(pickedTitles)-1))
	}
	if areaName != "" {
		parts = append(parts, areaName)
	}
	return strings.Join(parts, " · ")
}

// SampleLimit is the number of rows of a preview sample.
const SampleLimit = 20
