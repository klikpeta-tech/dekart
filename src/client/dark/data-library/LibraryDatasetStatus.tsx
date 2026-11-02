import type { ReactElement } from 'react'
import { formatDate } from './format'
import styles from './LibraryDetail.module.css'

/** Props of {@link LibraryDatasetStatus}. */
export interface LibraryDatasetStatusProps {
  /** Name of the dataset, shown as the heading. Rendered as text, never as markup. */
  title: string
  /** `ready` when the data is loaded, `loading` while it is being fetched, `failed` after an error, `ended` when access to the item has ended. `ended` wins over what the data itself is doing. */
  state: 'ready' | 'loading' | 'failed' | 'ended'
  /** Error text shown next to the failed state. Rendered as text, never as markup. */
  error?: string
  /** When access ended, in Unix seconds. `0` or omitted when unknown; the date is then left out. */
  accessEndedAt?: number
  /** True when the user may change the selection. Hides the two change buttons otherwise. */
  canChange: boolean
  /** True when the user may refresh the data. Hides the refresh button otherwise. */
  canRefresh: boolean
  /** True when the dataset has categories, so the "Change category" button is shown. */
  hasCategories: boolean
  /** Called when "Change category" is pressed. */
  onChangeCategory: () => void
  /** Called when "Change area" is pressed. */
  onChangeArea: () => void
  /** Called when "Refresh" is pressed. */
  onRefresh: () => void
  /** Called when "Retry" is pressed in the failed state. */
  onRetry: () => void
  /** Called when "Open in Library" is pressed. Without it the button is not shown. */
  onOpenLibrary?: () => void
}

/**
 * Status panel of a dataset that comes from a catalogue: a status bar and the actions that change or refresh it.
 *
 * Shows `Ready`, `Loading…`, a failure with a Retry button, or `Access ended <date>` with a pointer to the user's
 * Klikpeta contact. After access ended the data that was already loaded stays, but Refresh and the change buttons are
 * disabled. No query text or connection detail is ever shown.
 *
 * @param props - See {@link LibraryDatasetStatusProps}.
 * @returns The panel element.
 */
export function LibraryDatasetStatus ({ title, state, error = '', accessEndedAt = 0, canChange, canRefresh, hasCategories, onChangeCategory, onChangeArea, onRefresh, onRetry, onOpenLibrary }: LibraryDatasetStatusProps): ReactElement {
  const ended = state === 'ended'
  const date = formatDate(accessEndedAt)
  const bar = { ready: styles.barReady, loading: styles.barLoading, failed: styles.barFailed, ended: styles.barEnded }[state]
  return (
    <div className={styles.status}>
      <h2 className={styles.title}>{title}</h2>
      <div className={`${styles.bar} ${bar}`} role='status'>
        <span className={styles.barText}>
          {state === 'ready' && 'Ready'}
          {state === 'loading' && 'Loading…'}
          {state === 'failed' && `Failed${error === '' ? '' : `: ${error}`}`}
          {ended && `Access ended${date === '' ? '' : ` ${date}`}. Your Klikpeta contact can extend it.`}
        </span>
        {state === 'failed' && canRefresh && <button type='button' className={styles.link} onClick={onRetry}>Retry</button>}
      </div>
      <div className={styles.actions}>
        {canChange && hasCategories && <button type='button' className={styles.secondary} disabled={ended} onClick={onChangeCategory}>Change category</button>}
        {canChange && <button type='button' className={styles.secondary} disabled={ended} onClick={onChangeArea}>Change area</button>}
        {canRefresh && <button type='button' className={styles.secondary} disabled={ended || state === 'loading'} onClick={onRefresh}>Refresh</button>}
        {onOpenLibrary !== undefined && <button type='button' className={styles.secondary} onClick={onOpenLibrary}>Open in Library</button>}
      </div>
    </div>
  )
}
