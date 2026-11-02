import { LibraryShape } from 'dekart-proto/dekart_pb'

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
