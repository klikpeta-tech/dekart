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
  /** Number of rows in the whole item. Never negative. */
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
