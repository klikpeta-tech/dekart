import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../lib/api', () => ({ get: vi.fn() }))

const { get } = await import('../lib/api')
const { downloadDataset } = await import('./dataset')

describe('downloadDataset', () => {
  beforeEach(() => {
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
