import { describe, expect, it } from 'vitest'
import { QueryExecutionEngine } from 'dekart-proto/dekart_pb'
import { datasetNeedsConnection } from './datasetNeedsConnection'

const workspace = { userDefinedConnection: true, isPlayground: false }
const sqlQuery = { id: 'q1', executionEngine: QueryExecutionEngine.QUERY_EXECUTION_ENGINE_CONNECTION }

describe('datasetNeedsConnection', () => {
  it('asks for a connection when a SQL dataset has none', () => {
    expect(datasetNeedsConnection({ queryId: 'q1', connectionId: '' }, sqlQuery, workspace)).toBe(true)
    expect(datasetNeedsConnection({ queryId: 'q1', connectionId: '00000000-0000-0000-0000-000000000000' }, sqlQuery, workspace)).toBe(true)
  })

  it('accepts a dataset that has a connection', () => {
    expect(datasetNeedsConnection({ queryId: 'q1', connectionId: 'c1' }, sqlQuery, workspace)).toBe(false)
  })

  it('leaves datasets that never use a connection alone', () => {
    const duckDBQuery = { id: 'q1', executionEngine: QueryExecutionEngine.QUERY_EXECUTION_ENGINE_DUCKDB }
    expect(datasetNeedsConnection({ queryId: 'q1', connectionId: '' }, duckDBQuery, workspace)).toBe(false)
    expect(datasetNeedsConnection({ fileId: 'f1', connectionId: '' }, undefined, workspace)).toBe(false)
    expect(datasetNeedsConnection({ connectionId: '' }, undefined, workspace)).toBe(false)
    expect(datasetNeedsConnection({ queryId: 'q1', connectionId: '', library: true }, sqlQuery, workspace)).toBe(false)
  })

  it('keeps the system connection where it is the SQL source', () => {
    const dataset = { queryId: 'q1', connectionId: '' }
    expect(datasetNeedsConnection(dataset, sqlQuery, { ...workspace, userDefinedConnection: false })).toBe(false)
    expect(datasetNeedsConnection(dataset, sqlQuery, { ...workspace, isPlayground: true })).toBe(false)
  })
})
