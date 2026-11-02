# data-library

Browse a catalogue of data items: filter and order them by text, category, geometry kind and region level,
and render the result as a searchable list with category chips and availability badges. Describe one item, pick a
region and categories from it, validate and name the selection, and show the state of a dataset made from it.

## Contents

- [API at a glance](#api-at-a-glance)
- [Interfaces](#interfaces)
  - [AccessBadgeProps](#api-accessbadgeprops)
  - [AreaNode](#api-areanode)
  - [AreaOption](#api-areaoption)
  - [AreaPickerProps](#api-areapickerprops)
  - [CatalogueItem](#api-catalogueitem)
  - [CategoryCount](#api-categorycount)
  - [CategoryPickerProps](#api-categorypickerprops)
  - [ItemDetail](#api-itemdetail)
  - [ItemShape](#api-itemshape)
  - [LibraryBrowserProps](#api-librarybrowserprops)
  - [LibraryDatasetStatusProps](#api-librarydatasetstatusprops)
  - [LibraryItemDetailProps](#api-libraryitemdetailprops)
  - [LibraryItemRowProps](#api-libraryitemrowprops)
  - [LibraryQuery](#api-libraryquery)
  - [SampleTable](#api-sampletable)
  - [Selection](#api-selection)
  - [SelectionIssue](#api-selectionissue)
  - [Taxonomy](#api-taxonomy-1)
  - [TaxonomyBrand](#api-taxonomybrand)
  - [TaxonomyCategory](#api-taxonomycategory)
  - [TaxonomyGroup](#api-taxonomygroup)
- [Type Aliases](#type-aliases)
  - [ShapeKind](#api-shapekind)
- [Functions](#functions)
  - [AccessBadge()](#api-accessbadge)
  - [addButtonLabel()](#api-addbuttonlabel)
  - [areaLevelLabel()](#api-arealevellabel)
  - [areaLevelOfCode()](#api-arealevelofcode)
  - [AreaPicker()](#api-areapicker)
  - [CategoryPicker()](#api-categorypicker)
  - [countItemsByCategory()](#api-countitemsbycategory)
  - [emptyLibraryQuery()](#api-emptylibraryquery)
  - [filterCatalogueItems()](#api-filtercatalogueitems)
  - [formatCount()](#api-formatcount)
  - [formatDate()](#api-formatdate)
  - [formatEstimate()](#api-formatestimate)
  - [LibraryBrowser()](#api-librarybrowser)
  - [LibraryDatasetStatus()](#api-librarydatasetstatus)
  - [LibraryItemDetail()](#api-libraryitemdetail)
  - [LibraryItemRow()](#api-libraryitemrow)
  - [shapeKindLabel()](#api-shapekindlabel)
  - [shapeKindsOf()](#api-shapekindsof)
  - [shapeLabel()](#api-shapelabel)
  - [tabName()](#api-tabname)
  - [validateSelection()](#api-validateselection)

## API at a glance

| API | Kind | Description |
| :--- | :--- | :--- |
| [AccessBadgeProps](#api-accessbadgeprops) | Interfaces | Props of AccessBadge. |
| [AreaNode](#api-areanode) | Interfaces | One region offered by the AreaPicker. |
| [AreaOption](#api-areaoption) | Interfaces | A selectable region in the browser's area select. |
| [AreaPickerProps](#api-areapickerprops) | Interfaces | Props of AreaPicker. |
| [CatalogueItem](#api-catalogueitem) | Interfaces | One browsable catalogue entry. |
| [CategoryCount](#api-categorycount) | Interfaces | Item count of one category under a query. |
| [CategoryPickerProps](#api-categorypickerprops) | Interfaces | Props of CategoryPicker. |
| [ItemDetail](#api-itemdetail) | Interfaces | A catalogue item with everything the detail view shows. |
| [ItemShape](#api-itemshape) | Interfaces | What one row of a catalogue item is. |
| [LibraryBrowserProps](#api-librarybrowserprops) | Interfaces | Props of LibraryBrowser. |
| [LibraryDatasetStatusProps](#api-librarydatasetstatusprops) | Interfaces | Props of LibraryDatasetStatus. |
| [LibraryItemDetailProps](#api-libraryitemdetailprops) | Interfaces | Props of LibraryItemDetail. |
| [LibraryItemRowProps](#api-libraryitemrowprops) | Interfaces | Props of LibraryItemRow. |
| [LibraryQuery](#api-libraryquery) | Interfaces | The filter state of a browser. |
| [SampleTable](#api-sampletable) | Interfaces | A few rows of an item, for a preview. |
| [Selection](#api-selection) | Interfaces | What is picked from an item: where, and which categories and brands. |
| [SelectionIssue](#api-selectionissue) | Interfaces | Why a selection cannot be added. |
| [Taxonomy](#api-taxonomy-1) | Interfaces | The categories a caller may pick from, in display order. |
| [TaxonomyBrand](#api-taxonomybrand) | Interfaces | One pickable brand inside a category. |
| [TaxonomyCategory](#api-taxonomycategory) | Interfaces | One pickable category. |
| [TaxonomyGroup](#api-taxonomygroup) | Interfaces | A group of categories. |
| [ShapeKind](#api-shapekind) | Type Aliases | Geometry kind of one row of a catalogue item. |
| [AccessBadge](#api-accessbadge) | Functions | Small pill that tells whether an item is available or already in use. |
| [addButtonLabel](#api-addbuttonlabel) | Functions | Label of the button that adds a selection to a map. |
| [areaLevelLabel](#api-arealevellabel) | Functions | Label of an area level. |
| [areaLevelOfCode](#api-arealevelofcode) | Functions | The area level a region code addresses. |
| [AreaPicker](#api-areapicker) | Functions | Cascading selects that narrow a selection to a region: province, then district or city, then sub-district. |
| [CategoryPicker](#api-categorypicker) | Functions | Picks categories, and optionally brands, from grouped lists. |
| [countItemsByCategory](#api-countitemsbycategory) | Functions | Counts items per category for the category chips. |
| [emptyLibraryQuery](#api-emptylibraryquery) | Functions | The query to start from: no text, category or shape filter. |
| [filterCatalogueItems](#api-filtercatalogueitems) | Functions | Filters and orders catalogue items for display. |
| [formatCount](#api-formatcount) | Functions | Formats a whole number with Indonesian digit grouping, for example `2100` becomes `2.100`. |
| [formatDate](#api-formatdate) | Functions | Formats a point in time as a long Indonesian calendar date in the Asia/Jakarta time zone. |
| [formatEstimate](#api-formatestimate) | Functions | Formats a row estimate with the noun that fits the geometry kind, using Indonesian digit grouping. |
| [LibraryBrowser](#api-librarybrowser) | Functions | Searchable, filterable list of catalogue items with category chips, an area select and a shape select. |
| [LibraryDatasetStatus](#api-librarydatasetstatus) | Functions | Status panel of a dataset that comes from a catalogue: a status bar and the actions that change or refresh it. |
| [LibraryItemDetail](#api-libraryitemdetail) | Functions | Detail view of one catalogue item: description, a grid of facts, the column names and the source credit, with room for pickers, a "Preview sample" button and one primary action. |
| [LibraryItemRow](#api-libraryitemrow) | Functions | Compact row for one catalogue item: title, category, shape, row count, coverage and an AccessBadge. |
| [shapeKindLabel](#api-shapekindlabel) | Functions | Label of a geometry kind, as used by the shape select. |
| [shapeKindsOf](#api-shapekindsof) | Functions | The geometry kinds present in a list of items, for the shape select. |
| [shapeLabel](#api-shapelabel) | Functions | Describes what one row is, including the administrative level or H3 resolution when known. |
| [tabName](#api-tabname) | Functions | Names the tab of a dataset made from a selection: `<Title> · <Category or brand> +n · <Area>`. |
| [validateSelection](#api-validateselection) | Functions | Checks a selection against what its item allows, so a button can be disabled and say why. |

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

<a id="api-areanode"></a>

### AreaNode

One region offered by the [AreaPicker](#api-areapicker).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-code"></a> `code` | `string` | Region code of 2, 4 or 7 digits. |
| <a id="api-name"></a> `name` | `string` | Display name. |
| <a id="api-rowcount"></a> `rowCount?` | `number` | Rows of the item inside this region. Shown next to the name when present. |

***

<a id="api-areaoption"></a>

### AreaOption

A selectable region in the browser's area select.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-code-1"></a> `code` | `string` | Region code of 2, 4 or 7 digits. |
| <a id="api-name-1"></a> `name` | `string` | Display name. |

***

<a id="api-areapickerprops"></a>

### AreaPickerProps

Props of [AreaPicker](#api-areapicker).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-levels"></a> `levels` | readonly `string`[] | Area levels the item can be narrowed to: any of `provinsi`, `kabkota`, `kecamatan`. Selects are shown down to the deepest of them; nothing is shown when empty. |
| <a id="api-loadareas"></a> `loadAreas` | (`parentCode`) => `Promise`\<readonly [`AreaNode`](#api-areanode)[]\> | Loads the regions directly below a region code; the empty code loads the top level. Called once per parent code, and again after a failure when the parent changes back. |
| <a id="api-onchange"></a> `onChange` | (`code`) => `void` | Called with the complete next region code, or an empty string for the whole coverage. |
| <a id="api-onnamechange"></a> `onNameChange?` | (`name`) => `void` | Called with the display name of the picked region whenever it becomes known, and with an empty string for the whole coverage. |
| <a id="api-value"></a> `value` | `string` | The picked region code of 2, 4 or 7 digits, or empty for the whole coverage. Any other value is treated as empty. |

***

<a id="api-catalogueitem"></a>

### CatalogueItem

One browsable catalogue entry. Every field is display or filter data; ids are opaque lookup keys.

#### Extended by

- [`ItemDetail`](#api-itemdetail)

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-arealevels"></a> `areaLevels` | `string`[] | Area levels the item can be narrowed to: any of `provinsi`, `kabkota`, `kecamatan`. |
| <a id="api-category"></a> `category` | `string` | Display category used for the category chips. Compared exactly, so equal categories must be spelled equally. |
| <a id="api-coveragesummary"></a> `coverageSummary` | `string` | Short description of the covered region. Searchable. |
| <a id="api-id"></a> `id` | `string` | Opaque, unique id. Passed back unchanged to `onSelectItem`. |
| <a id="api-provider"></a> `provider` | `string` | Organisation that provides the data. Searchable. |
| <a id="api-rowcount-1"></a> `rowCount` | `number` | Number of rows in the whole item. Never negative. |
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

<a id="api-categorypickerprops"></a>

### CategoryPickerProps

Props of [CategoryPicker](#api-categorypicker).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-brandcounts"></a> `brandCounts?` | `Readonly`\<`Record`\<`string`, `number`\>\> | Row counts by brand id, shown next to the titles. Brands without an entry show no count. |
| <a id="api-brandids"></a> `brandIds` | readonly `string`[] | Ids of the picked brands. |
| <a id="api-categorycounts"></a> `categoryCounts?` | `Readonly`\<`Record`\<`string`, `number`\>\> | Row counts by category id, shown next to the titles. Categories without an entry show no count. |
| <a id="api-categoryids"></a> `categoryIds` | readonly `string`[] | Ids of the picked categories. |
| <a id="api-onchange-1"></a> `onChange` | (`next`) => `void` | Called with the complete next picks. Brands of a category that is no longer picked are dropped. |
| <a id="api-taxonomy"></a> `taxonomy` | [`Taxonomy`](#api-taxonomy-1) | The categories the user may pick from. Only these are shown; nothing outside them can be picked. |

***

<a id="api-itemdetail"></a>

### ItemDetail

A catalogue item with everything the detail view shows.

#### Extends

- [`CatalogueItem`](#api-catalogueitem)

#### Properties

| Property | Type | Description | Inherited from |
| ------ | ------ | ------ | ------ |
| <a id="api-arealevels-1"></a> `areaLevels` | `string`[] | Area levels the item can be narrowed to: any of `provinsi`, `kabkota`, `kecamatan`. | [`CatalogueItem`](#api-catalogueitem).[`areaLevels`](#api-arealevels) |
| <a id="api-attribution"></a> `attribution` | `string` | Credit line for the data source. Empty when there is none. | - |
| <a id="api-cadence"></a> `cadence` | `string` | How often the data is refreshed, as display text. Empty when unknown. | - |
| <a id="api-category-2"></a> `category` | `string` | Display category used for the category chips. Compared exactly, so equal categories must be spelled equally. | [`CatalogueItem`](#api-catalogueitem).[`category`](#api-category) |
| <a id="api-columns"></a> `columns` | `string`[] | Names of the columns a row has. | - |
| <a id="api-coveragesummary-1"></a> `coverageSummary` | `string` | Short description of the covered region. Searchable. | [`CatalogueItem`](#api-catalogueitem).[`coverageSummary`](#api-coveragesummary) |
| <a id="api-description"></a> `description` | `string` | Description as plain text. Blank lines separate paragraphs; it is rendered as text, never as markup. | - |
| <a id="api-hastaxonomy"></a> `hasTaxonomy` | `boolean` | True when rows carry categories (and optional brands) that must be picked. | - |
| <a id="api-id-1"></a> `id` | `string` | Opaque, unique id. Passed back unchanged to `onSelectItem`. | [`CatalogueItem`](#api-catalogueitem).[`id`](#api-id) |
| <a id="api-maxrows"></a> `maxRows` | `number` | Largest selection that can be added, in rows. `0` means no limit. | - |
| <a id="api-provider-1"></a> `provider` | `string` | Organisation that provides the data. Searchable. | [`CatalogueItem`](#api-catalogueitem).[`provider`](#api-provider) |
| <a id="api-rowcount-2"></a> `rowCount` | `number` | Number of rows in the whole item. Never negative. | [`CatalogueItem`](#api-catalogueitem).[`rowCount`](#api-rowcount-1) |
| <a id="api-searchterms-1"></a> `searchTerms?` | `string`[] | Extra searchable words that are not shown, for example the titles of sub-categories inside the item. | [`CatalogueItem`](#api-catalogueitem).[`searchTerms`](#api-searchterms) |
| <a id="api-shape-1"></a> `shape` | [`ItemShape`](#api-itemshape) | What one row is. | [`CatalogueItem`](#api-catalogueitem).[`shape`](#api-shape) |
| <a id="api-sourceupdatedat"></a> `sourceUpdatedAt` | `number` | When the source data was last updated, in Unix seconds. `0` when unknown. | - |
| <a id="api-title-1"></a> `title` | `string` | Display title. The year, when relevant, is part of the title. | [`CatalogueItem`](#api-catalogueitem).[`title`](#api-title) |

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

<a id="api-librarydatasetstatusprops"></a>

### LibraryDatasetStatusProps

Props of [LibraryDatasetStatus](#api-librarydatasetstatus).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-accessendedat"></a> `accessEndedAt?` | `number` | When access ended, in Unix seconds. `0` or omitted when unknown; the date is then left out. |
| <a id="api-canchange"></a> `canChange` | `boolean` | True when the user may change the selection. Hides the two change buttons otherwise. |
| <a id="api-canrefresh"></a> `canRefresh` | `boolean` | True when the user may refresh the data. Hides the refresh button otherwise. |
| <a id="api-error"></a> `error?` | `string` | Error text shown next to the failed state. Rendered as text, never as markup. |
| <a id="api-hascategories"></a> `hasCategories` | `boolean` | True when the dataset has categories, so the "Change category" button is shown. |
| <a id="api-onchangearea"></a> `onChangeArea` | () => `void` | Called when "Change area" is pressed. |
| <a id="api-onchangecategory"></a> `onChangeCategory` | () => `void` | Called when "Change category" is pressed. |
| <a id="api-onopenlibrary"></a> `onOpenLibrary?` | () => `void` | Called when "Open in Library" is pressed. Without it, or once access has ended, the button is not shown. |
| <a id="api-onrefresh"></a> `onRefresh` | () => `void` | Called when "Refresh" is pressed. |
| <a id="api-onretry"></a> `onRetry` | () => `void` | Called when "Retry" is pressed in the failed state. |
| <a id="api-state-1"></a> `state` | `"ended"` \| `"loading"` \| `"failed"` \| `"ready"` | `ready` when the data is loaded, `loading` while it is being fetched, `failed` after an error, `ended` when access to the item has ended. `ended` wins over what the data itself is doing. |
| <a id="api-title-2"></a> `title` | `string` | Name of the dataset, shown as the heading. Rendered as text, never as markup. |

***

<a id="api-libraryitemdetailprops"></a>

### LibraryItemDetailProps

Props of [LibraryItemDetail](#api-libraryitemdetail).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-children"></a> `children?` | `ReactNode` | Pickers or other controls, rendered between the column list and the action row. |
| <a id="api-issues"></a> `issues?` | readonly [`SelectionIssue`](#api-selectionissue)[] | Reasons the current selection cannot be added. Shown as a list above the action row. |
| <a id="api-item"></a> `item` | [`ItemDetail`](#api-itemdetail) | The item to describe. All text is rendered as text, never as markup. |
| <a id="api-notice"></a> `notice?` | `string` | An error to show above the action row, for example the text of a refused command. Empty for none. |
| <a id="api-onback-1"></a> `onBack?` | () => `void` | Called when the back button is pressed. Without it no back button is shown. |
| <a id="api-onpreviewsample"></a> `onPreviewSample?` | () => `void` | Called when "Preview sample" is pressed. Without it the button is not shown. |
| <a id="api-previewdisabled"></a> `previewDisabled?` | `boolean` | True while "Preview sample" is disabled, for example because the selection is not valid. |
| <a id="api-primary"></a> `primary?` | `object` | The primary action. Without it no primary button is shown. |
| `primary.disabled` | `boolean` | True while the action cannot be taken. |
| `primary.label` | `string` | Button text. |
| `primary.onClick` | () => `void` | Called when the button is pressed while enabled. |
| <a id="api-sample"></a> `sample?` | [`SampleTable`](#api-sampletable) \| `null` | The previewed rows, or `null` for none. Shown as a table. |

***

<a id="api-libraryitemrowprops"></a>

### LibraryItemRowProps

Props of [LibraryItemRow](#api-libraryitemrow).

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-added"></a> `added?` | `boolean` | True when the item is already in use; shows the added badge instead of the available one. |
| <a id="api-addedlabel-2"></a> `addedLabel?` | `string` | Text of the added badge. Defaults to `In use`. |
| <a id="api-item-1"></a> `item` | [`CatalogueItem`](#api-catalogueitem) | The item to show. All text is rendered as text, never as markup. |
| <a id="api-onselect"></a> `onSelect?` | (`itemId`) => `void` | Called with the item id when the row is activated. Without it the row is not interactive. |

***

<a id="api-libraryquery"></a>

### LibraryQuery

The filter state of a browser.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-areacode"></a> `areaCode` | `string` | Region code of 2, 4 or 7 digits, or empty for no region filter. Any other value is treated as empty. |
| <a id="api-category-3"></a> `category` | `string` | Exact [CatalogueItem.category](#api-category), or empty for all categories. |
| <a id="api-search"></a> `search` | `string` | Free text. Words are matched case- and accent-insensitively; every word must match. Empty means no text filter. |
| <a id="api-shape-2"></a> `shape` | `""` \| [`ShapeKind`](#api-shapekind) | Geometry kind, or empty for all kinds. |

***

<a id="api-sampletable"></a>

### SampleTable

A few rows of an item, for a preview. All cells are text.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-columns-1"></a> `columns` | `string`[] | Column names. |
| <a id="api-rows"></a> `rows` | `string`[][] | Rows of cell text, in column order. |

***

<a id="api-selection"></a>

### Selection

What is picked from an item: where, and which categories and brands.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-areacode-1"></a> `areaCode` | `string` | Region code of 2, 4 or 7 digits, or empty for the whole coverage. |
| <a id="api-brandids-1"></a> `brandIds` | `string`[] | Ids of the picked brands. Optional; each belongs to one of the picked categories. |
| <a id="api-categoryids-1"></a> `categoryIds` | `string`[] | Ids of the picked categories. Required (at least one) for items with a taxonomy. |

***

<a id="api-selectionissue"></a>

### SelectionIssue

Why a selection cannot be added.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-code-2"></a> `code` | \| `"empty"` \| `"area-invalid"` \| `"area-level"` \| `"category-required"` \| `"too-many"` | Machine-readable reason: `area-invalid`, `area-level`, `category-required`, `empty` or `too-many`. |
| <a id="api-message"></a> `message` | `string` | Display text that tells the user what to change. |

***

<a id="api-taxonomy-1"></a>

### Taxonomy

The categories a caller may pick from, in display order.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-brands"></a> `brands` | [`TaxonomyBrand`](#api-taxonomybrand)[] | Brands, in display order. |
| <a id="api-categories"></a> `categories` | [`TaxonomyCategory`](#api-taxonomycategory)[] | Categories, in display order within their group. |
| <a id="api-groups"></a> `groups` | [`TaxonomyGroup`](#api-taxonomygroup)[] | Groups, in display order. |

***

<a id="api-taxonomybrand"></a>

### TaxonomyBrand

One pickable brand inside a category.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-categoryid"></a> `categoryId` | `string` | Id of the [TaxonomyCategory](#api-taxonomycategory) it belongs to. |
| <a id="api-id-2"></a> `id` | `string` | Opaque, unique id. |
| <a id="api-title-3"></a> `title` | `string` | Display title. |

***

<a id="api-taxonomycategory"></a>

### TaxonomyCategory

One pickable category.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-groupid"></a> `groupId` | `string` | Id of the [TaxonomyGroup](#api-taxonomygroup) it belongs to. |
| <a id="api-id-3"></a> `id` | `string` | Opaque, unique id. |
| <a id="api-title-4"></a> `title` | `string` | Display title. |

***

<a id="api-taxonomygroup"></a>

### TaxonomyGroup

A group of categories.

#### Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="api-id-4"></a> `id` | `string` | Opaque, unique id. |
| <a id="api-title-5"></a> `title` | `string` | Display title. |

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

<a id="api-addbuttonlabel"></a>

### addButtonLabel()

```ts
function addButtonLabel(kind, estimate): string;
```

Label of the button that adds a selection to a map.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kind` | [`ShapeKind`](#api-shapekind) | What one row is. |
| `estimate` | `number` \| `null` | Estimated number of rows, or `null` while it is unknown. |

#### Returns

`string`

For example `Add 2.100 places to map`, or `Add to map` when the estimate is unknown.

***

<a id="api-arealevellabel"></a>

### areaLevelLabel()

```ts
function areaLevelLabel(level): string;
```

Label of an area level.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `level` | `string` | `provinsi`, `kabkota` or `kecamatan`. |

#### Returns

`string`

`Provinsi`, `Kabupaten/Kota` or `Kecamatan`; any other level is returned as given.

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

<a id="api-areapicker"></a>

### AreaPicker()

```ts
function AreaPicker(props):
  | ReactElement<any, string | JSXElementConstructor<any>>
  | null;
```

Cascading selects that narrow a selection to a region: province, then district or city, then sub-district.

The picker is controlled by a single region code; the code prefixes of 2 and 4 digits identify the parents. Choosing
a region at some level clears the levels below it; choosing the "all" option of a level picks its parent. Each
option shows the row count of its region when the loader supplies one. Names are rendered as text, never as markup.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `props` | [`AreaPickerProps`](#api-areapickerprops) | See [AreaPickerProps](#api-areapickerprops). |

#### Returns

  \| `ReactElement`\<`any`, `string` \| `JSXElementConstructor`\<`any`\>\>
  \| `null`

The picker element, or `null` when `levels` has no known level.

***

<a id="api-categorypicker"></a>

### CategoryPicker()

```ts
function CategoryPicker(props): ReactElement;
```

Picks categories, and optionally brands, from grouped lists.

Categories are listed in collapsible groups; a group opens by default when it is the first or holds a pick. Brands
of the picked categories appear in a second row and narrow the selection further; picking none keeps every brand.
At least one category is expected: the picker says so while none is picked, and [validateSelection](#api-validateselection) reports
it. Titles are rendered as text, never as markup.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `props` | [`CategoryPickerProps`](#api-categorypickerprops) | See [CategoryPickerProps](#api-categorypickerprops). |

#### Returns

`ReactElement`

The picker element.

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

<a id="api-formatdate"></a>

### formatDate()

```ts
function formatDate(epochSeconds): string;
```

Formats a point in time as a long Indonesian calendar date in the Asia/Jakarta time zone.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `epochSeconds` | `number` | Unix time in seconds. |

#### Returns

`string`

For example `2 November 2026`; an empty string when the input is not a positive finite number.

***

<a id="api-formatestimate"></a>

### formatEstimate()

```ts
function formatEstimate(kind, count): string;
```

Formats a row estimate with the noun that fits the geometry kind, using Indonesian digit grouping.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `kind` | [`ShapeKind`](#api-shapekind) | What one row is. |
| `count` | `number` | Estimated number of rows. Negative and non-finite values count as `0`; fractions are rounded down. |

#### Returns

`string`

For example `2.100 places`, `1 place` or `514 areas`.

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

<a id="api-librarydatasetstatus"></a>

### LibraryDatasetStatus()

```ts
function LibraryDatasetStatus(props): ReactElement;
```

Status panel of a dataset that comes from a catalogue: a status bar and the actions that change or refresh it.

Shows `Ready`, `Loading…`, a failure with a Retry button, or `Access ended <date>` with a pointer to the user's
Klikpeta contact. After access ended the data that was already loaded stays, but Refresh and the change buttons are
disabled and Open in Library is hidden. No query text or connection detail is ever shown.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `props` | [`LibraryDatasetStatusProps`](#api-librarydatasetstatusprops) | See [LibraryDatasetStatusProps](#api-librarydatasetstatusprops). |

#### Returns

`ReactElement`

The panel element.

***

<a id="api-libraryitemdetail"></a>

### LibraryItemDetail()

```ts
function LibraryItemDetail(props): ReactElement;
```

Detail view of one catalogue item: description, a grid of facts, the column names and the source credit, with room
for pickers, a "Preview sample" button and one primary action.

Facts with no value are left out. The row count is shown with Indonesian digit grouping and the update date as a long
Indonesian date. Nothing about how the data is stored or queried is shown.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `props` | [`LibraryItemDetailProps`](#api-libraryitemdetailprops) | See [LibraryItemDetailProps](#api-libraryitemdetailprops). |

#### Returns

`ReactElement`

The detail element.

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

***

<a id="api-tabname"></a>

### tabName()

```ts
function tabName(
   title,
   pickedTitles,
   areaName): string;
```

Names the tab of a dataset made from a selection: `<Title> · <Category or brand> +n · <Area>`.

Only the first picked title is shown; the others are counted. Without picks the middle part is left out.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `title` | `string` | The item title. |
| `pickedTitles` | readonly `string`[] | Titles of the picked brands, or of the picked categories when no brand is picked, in display order. |
| `areaName` | `string` | Name of the picked area; `Indonesia` is used when it is empty (whole coverage). |

#### Returns

`string`

For example `Tempat Usaha · Kedai kopi +1 · Kota Bandung`. Titles are used as given: no escaping is applied or needed for text.

***

<a id="api-validateselection"></a>

### validateSelection()

```ts
function validateSelection(
   item,
   selection,
   estimate): SelectionIssue[];
```

Checks a selection against what its item allows, so a button can be disabled and say why.

Reports at most one issue per rule, in this order: a malformed area code (`area-invalid`); an area level the item
cannot be narrowed to (`area-level`); no category on an item with a taxonomy (`category-required`); an estimate of
zero rows (`empty`); an estimate over the item's row limit (`too-many`). The empty area code (whole coverage) is
always allowed. The estimate rules apply only when `estimate` is a number.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `item` | `Pick`\<[`ItemDetail`](#api-itemdetail), `"areaLevels"` \| `"hasTaxonomy"` \| `"maxRows"`\> | The item's limits: `areaLevels`, `hasTaxonomy` and `maxRows`. |
| `selection` | [`Selection`](#api-selection) | What the user picked. |
| `estimate` | `number` \| `null` | Estimated number of rows, or `null` while it is unknown. |

#### Returns

[`SelectionIssue`](#api-selectionissue)[]

The issues; an empty list means the selection can be added.
