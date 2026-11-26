import { describe, expect, it, vi } from 'vitest'
import { QueryExecutionEngine, QueryJob } from 'dekart-proto/dekart_pb'
import { runDuckDBGraph } from './duckdb'
import { get } from '../lib/api'

const runtime = vi.hoisted(() => ({
  nextGeneration: () => 1,
  isCurrent: () => true,
  acquireExecution: async () => () => {},
  discardJobTables: async () => {},
  bindParameters: async () => {},
  registerHTTPSource: async () => {},
  executeNode: async () => ({ totalRows: 0 })
}))

vi.mock('../lib/duckdb/runtime', () => ({ getDuckDBRuntime: () => runtime }))
vi.mock('../lib/api', () => ({ get: vi.fn() }))
vi.mock('@kepler.gl/actions', () => ({ addDataToMap: vi.fn(), replaceDataInMap: vi.fn() }))
vi.mock('../reducers/keplerReducer', () => ({ filterRestore: {} }))
vi.mock('./message', () => ({ setError: vi.fn() }))
vi.mock('./kepler', () => ({
  consumeAutoCreateLayer: vi.fn(),
  keplerDatasetFinishUpdating: vi.fn(),
  keplerDatasetStartUpdating: vi.fn()
}))

describe('runDuckDBGraph', () => {
  // A shared map is opened by a member whose newest workspace is not the map's workspace;
  // without the selected workspace the server resolves the wrong one and answers 404.
  it('downloads HTTP sources in the selected workspace', async () => {
    get.mockResolvedValue(new window.Response('a,b\n1,2\n'))
    const state = {
      files: [],
      dataset: { list: [{ id: 'dataset-1', queryId: 'query-1' }], autoCreateLayerIds: ['dataset-1'] },
      queries: [{ id: 'query-1', executionEngine: QueryExecutionEngine.QUERY_EXECUTION_ENGINE_DUCKDB }],
      queryJobs: [{
        id: 'job-1',
        queryId: 'query-1',
        queryParamsHash: 'hash',
        jobStatus: QueryJob.JobStatus.JOB_STATUS_DONE,
        dependencyRevisionsList: [],
        httpSourcesList: [{ sourceId: 'source-1', extension: 'csv' }]
      }],
      queryParams: { url: '', hash: 'hash' },
      duckDBJobStates: {},
      keplerGl: {},
      token: { access_token: 'token' },
      report: { id: 'report-1', queryParamsList: [] },
      user: { claimEmailCookie: null, loginHint: 'david@example.com', preferredWorkspaceId: 'workspace-b7' }
    }

    await runDuckDBGraph()(vi.fn(), () => state)

    expect(get).toHaveBeenCalledTimes(1)
    const [endpoint, , , , , reportId, loginHint, workspaceId] = get.mock.calls[0]
    expect(endpoint).toBe('/dataset-source/dataset-1/source-1.csv')
    expect(reportId).toBe('report-1')
    expect(loginHint).toBe('david@example.com')
    expect(workspaceId).toBe('workspace-b7')
  })
})
