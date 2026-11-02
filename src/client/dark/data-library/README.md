# data-library

Browse a catalogue of data items: filter and order them by text, category, geometry kind and region level,
and render the result as a searchable list with category chips and availability badges.

## Contents

- [API at a glance](#api-at-a-glance)
- [Interfaces](#interfaces)
  - [AccessBadgeProps](#api-accessbadgeprops)
  - [AreaOption](#api-areaoption)
  - [CatalogueItem](#api-catalogueitem)
  - [CategoryCount](#api-categorycount)
  - [ItemShape](#api-itemshape)
  - [LibraryBrowserProps](#api-librarybrowserprops)
  - [LibraryItemRowProps](#api-libraryitemrowprops)
  - [LibraryQuery](#api-libraryquery)
- [Type Aliases](#type-aliases)
  - [ShapeKind](#api-shapekind)
- [Functions](#functions)
  - [AccessBadge()](#api-accessbadge)
  - [areaLevelOfCode()](#api-arealevelofcode)
  - [countItemsByCategory()](#api-countitemsbycategory)
  - [emptyLibraryQuery()](#api-emptylibraryquery)
  - [filterCatalogueItems()](#api-filtercatalogueitems)
  - [formatCount()](#api-formatcount)
  - [LibraryBrowser()](#api-librarybrowser)
  - [LibraryItemRow()](#api-libraryitemrow)
  - [shapeKindLabel()](#api-shapekindlabel)
  - [shapeKindsOf()](#api-shapekindsof)
  - [shapeLabel()](#api-shapelabel)

## API at a glance

| API | Kind | Description |
| :--- | :--- | :--- |
| [AccessBadgeProps](#api-accessbadgeprops) | Interfaces | Props of AccessBadge. |
| [AreaOption](#api-areaoption) | Interfaces | A selectable region in the browser's area select. |
| [CatalogueItem](#api-catalogueitem) | Interfaces | One browsable catalogue entry. |
| [CategoryCount](#api-categorycount) | Interfaces | Item count of one category under a query. |
| [ItemShape](#api-itemshape) | Interfaces | What one row of a catalogue item is. |
| [LibraryBrowserProps](#api-librarybrowserprops) | Interfaces | Props of LibraryBrowser. |
| [LibraryItemRowProps](#api-libraryitemrowprops) | Interfaces | Props of LibraryItemRow. |
| [LibraryQuery](#api-libraryquery) | Interfaces | The filter state of a browser. |
| [ShapeKind](#api-shapekind) | Type Aliases | Geometry kind of one row of a catalogue item. |
| [AccessBadge](#api-accessbadge) | Functions | Small pill that tells whether an item is available or already in use. |
| [areaLevelOfCode](#api-arealevelofcode) | Functions | The area level a region code addresses. |
| [countItemsByCategory](#api-countitemsbycategory) | Functions | Counts items per category for the category chips. |
| [emptyLibraryQuery](#api-emptylibraryquery) | Functions | The query to start from: no text, category or shape filter. |
| [filterCatalogueItems](#api-filtercatalogueitems) | Functions | Filters and orders catalogue items for display. |
| [formatCount](#api-formatcount) | Functions | Formats a whole number with Indonesian digit grouping, for example `2100` becomes `2.100`. |
| [LibraryBrowser](#api-librarybrowser) | Functions | Searchable, filterable list of catalogue items with category chips, an area select and a shape select. |
| [LibraryItemRow](#api-libraryitemrow) | Functions | Compact row for one catalogue item: title, category, shape, row count, coverage and an AccessBadge. |
| [shapeKindLabel](#api-shapekindlabel) | Functions | Label of a geometry kind, as used by the shape select. |
| [shapeKindsOf](#api-shapekindsof) | Functions | The geometry kinds present in a list of items, for the shape select. |
| [shapeLabel](#api-shapelabel) | Functions | Describes what one row is, including the administrative level or H3 resolution when known. |

## Interfaces

<a id="api-accessbadgeprops"></a>

### AccessBadgeProps

Props of [AccessBadge](#api-accessbadge).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-addedlabel"></a> `addedLabel?` | `string` | Text for the `added` state. Defaults to `In use`. Rendered as text, never as markup. |
| <a id="api-state"></a> `state` | `"available"` \| `"added"` | `available` for an item that can be added, `added` for one that is already in use. |

***

<a id="api-areaoption"></a>

### AreaOption

A selectable region in the browser's area select.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-code"></a> `code` | `string` | Region code of 2, 4 or 7 digits. |
| <a id="api-name"></a> `name` | `string` | Display name. |

***

<a id="api-catalogueitem"></a>

### CatalogueItem

One browsable catalogue entry. Every field is display or filter data; ids are opaque lookup keys.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-arealevels"></a> `areaLevels` | `string`[] | Area levels the item can be narrowed to: any of `provinsi`, `kabkota`, `kecamatan`. |
| <a id="api-category"></a> `category` | `string` | Display category used for the category chips. Compared exactly, so equal categories must be spelled equally. |
| <a id="api-coveragesummary"></a> `coverageSummary` | `string` | Short description of the covered region. Searchable. |
| <a id="api-id"></a> `id` | `string` | Opaque, unique id. Passed back unchanged to `onSelectItem`. |
| <a id="api-provider"></a> `provider` | `string` | Organisation that provides the data. Searchable. |
| <a id="api-rowcount"></a> `rowCount` | `number` | Number of rows in the whole item. Never negative. |
| <a id="api-searchterms"></a> `searchTerms?` | `string`[] | Extra searchable words that are not shown, for example the titles of sub-categories inside the item. |
| <a id="api-shape"></a> `shape` | [`ItemShape`](#api-itemshape) | What one row is. |
| <a id="api-title"></a> `title` | `string` | Display title. The year, when relevant, is part of the title. |

***

<a id="api-categorycount"></a>

### CategoryCount

Item count of one category under a query.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-category-1"></a> `category` | `string` | The category as spelled on the items. |
| <a id="api-count"></a> `count` | `number` | How many items of that category match the other filters. |

***

<a id="api-itemshape"></a>

### ItemShape

What one row of a catalogue item is.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-adminlevel"></a> `adminLevel?` | `string` | For `areas`: the administrative level the rows describe: `provinsi`, `kabkota` or `kecamatan`. |
| <a id="api-h3resolution"></a> `h3Resolution?` | `number` | For `h3`: the H3 resolution of the cells. |
| <a id="api-kind"></a> `kind` | [`ShapeKind`](#api-shapekind) | Geometry kind. |

***

<a id="api-librarybrowserprops"></a>

### LibraryBrowserProps

Props of [LibraryBrowser](#api-librarybrowser).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-addeditemids"></a> `addedItemIds?` | readonly `string`[] | Ids of items already in use; they show the added badge. |
| <a id="api-addedlabel-1"></a> `addedLabel?` | `string` | Text of the added badge. Defaults to `In use`. |
| <a id="api-areaoptions"></a> `areaOptions?` | readonly [`AreaOption`](#api-areaoption)[] | Regions offered by the area select, besides "All areas". A `query.areaCode` that is not listed is added as its code. |
| <a id="api-items"></a> `items` | readonly [`CatalogueItem`](#api-catalogueitem)[] \| `null` | The whole catalogue to browse, or `null` while it loads (skeleton rows are shown). |
| <a id="api-onback"></a> `onBack?` | () => `void` | Called when the back button is pressed. Without it no back button is shown. |
| <a id="api-onquerychange"></a> `onQueryChange` | (`query`) => `void` | Called with the complete next query whenever the user changes a filter. |
| <a id="api-onselectitem"></a> `onSelectItem?` | (`itemId`) => `void` | Called with the item id when a row is activated. Without it rows are not interactive. |
| <a id="api-query"></a> `query` | [`LibraryQuery`](#api-libraryquery) | Current filter state. The browser is controlled: it renders this and reports changes. |

***

<a id="api-libraryitemrowprops"></a>

### LibraryItemRowProps

Props of [LibraryItemRow](#api-libraryitemrow).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-added"></a> `added?` | `boolean` | True when the item is already in use; shows the added badge instead of the available one. |
| <a id="api-addedlabel-2"></a> `addedLabel?` | `string` | Text of the added badge. Defaults to `In use`. |
| <a id="api-item"></a> `item` | [`CatalogueItem`](#api-catalogueitem) | The item to show. All text is rendered as text, never as markup. |
| <a id="api-onselect"></a> `onSelect?` | (`itemId`) => `void` | Called with the item id when the row is activated. Without it the row is not interactive. |

***

<a id="api-libraryquery"></a>

### LibraryQuery

The filter state of a browser.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-areacode"></a> `areaCode` | `string` | Region code of 2, 4 or 7 digits, or empty for no region filter. Any other value is treated as empty. |
| <a id="api-category-2"></a> `category` | `string` | Exact [CatalogueItem.category](#api-category), or empty for all categories. |
| <a id="api-search"></a> `search` | `string` | Free text. Words are matched case- and accent-insensitively; every word must match. Empty means no text filter. |
| <a id="api-shape-1"></a> `shape` | `""` \| [`ShapeKind`](#api-shapekind) | Geometry kind, or empty for all kinds. |

## Type Aliases

<a id="api-shapekind"></a>

### ShapeKind

```ts
type ShapeKind = "points" | "lines" | "areas" | "h3";
```

Geometry kind of one row of a catalogue item.

## Functions

<a id="api-accessbadge"></a>

### AccessBadge()

```ts
function AccessBadge(props): ReactElement;
```

Small pill that tells whether an item is available or already in use.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `props` | [`AccessBadgeProps`](#api-accessbadgeprops) | See [AccessBadgeProps](#api-accessbadgeprops). |

#### Returns

`ReactElement`

A `span` whose text is `Available` or the added label.

***

<a id="api-arealevelofcode"></a>

### areaLevelOfCode()

```ts
function areaLevelOfCode(areaCode): string | undefined;
```

The area level a region code addresses.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `areaCode` | `string` | A region code. |

#### Returns

`string` \| `undefined`

`provinsi` for 2 digits, `kabkota` for 4, `kecamatan` for 7, and `undefined` for anything else, including empty.

***

<a id="api-countitemsbycategory"></a>

### countItemsByCategory()

```ts
function countItemsByCategory(items, query): CategoryCount[];
```

Counts items per category for the category chips. The category in the query is ignored, so every chip shows how many
items it would show if picked; the text, shape and region filters apply.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `items` | readonly [`CatalogueItem`](#api-catalogueitem)[] | The catalogue items. |
| `query` | [`LibraryQuery`](#api-libraryquery) | The filter state. |

#### Returns

[`CategoryCount`](#api-categorycount)[]

One entry per category that has a matching item, sorted by category. Empty categories are left out.

***

<a id="api-emptylibraryquery"></a>

### emptyLibraryQuery()

```ts
function emptyLibraryQuery(areaCode): LibraryQuery;
```

The query to start from: no text, category or shape filter.

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `areaCode` | `string` | `''` | Optional preset region code. Defaults to no region filter. |

#### Returns

[`LibraryQuery`](#api-libraryquery)

A fresh query.

***

<a id="api-filtercatalogueitems"></a>

### filterCatalogueItems()

```ts
function filterCatalogueItems(items, query): CatalogueItem[];
```

Filters and orders catalogue items for display. Does not modify the input.

Filtering: an item must match the text search, the category, the shape kind and the region. The text search looks at
title, category, provider, coverage summary and `searchTerms`. A region code only keeps items whose `areaLevels` include the
code's level (2 digits `provinsi`, 4 `kabkota`, 7 `kecamatan`); an invalid code filters nothing.

Ordering: without text, by category then title. With text, titles that start with the text first, then titles that contain
every word, then matches through other fields; ties by title. Comparison is accent-insensitive, using Indonesian collation.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `items` | readonly [`CatalogueItem`](#api-catalogueitem)[] | The catalogue items. |
| `query` | [`LibraryQuery`](#api-libraryquery) | The filter state. |

#### Returns

[`CatalogueItem`](#api-catalogueitem)[]

The matching items in display order.

#### Example

```ts
const shown = filterCatalogueItems(items, { ...emptyLibraryQuery('3171'), search: 'apotek' })
```

***

<a id="api-formatcount"></a>

### formatCount()

```ts
function formatCount(value): string;
```

Formats a whole number with Indonesian digit grouping, for example `2100` becomes `2.100`.
Negative and non-finite inputs are formatted as `0`; fractions are rounded down.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `value` | `number` | The number to format. |

#### Returns

`string`

The formatted text.

***

<a id="api-librarybrowser"></a>

### LibraryBrowser()

```ts
function LibraryBrowser(props): ReactElement;
```

Searchable, filterable list of catalogue items with category chips, an area select and a shape select.

Filtering and ordering follow [filterCatalogueItems](#api-filtercatalogueitems); chip counts follow [countItemsByCategory](#api-countitemsbycategory). The shape
select lists only the kinds present in `items`. While `items` is `null` it shows skeleton rows; when nothing matches it
shows an empty state with a button that clears every filter.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `props` | [`LibraryBrowserProps`](#api-librarybrowserprops) | See [LibraryBrowserProps](#api-librarybrowserprops). |

#### Returns

`ReactElement`

The browser element.

***

<a id="api-libraryitemrow"></a>

### LibraryItemRow()

```ts
function LibraryItemRow(props): ReactElement;
```

Compact row for one catalogue item: title, category, shape, row count, coverage and an [AccessBadge](#api-accessbadge).

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `props` | [`LibraryItemRowProps`](#api-libraryitemrowprops) | See [LibraryItemRowProps](#api-libraryitemrowprops). |

#### Returns

`ReactElement`

A list item; a button inside it when `onSelect` is given.

***

<a id="api-shapekindlabel"></a>

### shapeKindLabel()

```ts
function shapeKindLabel(kind): string;
```

Label of a geometry kind, as used by the shape select.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kind` | [`ShapeKind`](#api-shapekind) | The geometry kind. |

#### Returns

`string`

A short English label, for example `Points`.

***

<a id="api-shapekindsof"></a>

### shapeKindsOf()

```ts
function shapeKindsOf(items): ShapeKind[];
```

The geometry kinds present in a list of items, for the shape select.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `items` | readonly [`CatalogueItem`](#api-catalogueitem)[] | The catalogue items. |

#### Returns

[`ShapeKind`](#api-shapekind)[]

Distinct kinds in the fixed order points, lines, areas, h3.

***

<a id="api-shapelabel"></a>

### shapeLabel()

```ts
function shapeLabel(shape): string;
```

Describes what one row is, including the administrative level or H3 resolution when known.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `shape` | [`ItemShape`](#api-itemshape) | The item shape. |

#### Returns

`string`

For example `Areas · Kabupaten/Kota` or `H3 hexagons · res 8`. Unknown administrative levels are shown as given.
