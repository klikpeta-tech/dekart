import { combineReducers } from 'redux'
import { libraryItemsLoaded, libraryProvincesLoaded } from '../actions/library'

// loaded stays false when the library is unavailable, which hides the Library row.
function loaded (state = false, action) {
  return action.type === libraryItemsLoaded.name ? true : state
}

function items (state = [], action) {
  return action.type === libraryItemsLoaded.name ? action.items : state
}

function catalogueSize (state = 0, action) {
  return action.type === libraryItemsLoaded.name ? action.catalogueSize : state
}

function searchTerms (state = {}, action) {
  return action.type === libraryItemsLoaded.name ? action.searchTerms : state
}

function provinces (state = [], action) {
  return action.type === libraryProvincesLoaded.name ? action.provinces : state
}

export default combineReducers({ loaded, items, catalogueSize, searchTerms, provinces })
