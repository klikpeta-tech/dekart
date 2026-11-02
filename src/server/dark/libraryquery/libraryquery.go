package libraryquery

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Errors returned by the builders. Compare with [errors.Is]; the wrapped text adds the offending field name
// but never echoes the rejected value.
var (
	// ErrInvalidIdentifier is returned for an empty, over-long or NUL-containing table or column name.
	ErrInvalidIdentifier = errors.New("libraryquery: invalid identifier")
	// ErrNoColumns is returned when a table lists no columns.
	ErrNoColumns = errors.New("libraryquery: table has no columns")
	// ErrInvalidArea is returned when an area code is not 2, 4 or 7 ASCII digits.
	ErrInvalidArea = errors.New("libraryquery: invalid area code")
	// ErrAreaLevelUnsupported is returned when the table has no column for the length of the area code.
	ErrAreaLevelUnsupported = errors.New("libraryquery: table cannot be filtered at this area level")
	// ErrInvalidID is returned when a category or brand id does not match [IDPattern].
	ErrInvalidID = errors.New("libraryquery: invalid id")
	// ErrCategoryRequired is returned when the table has a category column and no category was requested.
	ErrCategoryRequired = errors.New("libraryquery: at least one category is required")
	// ErrNoAllowedCategory is returned when none of the requested categories is in the allowed set.
	ErrNoAllowedCategory = errors.New("libraryquery: no requested category is allowed")
	// ErrCategoriesUnsupported is returned when categories are requested from a table without a category column.
	ErrCategoriesUnsupported = errors.New("libraryquery: table has no category column")
	// ErrBrandsUnsupported is returned when brands are requested from a table without a brand column.
	ErrBrandsUnsupported = errors.New("libraryquery: table has no brand column")
	// ErrTooManyValues is returned when a list holds more than [MaxListSize] entries.
	ErrTooManyValues = errors.New("libraryquery: too many values")
	// ErrInvalidLimit is returned for a sample limit outside 1 to [MaxSampleLimit].
	ErrInvalidLimit = errors.New("libraryquery: invalid limit")
)

const (
	// MaxListSize is the largest number of category or brand ids accepted in one list.
	MaxListSize = 500
	// MaxSampleLimit is the largest row limit [BuildSample] accepts.
	MaxSampleLimit = 1000
	maxIdentifier  = 128
)

// IDPattern is the pattern a category or brand id must match: 1 to 64 ASCII lowercase letters, digits or underscores.
const IDPattern = `^[a-z0-9_]{1,64}$`

var (
	idRegexp   = regexp.MustCompile(`\A[a-z0-9_]{1,64}\z`)
	areaRegexp = regexp.MustCompile(`\A(?:[0-9]{2}|[0-9]{4}|[0-9]{7})\z`)
)

// AreaColumn names the column that holds area codes of one length.
type AreaColumn struct {
	// Length is the number of digits of the area codes in Column: 2, 4 or 7.
	Length int
	// Column is the column name; it is quoted as an identifier.
	Column string
}

// Table describes a relation to read from. All names are identifiers: they are quoted, never validated against
// a pattern, so they must come from a trusted catalogue and not from a request.
type Table struct {
	// Schema is the optional schema name.
	Schema string
	// Name is the table or view name.
	Name string
	// Columns are the columns to select, in order. At least one is required.
	Columns []string
	// AreaColumns lists the columns that area filters apply to, one per code length. Empty when the table has no area filter.
	AreaColumns []AreaColumn
	// CategoryColumn is the column holding category ids, or empty when rows have no category.
	CategoryColumn string
	// BrandColumn is the column holding brand ids, or empty when rows have no brand.
	BrandColumn string
}

// Selection is the untrusted part of a query: which area, categories and brands to keep.
type Selection struct {
	// Area is an area code of 2, 4 or 7 ASCII digits, or empty for no area filter.
	Area string
	// Categories are the requested category ids. Required, at least one, when the table has a category column.
	Categories []string
	// Brands are optional brand ids, allowed only when the table has a brand column.
	Brands []string
}

