import { GetLibraryItemRequest, ListLibraryAreasRequest, ListLibraryItemsRequest } from 'dekart-proto/dekart_pb'
import { Dekart } from 'dekart-proto/dekart_pb_service'
import { grpcCall } from './grpc'

export function libraryItemsLoaded (items, catalogueSize, searchTerms) {
  return { type: libraryItemsLoaded.name, items, catalogueSize, searchTerms }
}

export function libraryProvincesLoaded (provinces) {
  return { type: libraryProvincesLoaded.name, provinces }
}

// Calls a library RPC and resolves with the response, or null when it fails.
// Errors are not shown: a server without a library answers Unavailable and the Library row then stays hidden.
function libraryCall (dispatch, method, request) {
  return new Promise((resolve) => {
    dispatch(grpcCall(method, request, resolve, () => resolve(null)))
  })
}

// Loads the items granted to the workspace, plus the titles of each taxonomy item's granted categories for search.
export function loadLibraryItems () {
  return async (dispatch) => {
    const response = await libraryCall(dispatch, Dekart.ListLibraryItems, new ListLibraryItemsRequest())
    if (!response) {
      return
    }
    const searchTerms = {}
    await Promise.all(response.itemsList.filter(item => item.hasTaxonomy).map(async (item) => {
      const request = new GetLibraryItemRequest()
      request.setItemId(item.id)
      const detail = await libraryCall(dispatch, Dekart.GetLibraryItem, request)
      searchTerms[item.id] = (detail?.taxonomy?.categoriesList || []).map(category => category.title)
    }))
    dispatch(libraryItemsLoaded(response.itemsList, response.catalogueSize, searchTerms))
  }
}

// Loads the provinces for the browser's area select.
export function loadLibraryProvinces () {
  return async (dispatch) => {
    const response = await libraryCall(dispatch, Dekart.ListLibraryAreas, new ListLibraryAreasRequest())
    if (response) {
      dispatch(libraryProvincesLoaded(response.areasList))
    }
  }
}
