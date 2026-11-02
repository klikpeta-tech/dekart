/**
 * Viewers must not see the Connections tab or its content.
 * `user.isViewer` defaults to true until the user stream arrives, so the role
 * is only trusted once the stream is loaded.
 *
 * @param {{ stream?: object | null, isViewer?: boolean }} user - state.user slice
 * @returns {boolean}
 */
export function canViewConnections (user) {
  return Boolean(user?.stream) && !user.isViewer
}
