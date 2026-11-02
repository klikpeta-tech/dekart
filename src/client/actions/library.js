import {
  AddLibraryDatasetRequest,
  CountLibrarySelectionRequest,
  GetLibraryItemRequest,
  GetLibrarySampleRequest,
  ListLibraryAreasRequest,
  ListLibraryItemsRequest,
  UpdateLibraryDatasetSelectionRequest
} from 'dekart-proto/dekart_pb'
import { Dekart } from 'dekart-proto/dekart_pb_service'
import { grpcCall } from './grpc'
import { toSelectionMessage } from '../lib/libraryItems'

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

// Calls a library RPC and resolves with { response } or { error, code }, so the caller can show the refusal in place.
// Nothing is dispatched to the global error banner.
function libraryCommand (dispatch, method, request) {
  return new Promise((resolve) => {
    dispatch(grpcCall(method, request, response => resolve({ response }), (err) => {
      resolve({ error: err.message, code: err.code })
      return null
    }))
  })
}

// Loads one item with the part of its taxonomy that is granted to the workspace.
export function fetchLibraryItem (itemId) {
  return (dispatch) => {
    const request = new GetLibraryItemRequest()
    request.setItemId(itemId)
    return libraryCommand(dispatch, Dekart.GetLibraryItem, request)
  }
}

// Loads the areas directly below parentCode (the provinces when empty), with the item's row count in each.
export function fetchLibraryAreas (parentCode, itemId) {
  return (dispatch) => {
    const request = new ListLibraryAreasRequest()
    request.setParentCode(parentCode)
    request.setItemId(itemId)
    return libraryCommand(dispatch, Dekart.ListLibraryAreas, request)
  }
}

// Estimates the rows of a selection, with counts per category and brand.
export function countLibrarySelection (itemId, selection) {
  return (dispatch) => {
    const request = new CountLibrarySelectionRequest()
    request.setItemId(itemId)
    request.setSelection(toSelectionMessage(selection))
    return libraryCommand(dispatch, Dekart.CountLibrarySelection, request)
  }
}

// Loads the first rows of a selection for the preview.
export function fetchLibrarySample (itemId, selection) {
  return (dispatch) => {
    const request = new GetLibrarySampleRequest()
    request.setItemId(itemId)
    request.setSelection(toSelectionMessage(selection))
    return libraryCommand(dispatch, Dekart.GetLibrarySample, request)
  }
}

// Binds the empty dataset to a library item. The new dataset and its run arrive through the report stream.
export function addLibraryDataset (datasetId, itemId, selection) {
  return (dispatch) => {
    const request = new AddLibraryDatasetRequest()
    request.setDatasetId(datasetId)
    request.setItemId(itemId)
    request.setSelection(toSelectionMessage(selection))
    return libraryCommand(dispatch, Dekart.AddLibraryDataset, request)
  }
}

// Replaces the selection of a library dataset. expectedVersion is the binding version the client last saw;
// a stale version is refused and the client reconciles from the report stream.
export function updateLibrarySelection (datasetId, expectedVersion, selection) {
  return (dispatch) => {
    const request = new UpdateLibraryDatasetSelectionRequest()
    request.setDatasetId(datasetId)
    request.setExpectedVersion(expectedVersion)
    request.setSelection(toSelectionMessage(selection))
    return libraryCommand(dispatch, Dekart.UpdateLibraryDatasetSelection, request)
  }
}
