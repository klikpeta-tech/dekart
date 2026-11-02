import { LibraryGrant } from 'dekart-proto/dekart_pb'

const STATUS_BY_STATE = {
  [LibraryGrant.State.STATE_ACTIVE]: 'active',
  [LibraryGrant.State.STATE_ENDED]: 'ended',
  [LibraryGrant.State.STATE_REVOKED]: 'revoked'
}

// toAccessGrant maps a LibraryGrant (toObject form) to the access-grants library input.
export function toAccessGrant (grant) {
  return {
    status: STATUS_BY_STATE[grant.state] || 'none',
    endsOn: grant.endsOn,
    endsAt: grant.accessEndsAt
  }
}
