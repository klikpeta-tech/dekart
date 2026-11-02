import { LibrarySelection, LibraryShape, QueryJob } from 'dekart-proto/dekart_pb'

const shapeKinds = {
  [LibraryShape.Kind.KIND_POINTS]: 'points',
  [LibraryShape.Kind.KIND_LINES]: 'lines',
  [LibraryShape.Kind.KIND_AREAS]: 'areas',
  [LibraryShape.Kind.KIND_H3]: 'h3'
}

/**
 * Maps a LibraryItem message (toObject form) to the data-library CatalogueItem.
 * An unspecified shape kind is shown as points, the least specific geometry.
 *
 * @param {object} item - LibraryItem message object
 * @param {string[]} [searchTerms] - titles of the item's granted taxonomy categories
 * @returns {import('../dark/data-library/index').CatalogueItem}
 */
export function toCatalogueItem (item, searchTerms = []) {
  return {
    id: item.id,
    title: item.title,
    category: item.category,
    provider: item.provider,
    coverageSummary: item.coverageSummary,
    shape: {
      kind: shapeKinds[item.shape?.kind] || 'points',
      adminLevel: item.shape?.adminLevel,
      h3Resolution: item.shape?.h3Resolution
    },
    rowCount: item.rowCount,
    areaLevels: item.areaLevelsList,
    searchTerms
  }
}

/**
 * Whether the user may open the Data Library from the dataset selector.
 * `user.isViewer` defaults to true until the user stream arrives, so the role
 * is only trusted once the stream is loaded.
 *
 * @param {{ stream?: object | null, isViewer?: boolean }} user - state.user slice
 * @returns {boolean}
 */
export function canBrowseLibrary (user) {
  return Boolean(user?.stream) && !user.isViewer
}

/**
 * Maps a GetLibraryItem response (toObject form) to the data-library ItemDetail.
 *
 * @param {object} item - LibraryItem message object
 * @returns {import('../dark/data-library/index').ItemDetail}
 */
export function toItemDetail (item) {
  return {
    ...toCatalogueItem(item),
    description: item.descriptionMd,
    attribution: item.attribution,
    sourceUpdatedAt: item.sourceUpdatedAt,
    cadence: item.cadence,
    columns: item.columnsList,
    maxRows: item.maxRows,
    hasTaxonomy: item.hasTaxonomy
  }
}

/**
 * Maps a LibraryTaxonomy message (toObject form) to the data-library Taxonomy, in server sort order.
 *
 * @param {object} [taxonomy] - LibraryTaxonomy message object
 * @returns {import('../dark/data-library/index').Taxonomy}
 */
export function toTaxonomy (taxonomy) {
  const bySortOrder = (a, b) => a.sortOrder - b.sortOrder
  return {
    groups: (taxonomy?.groupsList || []).slice().sort(bySortOrder).map(({ id, title }) => ({ id, title })),
    categories: (taxonomy?.categoriesList || []).slice().sort(bySortOrder).map(({ id, groupId, title }) => ({ id, groupId, title })),
    brands: (taxonomy?.brandsList || []).map(({ id, categoryId, title }) => ({ id, categoryId, title }))
  }
}

/**
 * Maps a LibrarySelection message (toObject form) to the data-library Selection.
 *
 * @param {object} [selection] - LibrarySelection message object
 * @returns {import('../dark/data-library/index').Selection}
 */
export function toSelection (selection) {
  return {
    areaCode: selection?.areaCode || '',
    categoryIds: selection?.categoryIdsList || [],
    brandIds: selection?.brandIdsList || []
  }
}

/**
 * Maps a data-library Selection to a LibrarySelection message.
 *
 * @param {import('../dark/data-library/index').Selection} selection
 * @returns {LibrarySelection}
 */
export function toSelectionMessage (selection) {
  const message = new LibrarySelection()
  message.setAreaCode(selection.areaCode)
  message.setCategoryIdsList(selection.categoryIds)
  message.setBrandIdsList(selection.brandIds)
  return message
}

/**
 * The state of a library dataset's status bar, from its job for the current map parameters.
 * A grant that ended wins: the last data stays, but nothing new is loaded.
 *
 * @param {{ accessEnded?: boolean }} binding - Dataset.library message object
 * @param {object | undefined} job - the dataset query's QueryJob for the current parameters hash
 * @param {boolean} pending - a run was sent and the stream has not answered yet
 * @returns {'ready' | 'loading' | 'failed' | 'ended'}
 */
export function libraryDatasetState (binding, job, pending) {
  if (binding.accessEnded) {
    return 'ended'
  }
  if (pending) {
    return 'loading'
  }
  if (job?.jobError) {
    return 'failed'
  }
  const done = job?.jobStatus === QueryJob.JobStatus.JOB_STATUS_DONE ||
    (job?.jobStatus === QueryJob.JobStatus.JOB_STATUS_DONE_LEGACY && job.jobResultId)
  return done ? 'ready' : 'loading'
}
