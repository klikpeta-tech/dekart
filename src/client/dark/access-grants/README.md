# access-grants

Classify time-limited access grants into display states and filter lists of grant rows.

## Contents

- [API at a glance](#api-at-a-glance)
- [Interfaces](#interfaces)
  - [FilterableRow](#api-filterablerow)
  - [Grant](#api-grant-1)
  - [RowState](#api-rowstate)
- [Type Aliases](#type-aliases)
  - [GrantStatus](#api-grantstatus-1)
  - [RowFilter](#api-rowfilter)
  - [RowKind](#api-rowkind)
- [Variables](#variables)
  - [endingSoonDays](#api-endingsoondays)
- [Functions](#functions)
  - [calendarDay()](#api-calendarday)
  - [filterRows()](#api-filterrows)
  - [grantRowState()](#api-grantrowstate)

## API at a glance

| API | Kind | Description |
| :--- | :--- | :--- |
| [FilterableRow](#api-filterablerow) | Interfaces | A row that can be searched and filtered. |
| [Grant](#api-grant-1) | Interfaces | Time-limited access recorded for one row. |
| [RowState](#api-rowstate) | Interfaces | Display state of one row. |
| [GrantStatus](#api-grantstatus-1) | Type Aliases | Latest recorded decision for one row, as reported by the source of truth. |
| [RowFilter](#api-rowfilter) | Type Aliases | List filter: everything, or one group of row kinds. |
| [RowKind](#api-rowkind) | Type Aliases | What a row currently looks like to the person managing access. |
| [endingSoonDays](#api-endingsoondays) | Variables | Active access that stops within this many days counts as ending soon. |
| [calendarDay](#api-calendarday) | Functions | The calendar day of a moment in a fixed UTC offset, for bounding date pickers. |
| [filterRows](#api-filterrows) | Functions | Keep the rows that match a filter chip and a search text, in their original order. |
| [grantRowState](#api-grantrowstate) | Functions | Classify one row's access at a given moment. |

## Interfaces

<a id="api-filterablerow"></a>

### FilterableRow

A row that can be searched and filtered.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-category"></a> `category` | `string` | Secondary text searched. |
| <a id="api-grant"></a> `grant` | [`Grant`](#api-grant-1) | Access recorded for the row. |
| <a id="api-title"></a> `title` | `string` | Main text searched. |

***

<a id="api-grant-1"></a>

### Grant

Time-limited access recorded for one row.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-endsat"></a> `endsAt` | `number` | Unix seconds at which access stops, or 0 when none was ever recorded. |
| <a id="api-endson"></a> `endsOn` | `string` | Last day of access as `YYYY-MM-DD`, or an empty string when none was ever recorded. |
| <a id="api-status"></a> `status` | [`GrantStatus`](#api-grantstatus-1) | Latest decision: never granted, active, past its end, or withdrawn. |

***

<a id="api-rowstate"></a>

### RowState

Display state of one row.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-daysleft"></a> `daysLeft` | `number` | Whole days of access left, rounded up; 0 unless `kind` is `granted` or `ending-soon`. |
| <a id="api-kind"></a> `kind` | [`RowKind`](#api-rowkind) | Category of the row's current access. |
| <a id="api-label"></a> `label` | `string` | Plain display text (`Not granted`, `Ends in 5 days`, `Ended 2026-11-12`). Render it as text, never as markup. |

## Type Aliases

<a id="api-grantstatus-1"></a>

### GrantStatus

```ts
type GrantStatus = "none" | "active" | "ended" | "revoked";
```

Latest recorded decision for one row, as reported by the source of truth.

***

<a id="api-rowfilter"></a>

### RowFilter

```ts
type RowFilter = "all" | "granted" | "ending-soon" | "not-granted";
```

List filter: everything, or one group of row kinds.

***

<a id="api-rowkind"></a>

### RowKind

```ts
type RowKind = "granted" | "ending-soon" | "ended" | "not-granted";
```

What a row currently looks like to the person managing access.

## Variables

<a id="api-endingsoondays"></a>

### endingSoonDays

```ts
const endingSoonDays: 14 = 14;
```

Active access that stops within this many days counts as ending soon.

## Functions

<a id="api-calendarday"></a>

### calendarDay()

```ts
function calendarDay(nowSeconds, utcOffsetMinutes): string;
```

The calendar day of a moment in a fixed UTC offset, for bounding date pickers.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `nowSeconds` | `number` | Moment in unix seconds, supplied by the caller. |
| `utcOffsetMinutes` | `number` | Offset of the calendar from UTC in minutes (420 for UTC+7). |

#### Returns

`string`

The day as `YYYY-MM-DD`.

#### Example

```ts
calendarDay(Date.UTC(2026, 10, 1, 20) / 1000, 420) // '2026-11-02'
```

***

<a id="api-filterrows"></a>

### filterRows()

```ts
function filterRows<T>(
   rows,
   options,
   nowSeconds): T[];
```

Keep the rows that match a filter chip and a search text, in their original order.

`granted` keeps rows with current access (including ending soon), `ending-soon`
keeps only rows whose access stops soon, and `not-granted` keeps rows without
current access, including ended ones. Search is a case-insensitive substring match
on `title` and `category`; blank search keeps everything.

#### Type Parameters

| Type Parameter |
| ------ |
| `T` *extends* [`FilterableRow`](#api-filterablerow) |

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `rows` | readonly `T`[] | Rows to filter; not modified. |
| `options` | \{ `filter`: [`RowFilter`](#api-rowfilter); `search`: `string`; \} | The filter chip and the search text. |
| `options.filter` | [`RowFilter`](#api-rowfilter) | Group of row kinds to keep. |
| `options.search` | `string` | Text to look for in title and category; blank matches all. |
| `nowSeconds` | `number` | Current time in unix seconds, supplied by the caller. |

#### Returns

`T`[]

A new array with the matching rows.

***

<a id="api-grantrowstate"></a>

### grantRowState()

```ts
function grantRowState(grant, nowSeconds): RowState;
```

Classify one row's access at a given moment.

Active access is `granted`, or `ending-soon` when at most [endingSoonDays](#api-endingsoondays)
days remain. Access whose `endsAt` has passed is `ended` even if the status still
says active, so a stale status never shows access that is gone. A withdrawn or
never-recorded grant is `not-granted`; a withdrawn grant keeps its `endsOn` for the caller.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `grant` | [`Grant`](#api-grant-1) | Access recorded for the row. |
| `nowSeconds` | `number` | Current time in unix seconds, supplied by the caller. |

#### Returns

[`RowState`](#api-rowstate)

The row's kind, label and remaining whole days.

#### Example

```ts
grantRowState({ status: 'active', endsOn: '2026-11-12', endsAt: 1_000_000 + 3 * 86400 }, 1_000_000)
// { kind: 'ending-soon', label: 'Ends in 3 days', daysLeft: 3 }
```
