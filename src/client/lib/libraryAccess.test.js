import { describe, it, expect } from 'vitest'
import { canManageLibraryAccess } from './libraryAccess'

describe('canManageLibraryAccess', () => {
  it('is only open to an Admin', () => {
    expect(canManageLibraryAccess({ stream: {}, isAdmin: true })).toBe(true)
    expect(canManageLibraryAccess({ stream: {}, isAdmin: false })).toBe(false)
  })

  it('does not trust the role before the user stream is loaded', () => {
    expect(canManageLibraryAccess({ stream: null, isAdmin: true })).toBe(false)
    expect(canManageLibraryAccess(undefined)).toBe(false)
  })
})
