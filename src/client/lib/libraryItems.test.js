import { describe, expect, it } from 'vitest'
import { LibraryShape } from 'dekart-proto/dekart_pb'
import { canBrowseLibrary, toCatalogueItem } from './libraryItems'

const message = {
  id: 'poi',
  title: 'Tempat & Usaha',
  category: 'Tempat',
  provider: 'Klikpeta',
  coverageSummary: 'Indonesia',
  shape: { kind: LibraryShape.Kind.KIND_H3, adminLevel: '', h3Resolution: 8 },
  rowCount: 2100,
  areaLevelsList: ['provinsi']
}

describe('toCatalogueItem', () => {
  it('maps a library item and its search terms', () => {
    expect(toCatalogueItem(message, ['Apotek'])).toEqual({
      id: 'poi',
      title: 'Tempat & Usaha',
      category: 'Tempat',
      provider: 'Klikpeta',
      coverageSummary: 'Indonesia',
      shape: { kind: 'h3', adminLevel: '', h3Resolution: 8 },
      rowCount: 2100,
      areaLevels: ['provinsi'],
      searchTerms: ['Apotek']
    })
  })

  it('shows an unspecified shape as points', () => {
    expect(toCatalogueItem({ ...message, shape: undefined }).shape.kind).toBe('points')
  })
})

describe('canBrowseLibrary', () => {
  it('is for editors and admins once the user stream arrived', () => {
    expect(canBrowseLibrary({ stream: {}, isViewer: false })).toBe(true)
    expect(canBrowseLibrary({ stream: {}, isViewer: true })).toBe(false)
    expect(canBrowseLibrary({ stream: null, isViewer: false })).toBe(false)
  })
})
