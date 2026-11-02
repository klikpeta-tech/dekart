import type { CatalogueItem, CategoryCount, LibraryQuery, ShapeKind } from './types'

const shapeOrder: ShapeKind[] = ['points', 'lines', 'areas', 'h3']

const levelByCodeLength: Record<number, string> = { 2: 'provinsi', 4: 'kabkota', 7: 'kecamatan' }

// Lowercases and strips accents so that "é" matches "e" and case never matters.
function fold (text: string): string {
  return text.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase()
}

function words (text: string): string[] {
  return fold(text).split(/[^\p{L}\p{N}]+/u).filter(Boolean)
}

/**
 * The query to start from: no text, category or shape filter.
 *
 * @param areaCode - Optional preset region code. Defaults to no region filter.
 * @returns A fresh query.
 */
export function emptyLibraryQuery (areaCode = ''): LibraryQuery {
  return { search: '', category: '', shape: '', areaCode }
}

/**
 * The area level a region code addresses.
 *
 * @param areaCode - A region code.
 * @returns `provinsi` for 2 digits, `kabkota` for 4, `kecamatan` for 7, and `undefined` for anything else, including empty.
 */
export function areaLevelOfCode (areaCode: string): string | undefined {
  return /^\d+$/.test(areaCode) ? levelByCodeLength[areaCode.length] : undefined
}

// The item can be narrowed to the area's level, or no valid area is requested.
function matchesArea (item: CatalogueItem, areaCode: string): boolean {
  const level = areaLevelOfCode(areaCode)
  return level === undefined || item.areaLevels.includes(level)
}

// Every search word appears in a visible field or in the extra search terms.
function matchesSearch (item: CatalogueItem, tokens: string[]): boolean {
  if (tokens.length === 0) return true
  const haystack = fold([item.title, item.category, item.provider, item.coverageSummary, ...(item.searchTerms ?? [])].join(' '))
  return tokens.every(token => haystack.includes(token))
}

function matchesFacets (item: CatalogueItem, query: LibraryQuery, tokens: string[]): boolean {
  return (query.shape === '' || item.shape.kind === query.shape) &&
    matchesArea(item, query.areaCode) &&
    matchesSearch(item, tokens)
}

// 0: the title starts with the search; 1: the title contains every word; 2: matched through other fields.
function searchRank (item: CatalogueItem, query: string, tokens: string[]): number {
  if (tokens.length === 0) return 0
  const title = fold(item.title)
  if (title.startsWith(fold(query).trim())) return 0
  return tokens.every(token => title.includes(token)) ? 1 : 2
}

const collator = new Intl.Collator('id', { sensitivity: 'base' })

/**
 * Filters and orders catalogue items for display. Does not modify the input.
 *
 * Filtering: an item must match the text search, the category, the shape kind and the region. The text search looks at
 * title, category, provider, coverage summary and `searchTerms`. A region code only keeps items whose `areaLevels` include the
 * code's level (2 digits `provinsi`, 4 `kabkota`, 7 `kecamatan`); an invalid code filters nothing.
 *
 * Ordering: without text, by category then title. With text, titles that start with the text first, then titles that contain
 * every word, then matches through other fields; ties by title. Comparison is accent-insensitive, using Indonesian collation.
 *
 * @param items - The catalogue items.
 * @param query - The filter state.
 * @returns The matching items in display order.
 * @example
 * ```ts
 * const shown = filterCatalogueItems(items, { ...emptyLibraryQuery('3171'), search: 'apotek' })
 * ```
 */
export function filterCatalogueItems (items: readonly CatalogueItem[], query: LibraryQuery): CatalogueItem[] {
  const tokens = words(query.search)
  return items
    .filter(item => (query.category === '' || item.category === query.category) && matchesFacets(item, query, tokens))
    .map(item => ({ item, rank: searchRank(item, query.search, tokens) }))
    .sort((a, b) => {
      if (a.rank !== b.rank) return a.rank - b.rank
      const byCategory = tokens.length === 0 ? collator.compare(a.item.category, b.item.category) : 0
      return byCategory !== 0 ? byCategory : collator.compare(a.item.title, b.item.title)
    })
    .map(({ item }) => item)
}

/**
 * Counts items per category for the category chips. The category in the query is ignored, so every chip shows how many
 * items it would show if picked; the text, shape and region filters apply.
 *
 * @param items - The catalogue items.
 * @param query - The filter state.
 * @returns One entry per category that has a matching item, sorted by category. Empty categories are left out.
 */
export function countItemsByCategory (items: readonly CatalogueItem[], query: LibraryQuery): CategoryCount[] {
  const tokens = words(query.search)
  const counts = new Map<string, number>()
  for (const item of items) {
    if (matchesFacets(item, query, tokens)) counts.set(item.category, (counts.get(item.category) ?? 0) + 1)
  }
  return [...counts.entries()]
    .map(([category, count]) => ({ category, count }))
    .sort((a, b) => collator.compare(a.category, b.category))
}

/**
 * The geometry kinds present in a list of items, for the shape select.
 *
 * @param items - The catalogue items.
 * @returns Distinct kinds in the fixed order points, lines, areas, h3.
 */
export function shapeKindsOf (items: readonly CatalogueItem[]): ShapeKind[] {
  return shapeOrder.filter(kind => items.some(item => item.shape.kind === kind))
}
