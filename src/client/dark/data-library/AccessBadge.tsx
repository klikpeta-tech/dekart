import type { ReactElement } from 'react'
import styles from './LibraryBrowser.module.css'

/** Props of {@link AccessBadge}. */
export interface AccessBadgeProps {
  /** `available` for an item that can be added, `added` for one that is already in use. */
  state: 'available' | 'added'
  /** Text for the `added` state. Defaults to `In use`. Rendered as text, never as markup. */
  addedLabel?: string
}

/**
 * Small pill that tells whether an item is available or already in use.
 *
 * @param props - See {@link AccessBadgeProps}.
 * @returns A `span` whose text is `Available` or the added label.
 */
export function AccessBadge ({ state, addedLabel = 'In use' }: AccessBadgeProps): ReactElement {
  return (
    <span className={state === 'added' ? styles.badgeAdded : styles.badgeAvailable}>
      {state === 'added' ? addedLabel : 'Available'}
    </span>
  )
}
