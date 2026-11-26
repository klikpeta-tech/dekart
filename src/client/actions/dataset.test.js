import React from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadDataset, processDownloadError } from './dataset'
import { get } from '../lib/api'
import { grpcCall } from './grpc'
import { failDuckDBSource } from './duckdb'
import { info, setError } from './message'

vi.mock('./grpc', () => ({ grpcCall: vi.fn(() => ({ type: 'grpcCall' })) }))
vi.mock('./duckdb', () => ({
  failDuckDBSource: vi.fn(() => ({ type: 'failDuckDBSource' })),
  registerDuckDBFileSource: vi.fn(),
  registerDuckDBSource: vi.fn(),
  removeDuckDBSource: vi.fn()
}))
vi.mock('../lib/api', () => ({ get: vi.fn() }))
vi.mock('./message', () => ({
  info: vi.fn(() => ({ type: 'info' })),
  setError: vi.fn(() => ({ type: 'setError' })),
  warn: vi.fn(),
  success: vi.fn()
}))

describe('missing query result recovery', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('React', React)
  })
  afterEach(() => vi.unstubAllGlobals())

  it.each([
    ['runnable query', 410, 'query-1', true, false, true],
    ['execution pending', 410, 'query-1', true, true, false],
    ['query cannot run', 410, 'query-1', false, false, false],
    ['missing uploaded file', 404, '', false, false, false]
  ])('%s', async (_, status, queryId, canRun, pending, expectRun) => {
    const controller = new AbortController()
    const dataset = { id: 'dataset-1', queryId }
    const state = {
      dataset: { downloading: [{ dataset, controller }] },
      queryStatus: { 'query-1': { canRun, queryText: 'select {{value}}' } },
      queryExecutionsPending: { 'query-1': pending },
      queryJobs: [],
      queryParams: { hash: 'current-params', list: [], url: '?value=42' }
    }
    const actions = []
    const dispatch = action => {
      if (typeof action === 'function') return action(dispatch, () => state)
      actions.push(action)
      return action
    }
    const error = { status, message: 'Dataset source missing' }

    await processDownloadError(error, dataset, 'Query 1', false, controller)(dispatch, () => state)

    expect(grpcCall).toHaveBeenCalledTimes(expectRun ? 1 : 0)
    expect(info).toHaveBeenCalledTimes(expectRun ? 1 : 0)
    if (expectRun) {
      const request = grpcCall.mock.calls[0][1]
      expect(request.getQueryId()).toBe('query-1')
      expect(request.getQueryText()).toBe('select {{value}}')
      expect(request.getQueryParamsValues()).toBe('?value=42')
      expect(actions.some(action => action.type === 'queryExecutionStarted')).toBe(true)
    }
    if (status === 404) {
      expect(failDuckDBSource).toHaveBeenCalledOnce()
      expect(setError).toHaveBeenCalledWith(error)
    }
  })
})

describe('downloadDataset', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    get.mockReset()
    get.mockResolvedValue(new window.Response('a,b\n1,2\n'))
  })

  // A shared map is opened by a member whose newest workspace is not the map's workspace;
  // without the selected workspace the server resolves the wrong one and answers 404.
  it('downloads the dataset source in the selected workspace', async () => {
    const dataset = { id: 'dataset-1' }
    const state = {
      files: [],
      dataset: { list: [dataset], downloading: [] },
      token: { access_token: 'token' },
      report: { id: 'report-1' },
      reportStatus: { snapshotMode: false },
      user: { claimEmailCookie: null, loginHint: 'david@example.com', preferredWorkspaceId: 'workspace-b7' }
    }

    await downloadDataset(dataset, 'source-1', 'csv', [])(vi.fn(), () => state)

    expect(get).toHaveBeenCalledTimes(1)
    const [endpoint, , , , , reportId, loginHint, workspaceId] = get.mock.calls[0]
    expect(endpoint).toBe('/dataset-source/dataset-1/source-1.csv')
    expect(reportId).toBe('report-1')
    expect(loginHint).toBe('david@example.com')
    expect(workspaceId).toBe('workspace-b7')
  })
})
