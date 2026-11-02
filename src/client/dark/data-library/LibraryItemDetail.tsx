import type { ReactElement, ReactNode } from 'react'
import { ArrowLeftOutlined } from '@ant-design/icons'
import { formatCount, formatDate, shapeLabel } from './format'
import styles from './LibraryDetail.module.css'
import type { ItemDetail, SampleTable, SelectionIssue } from './types'

/** Props of {@link LibraryItemDetail}. */
export interface LibraryItemDetailProps {
  /** The item to describe. All text is rendered as text, never as markup. */
  item: ItemDetail
  /** Pickers or other controls, rendered between the column list and the action row. */
  children?: ReactNode
  /** Reasons the current selection cannot be added. Shown as a list above the action row. */
  issues?: readonly SelectionIssue[]
  /** An error to show above the action row, for example the text of a refused command. Empty for none. */
  notice?: string
  /** The primary action. Without it no primary button is shown. */
  primary?: {
    /** Button text. */
    label: string
    /** True while the action cannot be taken. */
    disabled: boolean
    /** Called when the button is pressed while enabled. */
    onClick: () => void
  }
  /** Called when "Preview sample" is pressed. Without it the button is not shown. */
  onPreviewSample?: () => void
  /** True while "Preview sample" is disabled, for example because the selection is not valid. */
  previewDisabled?: boolean
  /** The previewed rows, or `null` for none. Shown as a table. */
  sample?: SampleTable | null
  /** Called when the back button is pressed. Without it no back button is shown. */
  onBack?: () => void
}

function paragraphs (text: string): string[] {
  return text.split(/\n{2,}/).map(part => part.trim()).filter(part => part !== '')
}

/**
 * Detail view of one catalogue item: description, a grid of facts, the column names and the source credit, with room
 * for pickers, a "Preview sample" button and one primary action.
 *
 * Facts with no value are left out. The row count is shown with Indonesian digit grouping and the update date as a long
 * Indonesian date. Nothing about how the data is stored or queried is shown.
 *
 * @param props - See {@link LibraryItemDetailProps}.
 * @returns The detail element.
 */
export function LibraryItemDetail ({ item, children, issues = [], notice = '', primary, onPreviewSample, previewDisabled = false, sample = null, onBack }: LibraryItemDetailProps): ReactElement {
  const facts: Array<[string, string]> = [
    ['Provider', item.provider],
    ['Category', item.category],
    ['Shape', shapeLabel(item.shape)],
    ['Rows', formatCount(item.rowCount)],
    ['Coverage', item.coverageSummary],
    ['Source updated', formatDate(item.sourceUpdatedAt)],
    ['Refresh cadence', item.cadence]
  ]
  return (
    <div className={styles.detail}>
      <div className={styles.header}>
        {onBack !== undefined && (
          <button type='button' className={styles.back} onClick={onBack} aria-label='Back'>
            <ArrowLeftOutlined />
          </button>
        )}
        <h2 className={styles.title}>{item.title}</h2>
      </div>
      <div>{paragraphs(item.description).map((text, index) => <p key={index} className={styles.paragraph}>{text}</p>)}</div>
      <dl className={styles.meta}>
        {facts.filter(([, value]) => value !== '').map(([label, value]) => (
          <div key={label} className={styles.fact}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
      {item.columns.length > 0 && (
        <div>
          <h3 className={styles.sectionTitle}>Columns</h3>
          <ul className={styles.columns}>
            {item.columns.map(column => <li key={column} className={styles.column}>{column}</li>)}
          </ul>
        </div>
      )}
      {item.attribution !== '' && <p className={styles.attribution}>{item.attribution}</p>}
      {children}
      {sample !== null && (
        <div className={styles.sample}>
          <table aria-label='Sample rows'>
            <thead><tr>{sample.columns.map(column => <th key={column}>{column}</th>)}</tr></thead>
            <tbody>
              {sample.rows.map((row, index) => <tr key={index}>{row.map((cell, cellIndex) => <td key={cellIndex}>{cell}</td>)}</tr>)}
            </tbody>
          </table>
        </div>
      )}
      {issues.length > 0 && (
        <ul className={styles.issues} role='alert'>
          {issues.map(issue => <li key={issue.code}>{issue.message}</li>)}
        </ul>
      )}
      {notice !== '' && <div className={styles.notice} role='alert'>{notice}</div>}
      <div className={styles.actions}>
        {primary !== undefined && (
          <button type='button' className={styles.primary} disabled={primary.disabled} onClick={() => primary.onClick()}>{primary.label}</button>
        )}
        {onPreviewSample !== undefined && (
          <button type='button' className={styles.secondary} disabled={previewDisabled} onClick={() => onPreviewSample()}>Preview sample</button>
        )}
      </div>
    </div>
  )
}
