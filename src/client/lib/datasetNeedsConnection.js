import { isSystemConnectionID } from '../actions/connection'
import { isDuckDBQuery } from './duckdb/constants'

// A SQL dataset has lost its connection when the map was copied across workspaces: the server blanks the
// connection on purpose, so the person copying must choose one of their own. A blank connection is the
// system connection, which is only a valid SQL source where connections are not user defined or in the playground.
// Library datasets keep their grant binding off the browser and are never blank here.
export function datasetNeedsConnection (dataset, query, { userDefinedConnection, isPlayground }) {
  if (!userDefinedConnection || isPlayground || !dataset || dataset.library || !query) {
    return false
  }
  return Boolean(dataset.queryId) && !isDuckDBQuery(query) && isSystemConnectionID(dataset.connectionId)
}
