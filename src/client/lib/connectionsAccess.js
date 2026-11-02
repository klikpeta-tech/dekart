/**
 * Only Admins may see the Connections tab or its content.
 * `user.isAdmin` defaults to false until the user stream arrives, so the role
 * is only trusted once the stream is loaded.
 *
 * @param {{ stream?: object | null, isAdmin?: boolean }} user - state.user slice
 * @returns {boolean}
 */
export function canViewConnections (user) {
  return Boolean(user?.stream) && Boolean(user.isAdmin)
}
