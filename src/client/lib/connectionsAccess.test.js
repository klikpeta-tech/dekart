import { describe, it, expect } from 'vitest'
import { canViewConnections } from './connectionsAccess'

describe('canViewConnections', () => {
  it('hides Connections from a Viewer', () => {
    expect(canViewConnections({ stream: {}, isViewer: true, isAdmin: false })).toBe(false)
  })

  it('hides Connections from an Editor', () => {
    expect(canViewConnections({ stream: {}, isViewer: false, isAdmin: false })).toBe(false)
  })

  it('keeps Connections for an Admin', () => {
    expect(canViewConnections({ stream: {}, isViewer: false, isAdmin: true })).toBe(true)
  })

  it('does not trust the isAdmin default before the user stream is loaded', () => {
    expect(canViewConnections({ stream: null, isAdmin: true })).toBe(false)
    expect(canViewConnections(undefined)).toBe(false)
  })
})
