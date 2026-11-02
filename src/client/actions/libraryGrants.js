import {
  GetLibraryTaxonomyRequest,
  ListLibraryGrantsRequest,
  RevokeLibraryGrantRequest,
  SetLibraryGrantCategoriesRequest,
  SetLibraryGrantRequest
} from 'dekart-proto/dekart_pb'
import { Dekart } from 'dekart-proto/dekart_pb_service'
import { grpcCall } from './grpc'

// libraryGrantsFailed keeps the server's refusal for the tab to show instead of a page-level error.
export function libraryGrantsFailed (itemId, err) {
  return { type: libraryGrantsFailed.name, itemId, message: err.message }
}

export function libraryGrantsUpdated (rowsList) {
  return { type: libraryGrantsUpdated.name, rowsList }
}

export function libraryGrantChanged (grant) {
  return { type: libraryGrantChanged.name, grant }
}

export function libraryTaxonomyLoaded (itemId, taxonomy) {
  return { type: libraryTaxonomyLoaded.name, itemId, taxonomy }
}

export function dismissLibraryGrantsError () {
  return { type: dismissLibraryGrantsError.name }
}

// listLibraryGrants loads the whole catalogue with the current workspace's grants.
export function listLibraryGrants () {
  return (dispatch) => {
    dispatch({ type: listLibraryGrants.name })
    dispatch(grpcCall(
      Dekart.ListLibraryGrants,
      new ListLibraryGrantsRequest(),
      (response) => dispatch(libraryGrantsUpdated(response.rowsList)),
      (err) => dispatch(libraryGrantsFailed('', err))
    ))
  }
}

// setLibraryGrant grants an item until endsOn (YYYY-MM-DD) or changes its end date.
export function setLibraryGrant (itemId, endsOn, categoryIds) {
  return (dispatch) => {
    dispatch({ type: setLibraryGrant.name, itemId })
    const request = new SetLibraryGrantRequest()
    request.setItemId(itemId)
    request.setEndsOn(endsOn)
    request.setCategoryIdsList(categoryIds)
    dispatch(grpcCall(
      Dekart.SetLibraryGrant,
      request,
      (response) => dispatch(libraryGrantChanged(response.grant)),
      (err) => dispatch(libraryGrantsFailed(itemId, err))
    ))
  }
}

// revokeLibraryGrant ends access to an item now; the end date is kept.
export function revokeLibraryGrant (itemId) {
  return (dispatch) => {
    dispatch({ type: revokeLibraryGrant.name, itemId })
    const request = new RevokeLibraryGrantRequest()
    request.setItemId(itemId)
    dispatch(grpcCall(
      Dekart.RevokeLibraryGrant,
      request,
      (response) => dispatch(libraryGrantChanged(response.grant)),
      (err) => dispatch(libraryGrantsFailed(itemId, err))
    ))
  }
}

// setLibraryGrantCategories replaces the granted categories of an item that has a taxonomy.
export function setLibraryGrantCategories (itemId, categoryIds) {
  return (dispatch) => {
    dispatch({ type: setLibraryGrantCategories.name, itemId })
    const request = new SetLibraryGrantCategoriesRequest()
    request.setItemId(itemId)
    request.setCategoryIdsList(categoryIds)
    dispatch(grpcCall(
      Dekart.SetLibraryGrantCategories,
      request,
      (response) => dispatch(libraryGrantChanged(response.grant)),
      (err) => dispatch(libraryGrantsFailed(itemId, err))
    ))
  }
}

// getLibraryTaxonomy loads the whole category taxonomy of an item.
export function getLibraryTaxonomy (itemId) {
  return (dispatch) => {
    dispatch({ type: getLibraryTaxonomy.name, itemId })
    const request = new GetLibraryTaxonomyRequest()
    request.setItemId(itemId)
    dispatch(grpcCall(
      Dekart.GetLibraryTaxonomy,
      request,
      (response) => dispatch(libraryTaxonomyLoaded(itemId, response.taxonomy)),
      (err) => dispatch(libraryGrantsFailed(itemId, err))
    ))
  }
}
