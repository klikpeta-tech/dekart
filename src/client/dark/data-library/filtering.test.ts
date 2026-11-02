import { describe, expect, it } from 'vitest'
import {
  areaLevelOfCode, countItemsByCategory, emptyLibraryQuery, filterCatalogueItems, formatCount, shapeKindsOf, shapeLabel
} from './index'
import type { CatalogueItem } from './index'

function item (id: string, title: string, category: string, extra: Partial<CatalogueItem> = {}): CatalogueItem {
  return {
    id,
    title,
    category,
    provider: 'BPS',
    coverageSummary: 'Indonesia',
    shape: { kind: 'areas', adminLevel: 'kabkota' },
    rowCount: 10,
    areaLevels: ['provinsi', 'kabkota'],
    ...extra
  }
}

const items = [
  item('poi', 'Tempat & Usaha Indonesia', 'Tempat', { shape: { kind: 'points' }, areaLevels: ['provinsi', 'kabkota', 'kecamatan'], searchTerms: ['Apotek', 'Restoran'] }),
  item('pop', 'Penduduk 2020', 'Demografi'),
  item('bound', 'Batas Kecamatan 2020', 'Batas wilayah', { areaLevels: ['kecamatan'] }),
  item('road', 'Jalan Nasional', 'Infrastruktur', { shape: { kind: 'lines' }, provider: 'Kementerian PUPR', areaLevels: ['provinsi'] }),
  item('hex', 'Kepadatan H3', 'Demografi', { shape: { kind: 'h3', h3Resolution: 8 } })
]
const ids = (list: CatalogueItem[]): string[] => list.map(i => i.id)

describe('filterCatalogueItems', () => {
  it('orders by category then title without a search', () => {
    expect(ids(filterCatalogueItems(items, emptyLibraryQuery()))).toEqual(['bound', 'hex', 'pop', 'road', 'poi'])
  })

  it('does not modify its input', () => {
    const copy = [...items]
    filterCatalogueItems(items, emptyLibraryQuery())
    expect(items).toEqual(copy)
  })

  it('matches place categories supplied as search terms', () => {
    expect(ids(filterCatalogueItems(items, { ...emptyLibraryQuery(), search: 'apotek' }))).toEqual(['poi'])
  })

  it('ignores case and accents and requires every word', () => {
    expect(ids(filterCatalogueItems(items, { ...emptyLibraryQuery(), search: 'TÉMPAT usaha' }))).toEqual(['poi'])
    expect(filterCatalogueItems(items, { ...emptyLibraryQuery(), search: 'tempat jalan' })).toEqual([])
  })

  it('searches provider and category as well as title', () => {
    expect(ids(filterCatalogueItems(items, { ...emptyLibraryQuery(), search: 'pupr' }))).toEqual(['road'])
    expect(ids(filterCatalogueItems(items, { ...emptyLibraryQuery(), search: 'batas wilayah' }))).toEqual(['bound'])
  })

  it('ranks title prefix, then title match, then other fields', () => {
    const ranked = [
      item('c', 'Zona', 'Alpha', { provider: 'Kota Data' }),
      item('b', 'Peta Kota', 'Alpha'),
      item('a', 'Kota Besar', 'Alpha')
    ]
    expect(ids(filterCatalogueItems(ranked, { ...emptyLibraryQuery(), search: 'kota' }))).toEqual(['a', 'b', 'c'])
  })

  it('filters by category and by shape kind', () => {
    expect(ids(filterCatalogueItems(items, { ...emptyLibraryQuery(), category: 'Demografi' }))).toEqual(['hex', 'pop'])
    expect(ids(filterCatalogueItems(items, { ...emptyLibraryQuery(), shape: 'lines' }))).toEqual(['road'])
  })

  it('keeps only items that can be narrowed to the level of the area code', () => {
    expect(ids(filterCatalogueItems(items, emptyLibraryQuery('31')))).toEqual(['hex', 'pop', 'road', 'poi'])
    expect(ids(filterCatalogueItems(items, emptyLibraryQuery('3171')))).toEqual(['hex', 'pop', 'poi'])
    expect(ids(filterCatalogueItems(items, emptyLibraryQuery('3171010')))).toEqual(['bound', 'poi'])
  })

  it('does not filter by an invalid area code', () => {
    expect(filterCatalogueItems(items, emptyLibraryQuery('abc'))).toHaveLength(items.length)
    expect(filterCatalogueItems(items, emptyLibraryQuery('317'))).toHaveLength(items.length)
  })
})

describe('countItemsByCategory', () => {
  it('counts per category and ignores the category in the query', () => {
    const counts = countItemsByCategory(items, { ...emptyLibraryQuery(), category: 'Demografi' })
    expect(counts).toEqual([
      { category: 'Batas wilayah', count: 1 },
      { category: 'Demografi', count: 2 },
      { category: 'Infrastruktur', count: 1 },
      { category: 'Tempat', count: 1 }
    ])
  })

  it('applies search, shape and area and leaves out empty categories', () => {
    expect(countItemsByCategory(items, { ...emptyLibraryQuery('3171010'), search: 'tempat' })).toEqual([{ category: 'Tempat', count: 1 }])
    expect(countItemsByCategory(items, { ...emptyLibraryQuery(), shape: 'h3' })).toEqual([{ category: 'Demografi', count: 1 }])
  })
})

describe('helpers', () => {
  it('maps area code length to level', () => {
    expect([areaLevelOfCode('31'), areaLevelOfCode('3171'), areaLevelOfCode('3171010'), areaLevelOfCode('3'), areaLevelOfCode('31a1'), areaLevelOfCode('')])
      .toEqual(['provinsi', 'kabkota', 'kecamatan', undefined, undefined, undefined])
  })

  it('lists shape kinds present in fixed order', () => {
    expect(shapeKindsOf(items)).toEqual(['points', 'lines', 'areas', 'h3'])
    expect(shapeKindsOf([items[3]])).toEqual(['lines'])
  })

  it('formats counts with Indonesian grouping', () => {
    expect(formatCount(2100)).toBe('2.100')
    expect(formatCount(2100000)).toBe('2.100.000')
    expect(formatCount(-5)).toBe('0')
    expect(formatCount(Number.NaN)).toBe('0')
  })

  it('labels shapes', () => {
    expect(shapeLabel({ kind: 'points' })).toBe('Points')
    expect(shapeLabel({ kind: 'areas', adminLevel: 'kabkota' })).toBe('Areas · Kabupaten/Kota')
    expect(shapeLabel({ kind: 'areas', adminLevel: 'desa' })).toBe('Areas · desa')
    expect(shapeLabel({ kind: 'h3', h3Resolution: 8 })).toBe('H3 hexagons · res 8')
  })
})