// quoteIdentifier returns name as a double-quoted identifier with embedded double quotes doubled.
func quoteIdentifier(name string) (string, error) {
	if name == "" || len(name) > maxIdentifier || strings.ContainsRune(name, 0) {
		return "", ErrInvalidIdentifier
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`, nil
}

// quoteLiteral returns value as a single-quoted literal with embedded single quotes doubled.
func quoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

// normalizeIDs validates every id against [IDPattern] and returns them sorted and deduplicated.
func normalizeIDs(field string, values []string) ([]string, error) {
	if len(values) > MaxListSize {
		return nil, fmt.Errorf("%w: %s", ErrTooManyValues, field)
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !idRegexp.MatchString(value) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidID, field)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

// ValidID reports whether id matches [IDPattern], the shape every category or brand id must have.
func ValidID(id string) bool {
	return idRegexp.MatchString(id)
}

// ValidArea reports whether code is empty (no area filter) or exactly 2, 4 or 7 ASCII digits.
func ValidArea(code string) bool {
	return code == "" || areaRegexp.MatchString(code)
}

// Intersect returns the ids present in both lists, sorted and without duplicates. Ids that do not match
// [IDPattern] are dropped, so the result is always safe to pass on. It does not modify its arguments.
func Intersect(requested, allowed []string) []string {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(requested))
	result := []string{}
	for _, id := range requested {
		if _, ok := allowedSet[id]; !ok || !idRegexp.MatchString(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// conditions returns the WHERE conditions for a selection, in the fixed order area, category, brand.
func conditions(table Table, selection Selection, allowed []string) ([]string, error) {
	var result []string
	if selection.Area != "" {
		if !areaRegexp.MatchString(selection.Area) {
			return nil, ErrInvalidArea
		}
		column := ""
		for _, candidate := range table.AreaColumns {
			if candidate.Length == len(selection.Area) {
				column = candidate.Column
			}
		}
		if column == "" {
			return nil, ErrAreaLevelUnsupported
		}
		quoted, err := quoteIdentifier(column)
		if err != nil {
			return nil, err
		}
		result = append(result, quoted+" = "+quoteLiteral(selection.Area))
	}
	if table.CategoryColumn == "" {
		if len(selection.Categories) > 0 {
			return nil, ErrCategoriesUnsupported
		}
	} else {
		requested, err := normalizeIDs("categories", selection.Categories)
		if err != nil {
			return nil, err
		}
		if len(requested) == 0 {
			return nil, ErrCategoryRequired
		}
		granted := Intersect(requested, allowed)
		if len(granted) == 0 {
			return nil, ErrNoAllowedCategory
		}
		condition, err := inList(table.CategoryColumn, granted)
		if err != nil {
			return nil, err
		}
		result = append(result, condition)
	}
	brands, err := normalizeIDs("brands", selection.Brands)
	if err != nil {
		return nil, err
	}
	if len(brands) > 0 {
		if table.BrandColumn == "" {
			return nil, ErrBrandsUnsupported
		}
		condition, err := inList(table.BrandColumn, brands)
		if err != nil {
			return nil, err
		}
		result = append(result, condition)
	}
	return result, nil
}

// inList renders `"column" IN ('a', 'b')` for already validated ids.
func inList(column string, ids []string) (string, error) {
	quoted, err := quoteIdentifier(column)
	if err != nil {
		return "", err
	}
	literals := make([]string, len(ids))
	for i, id := range ids {
		literals[i] = quoteLiteral(id)
	}
	return quoted + " IN (" + strings.Join(literals, ", ") + ")", nil
}

// Build returns the SELECT statement that reads the rows of table matching selection.
//
// allowed is the set of category ids the reader may see; it is only consulted when the table has a category
// column, where the emitted filter uses the intersection of allowed and selection.Categories.
//
// Escaping: schema, table and column names are double-quoted with embedded double quotes doubled; area, category
// and brand values must match their strict patterns first and are then emitted as single-quoted literals with
// embedded single quotes doubled. A value that fails validation is rejected, never escaped. The result is one
// statement with no trailing semicolon.
//
// Errors: [ErrNoColumns], [ErrInvalidIdentifier], [ErrInvalidArea], [ErrAreaLevelUnsupported], [ErrInvalidID],
// [ErrCategoryRequired], [ErrNoAllowedCategory], [ErrCategoriesUnsupported], [ErrBrandsUnsupported] and
// [ErrTooManyValues]. Build never returns partial SQL together with an error.
func Build(table Table, selection Selection, allowed []string) (string, error) {
	if len(table.Columns) == 0 {
		return "", ErrNoColumns
	}
	columns := make([]string, len(table.Columns))
	for i, column := range table.Columns {
		quoted, err := quoteIdentifier(column)
		if err != nil {
			return "", err
		}
		columns[i] = quoted
	}
	name, err := quoteIdentifier(table.Name)
	if err != nil {
		return "", err
	}
	relation := name
	if table.Schema != "" {
		schema, err := quoteIdentifier(table.Schema)
		if err != nil {
			return "", err
		}
		relation = schema + "." + name
	}
	where, err := conditions(table, selection, allowed)
	if err != nil {
		return "", err
	}
	statement := "SELECT " + strings.Join(columns, ", ") + "\nFROM " + relation
	if len(where) > 0 {
		statement += "\nWHERE " + strings.Join(where, "\n  AND ")
	}
	return statement, nil
}

// BuildSample returns the statement of [Build] followed by a LIMIT clause of limit rows.
//
// limit must be between 1 and [MaxSampleLimit]; it is formatted as a decimal integer, never taken as text.
// All other escaping and errors are those of [Build], plus [ErrInvalidLimit].
func BuildSample(table Table, selection Selection, allowed []string, limit int) (string, error) {
	if limit < 1 || limit > MaxSampleLimit {
		return "", ErrInvalidLimit
	}
	statement, err := Build(table, selection, allowed)
	if err != nil {
		return "", err
	}
	return statement + "\nLIMIT " + strconv.Itoa(limit), nil
}
