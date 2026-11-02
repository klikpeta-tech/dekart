// Matches the /workspace/invite/:inviteId route in App.jsx and the link in the invite email
export function buildInviteLink (origin, inviteId) {
  return `${origin}/workspace/invite/${inviteId}`
}
