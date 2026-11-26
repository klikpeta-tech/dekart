import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { del, get, post, put } from './api'

const WORKSPACE_HEADER = 'X-Dekart-Workspace-Id'

class FakeXhr {
  constructor () {
    this.headers = {}
    FakeXhr.last = this
  }

  open (method, url) {
    this.method = method
    this.url = url
  }

  setRequestHeader (name, value) {
    this.headers[name] = value
  }

  getAllResponseHeaders () {
    return 'content-type: text/csv\r\n'
  }

  send () {
    this.status = 200
    this.response = new ArrayBuffer(0)
    this.onload()
  }
}

describe('get', () => {
  beforeEach(() => {
    vi.stubGlobal('XMLHttpRequest', FakeXhr)
    window.XMLHttpRequest = FakeXhr
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends the selected workspace so the server does not fall back to the newest workspace', async () => {
    await get('/dataset-source/d/s.csv', { access_token: 't' }, null, null, null, 'report-1', 'user@example.com', 'workspace-1')

    expect(FakeXhr.last.url).toBe('/api/v1/dataset-source/d/s.csv')
    expect(FakeXhr.last.headers[WORKSPACE_HEADER]).toBe('workspace-1')
    expect(FakeXhr.last.headers['X-Dekart-Report-Id']).toBe('report-1')
  })

  it('omits the workspace header when no workspace is selected', async () => {
    await get('/dataset-source/d/s.csv', { access_token: 't' })

    expect(WORKSPACE_HEADER in FakeXhr.last.headers).toBe(false)
  })
})

describe('JSON and raw requests', () => {
  let fetchMock

  beforeEach(() => {
    fetchMock = vi.fn().mockImplementation(async () => new window.Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    window.fetch = fetchMock
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const sentHeaders = () => fetchMock.mock.calls[0][1].headers

  it('send the selected workspace on post, put, raw put and del', async () => {
    const calls = [
      () => post('/x', {}, null, 'workspace-1'),
      () => put('/x', {}, null, 'application/json', 'workspace-1'),
      () => put('/x', new Blob(['a']), null, 'application/octet-stream', 'workspace-1'),
      () => del('/x', null, 'workspace-1')
    ]
    for (const call of calls) {
      fetchMock.mockClear()
      await call()
      expect(sentHeaders()[WORKSPACE_HEADER]).toBe('workspace-1')
    }
  })

  it('omit the workspace header when no workspace is selected', async () => {
    await post('/x', {})

    expect(WORKSPACE_HEADER in sentHeaders()).toBe(false)
  })
})
