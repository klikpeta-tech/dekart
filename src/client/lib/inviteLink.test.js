import { describe, it, expect } from 'vitest'
import { buildInviteLink } from './inviteLink'

describe('buildInviteLink', () => {
  it('builds the /workspace/invite/:inviteId URL from the origin', () => {
    expect(buildInviteLink('https://geo.klikpeta.com', 'abc-123'))
      .toBe('https://geo.klikpeta.com/workspace/invite/abc-123')
  })

  it('does not depend on the current page path', () => {
    expect(buildInviteLink('http://localhost:3000', 'x'))
      .toBe('http://localhost:3000/workspace/invite/x')
  })
})
