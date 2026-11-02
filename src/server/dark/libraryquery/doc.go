// Package libraryquery builds read-only SQL for choosing rows of a catalogued table by area code, category and brand.
//
// The package is pure: it takes a table description and a selection and returns SQL text. It reads no
// environment, clock, network or database and keeps no state, so the same input always gives the same text.
//
// # Trust model
//
// Identifiers (schema, table and column names) come from the caller's catalogue and are always double-quoted,
// with embedded double quotes doubled. Selection values (area code, category and brand ids) come from untrusted
// input and are never quoted into SQL unless they match a strict pattern first:
//
//   - an area code is exactly 2, 4 or 7 ASCII digits;
//   - a category or brand id is 1 to 64 characters of ASCII lowercase letters, digits and underscore.
//
// A value that does not match is rejected with [ErrInvalidArea] or [ErrInvalidID]; it is never escaped and
// emitted. Accepted values are still written as single-quoted literals with embedded single quotes doubled,
// as a second layer. No value is ever emitted as an identifier, and nothing in the output depends on the
// caller's locale.
//
// # Category access
//
// A table can carry a category column. The caller passes the set of category ids the reader may see; the
// emitted filter uses the intersection of that set and the requested categories, so a category that is not
// allowed can never reach the SQL. When the intersection is empty, building fails with [ErrNoAllowedCategory].
//
// # Output shape
//
// Every statement is a single SELECT of the table's columns with an optional WHERE clause of AND-ed
// conditions in fixed order: area, category, brand. Lists inside IN (...) are sorted and deduplicated.
package libraryquery
