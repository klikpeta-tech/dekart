import { describe, expect, it } from 'vitest'
import { LibraryShape, QueryJob } from 'dekart-proto/dekart_pb'
import { canBrowseLibrary, libraryDatasetState, toCatalogueItem } from './libraryItems'

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

describe('libraryDatasetState', () => {
  const { JOB_STATUS_RUNNING, JOB_STATUS_DONE, JOB_STATUS_DONE_LEGACY } = QueryJob.JobStatus
  const binding = { accessEnded: false }

  it('is loading without a job or while a run is pending', () => {
    expect(libraryDatasetState(binding, undefined, false)).toBe('loading')
    expect(libraryDatasetState(binding, { jobStatus: JOB_STATUS_DONE }, true)).toBe('loading')
    expect(libraryDatasetState(binding, { jobStatus: JOB_STATUS_RUNNING }, false)).toBe('loading')
  })

  it('is ready when the job is done, and failed when it has an error', () => {
    expect(libraryDatasetState(binding, { jobStatus: JOB_STATUS_DONE }, false)).toBe('ready')
    expect(libraryDatasetState(binding, { jobStatus: JOB_STATUS_DONE_LEGACY, jobResultId: 'r' }, false)).toBe('ready')
    expect(libraryDatasetState(binding, { jobStatus: JOB_STATUS_DONE_LEGACY }, false)).toBe('loading')
    expect(libraryDatasetState(binding, { jobError: 'boom' }, false)).toBe('failed')
  })

  it('shows access ended whatever the job did', () => {
    expect(libraryDatasetState({ accessEnded: true }, { jobStatus: JOB_STATUS_DONE }, false)).toBe('ended')
    expect(libraryDatasetState({ accessEnded: true }, { jobError: 'x' }, true)).toBe('ended')
  })
})
