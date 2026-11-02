import {
  dismissLibraryGrantsError,
  getLibraryTaxonomy,
  libraryGrantChanged,
  libraryGrantsFailed,
  libraryGrantsUpdated,
  libraryTaxonomyLoaded,
  listLibraryGrants,
  revokeLibraryGrant,
  setLibraryGrant,
  setLibraryGrantCategories
} from '../actions/libraryGrants'

const defaultState = {
  rows: [],
  loaded: false,
  error: '',
  savingItemId: '',
  taxonomy: {}
}

export default function libraryGrants (state = defaultState, action) {
  switch (action.type) {
    case listLibraryGrants.name:
      return { ...state, error: '' }
    case libraryGrantsUpdated.name:
      return { ...state, rows: action.rowsList, loaded: true }
    case setLibraryGrant.name:
    case revokeLibraryGrant.name:
    case setLibraryGrantCategories.name:
      return { ...state, error: '', savingItemId: action.itemId }
    case libraryGrantChanged.name:
      return {
        ...state,
        savingItemId: '',
        rows: state.rows.map(row => row.item.id === action.grant.itemId ? { ...row, grant: action.grant } : row)
      }
    case getLibraryTaxonomy.name:
      return { ...state, error: '' }
    case libraryTaxonomyLoaded.name:
      return { ...state, taxonomy: { ...state.taxonomy, [action.itemId]: action.taxonomy } }
    case libraryGrantsFailed.name:
      // a refusal ends the pending save and, for the list call, still marks the tab as loaded
      return { ...state, savingItemId: '', loaded: true, error: action.message }
    case dismissLibraryGrantsError.name:
      return { ...state, error: '' }
    default:
      return state
  }
}
