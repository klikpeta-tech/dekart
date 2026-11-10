/** Geometry kind of one row of a catalogue item. */
export type ShapeKind = 'points' | 'lines' | 'areas' | 'h3'

/** What one row of a catalogue item is. */
export interface ItemShape {
  /** Geometry kind. */
  kind: ShapeKind
  /** For `areas`: the administrative level the rows describe: `provinsi`, `kabkota` or `kecamatan`. */
  adminLevel?: string
  /** For `h3`: the H3 resolution of the cells. */
  h3Resolution?: number
}

/** One browsable catalogue entry. Every field is display or filter data; ids are opaque lookup keys. */
export interface CatalogueItem {
  /** Opaque, unique id. Passed back unchanged to `onSelectItem`. */
  id: string
  /** Display title. The year, when relevant, is part of the title. */
  title: string
  /** Display category used for the category chips. Compared exactly, so equal categories must be spelled equally. */
  category: string
  /** Organisation that provides the data. Searchable. */
  provider: string
  /** Short description of the covered region. Searchable. */
  coverageSummary: string
  /** What one row is. */
  shape: ItemShape
  /** Number of rows in the whole item. Never negative. `0` means unknown and is not shown. */
  rowCount: number
  /** Area levels the item can be narrowed to: any of `provinsi`, `kabkota`, `kecamatan`. */
  areaLevels: string[]
  /** Extra searchable words that are not shown, for example the titles of sub-categories inside the item. */
  searchTerms?: string[]
}

/** The filter state of a browser. */
export interface LibraryQuery {
  /** Free text. Words are matched case- and accent-insensitively; every word must match. Empty means no text filter. */
  search: string
  /** Exact {@link CatalogueItem.category}, or empty for all categories. */
  category: string
  /** Geometry kind, or empty for all kinds. */
  shape: ShapeKind | ''
  /** Region code of 2, 4 or 7 digits, or empty for no region filter. Any other value is treated as empty. */
  areaCode: string
}

/** A selectable region in the browser's area select. */
export interface AreaOption {
  /** Region code of 2, 4 or 7 digits. */
  code: string
  /** Display name. */
  name: string
}

/** Item count of one category under a query. */
export interface CategoryCount {
  /** The category as spelled on the items. */
  category: string
  /** How many items of that category match the other filters. */
  count: number
}

/** A catalogue item with everything the detail view shows. */
export interface ItemDetail extends CatalogueItem {
  /** Description as CommonMark Markdown. Raw HTML in it is shown as text, never as markup; images are not rendered; only web, mail and phone links stay links. */
  description: string
  /** Credit line for the data source. Empty when there is none. */
  attribution: string
  /** When the source data was last updated, in Unix seconds. `0` when unknown. */
  sourceUpdatedAt: number
  /** How often the data is refreshed, as display text. Empty when unknown. */
  cadence: string
  /** Names of the columns a row has. */
  columns: string[]
  /** Largest selection that can be added, in rows. `0` means no limit. */
  maxRows: number
  /** True when rows carry categories (and optional brands) that must be picked. */
  hasTaxonomy: boolean
}

/** What is picked from an item: where, and which categories and brands. */
export interface Selection {
  /** Region code of 2, 4 or 7 digits, or empty for the whole coverage. */
  areaCode: string
  /** Ids of the picked categories. Required (at least one) for items with a taxonomy. */
  categoryIds: string[]
  /** Ids of the picked brands. Optional; each belongs to one of the picked categories. */
  brandIds: string[]
}

/** One region offered by the {@link AreaPicker}. */
export interface AreaNode {
  /** Region code of 2, 4 or 7 digits. */
  code: string
  /** Display name. */
  name: string
  /** Rows of the item inside this region. Shown next to the name when present. */
  rowCount?: number
}

/** A group of categories. */
export interface TaxonomyGroup {
  /** Opaque, unique id. */
  id: string
  /** Display title. */
  title: string
}

/** One pickable category. */
export interface TaxonomyCategory {
  /** Opaque, unique id. */
  id: string
  /** Id of the {@link TaxonomyGroup} it belongs to. */
  groupId: string
  /** Display title. */
  title: string
}

/** One pickable brand inside a category. */
export interface TaxonomyBrand {
  /** Opaque, unique id. */
  id: string
  /** Id of the {@link TaxonomyCategory} it belongs to. */
  categoryId: string
  /** Display title. */
  title: string
}

/** The categories a caller may pick from, in display order. */
export interface Taxonomy {
  /** Groups, in display order. */
  groups: TaxonomyGroup[]
  /** Categories, in display order within their group. */
  categories: TaxonomyCategory[]
  /** Brands, in display order. */
  brands: TaxonomyBrand[]
}

/** Why a selection cannot be added. */
export interface SelectionIssue {
  /** Machine-readable reason: `area-invalid`, `area-level`, `category-required`, `empty` or `too-many`. */
  code: 'area-invalid' | 'area-level' | 'category-required' | 'empty' | 'too-many'
  /** Display text that tells the user what to change. */
  message: string
}

/** A few rows of an item, for a preview. All cells are text. */
export interface SampleTable {
  /** Column names. */
  columns: string[]
  /** Rows of cell text, in column order. */
  rows: string[][]
}
