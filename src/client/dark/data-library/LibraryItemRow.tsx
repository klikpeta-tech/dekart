import type { ReactElement } from 'react'
import { AccessBadge } from './AccessBadge'
import { formatCount, shapeLabel } from './format'
import styles from './LibraryBrowser.module.css'
import type { CatalogueItem } from './types'

/** Props of {@link LibraryItemRow}. */
export interface LibraryItemRowProps {
  /** The item to show. All text is rendered as text, never as markup. */
  item: CatalogueItem
  /** True when the item is already in use; shows the added badge instead of the available one. */
  added?: boolean
  /** Text of the added badge. Defaults to `In use`. */
  addedLabel?: string
  /** Called with the item id when the row is activated. Without it the row is not interactive. */
  onSelect?: (itemId: string) => void
}

/**
 * Compact row for one catalogue item: title, category, shape, row count, coverage and an {@link AccessBadge}.
 *
 * @param props - See {@link LibraryItemRowProps}.
 * @returns A list item; a button inside it when `onSelect` is given.
 */
export function LibraryItemRow ({ item, added = false, addedLabel, onSelect }: LibraryItemRowProps): ReactElement {
  const body = (
    <>
      <span className={styles.rowMain}>
        <span className={styles.rowTitle}>{item.title}</span>
        <span className={styles.rowMeta}>
          {[item.category, shapeLabel(item.shape), `${formatCount(item.rowCount)} rows`].join(' · ')}
        </span>
        {item.coverageSummary !== '' && <span className={styles.rowMeta}>{item.coverageSummary}</span>}
      </span>
      <AccessBadge state={added ? 'added' : 'available'} addedLabel={addedLabel} />
    </>
  )
  return (
    <li className={styles.rowItem}>
      {onSelect !== undefined
        ? <button type='button' className={styles.row} onClick={() => onSelect(item.id)}>{body}</button>
        : <div className={styles.row}>{body}</div>}
    </li>
  )
}
