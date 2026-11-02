/**
 * Browse a catalogue of data items: filter and order them by text, category, geometry kind and region level,
 * and render the result as a searchable list with category chips and availability badges.
 * @packageDocumentation
 */
export { AccessBadge } from './AccessBadge'
export type { AccessBadgeProps } from './AccessBadge'
export { areaLevelOfCode, countItemsByCategory, emptyLibraryQuery, filterCatalogueItems, shapeKindsOf } from './filterItems'
export { formatCount, shapeKindLabel, shapeLabel } from './format'
export { LibraryBrowser } from './LibraryBrowser'
export type { LibraryBrowserProps } from './LibraryBrowser'
export { LibraryItemRow } from './LibraryItemRow'
export type { LibraryItemRowProps } from './LibraryItemRow'
export type { AreaOption, CatalogueItem, CategoryCount, ItemShape, LibraryQuery, ShapeKind } from './types'
