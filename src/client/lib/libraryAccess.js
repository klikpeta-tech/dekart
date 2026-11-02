/**
 * Only workspace Admins manage Data Library access.
 * `user.isAdmin` defaults to false until the user stream arrives, so the role
 * is only trusted once the stream is loaded.
 *
 * @param {{ stream?: object | null, isAdmin?: boolean }} user - state.user slice
 * @returns {boolean}
 */
export function canManageLibraryAccess (user) {
  return Boolean(user?.stream) && Boolean(user.isAdmin)
}
