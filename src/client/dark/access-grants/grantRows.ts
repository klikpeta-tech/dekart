/** Latest recorded decision for one row, as reported by the source of truth. */
export type GrantStatus = 'none' | 'active' | 'ended' | 'revoked'

/** Time-limited access recorded for one row. */
export interface Grant {
  /** Latest decision: never granted, active, past its end, or withdrawn. */
  status: GrantStatus
  /** Last day of access as `YYYY-MM-DD`, or an empty string when none was ever recorded. */
  endsOn: string
  /** Unix seconds at which access stops, or 0 when none was ever recorded. */
  endsAt: number
}

/** What a row currently looks like to the person managing access. */
export type RowKind = 'granted' | 'ending-soon' | 'ended' | 'not-granted'

/** Display state of one row. */
export interface RowState {
  /** Category of the row's current access. */
  kind: RowKind
  /** Plain display text (`Not granted`, `Ends in 5 days`, `Ended 2026-11-12`). Render it as text, never as markup. */
  label: string
  /** Whole days of access left, rounded up; 0 unless `kind` is `granted` or `ending-soon`. */
  daysLeft: number
}

/** List filter: everything, or one group of row kinds. */
export type RowFilter = 'all' | 'granted' | 'ending-soon' | 'not-granted'

/** A row that can be searched and filtered. */
export interface FilterableRow {
  /** Main text searched. */
  title: string
  /** Secondary text searched. */
  category: string
  /** Access recorded for the row. */
  grant: Grant
}

/** Active access that stops within this many days counts as ending soon. */
export const endingSoonDays = 14

const SECONDS_PER_DAY = 86400

/**
 * Classify one row's access at a given moment.
 *
 * Active access is `granted`, or `ending-soon` when at most {@link endingSoonDays}
 * days remain. Access whose `endsAt` has passed is `ended` even if the status still
 * says active, so a stale status never shows access that is gone. A withdrawn or
 * never-recorded grant is `not-granted`; a withdrawn grant keeps its `endsOn` for the caller.
 *
 * @param grant - Access recorded for the row.
 * @param nowSeconds - Current time in unix seconds, supplied by the caller.
 * @returns The row's kind, label and remaining whole days.
 *
 * @example
 * ```ts
 * grantRowState({ status: 'active', endsOn: '2026-11-12', endsAt: 1_000_000 + 3 * 86400 }, 1_000_000)
 * // { kind: 'ending-soon', label: 'Ends in 3 days', daysLeft: 3 }
 * ```
 */
export function grantRowState (grant: Grant, nowSeconds: number): RowState {
  const isActive = grant.status === 'active' && grant.endsAt > nowSeconds
  if (isActive) {
    const daysLeft = Math.ceil((grant.endsAt - nowSeconds) / SECONDS_PER_DAY)
    return {
      kind: daysLeft <= endingSoonDays ? 'ending-soon' : 'granted',
      label: daysLeft === 1 ? 'Ends in 1 day' : `Ends in ${daysLeft} days`,
      daysLeft
    }
  }
  if (grant.status === 'ended' || grant.status === 'active') {
    return { kind: 'ended', label: `Ended ${grant.endsOn}`, daysLeft: 0 }
  }
  return { kind: 'not-granted', label: 'Not granted', daysLeft: 0 }
}

function matchesFilter (kind: RowKind, filter: RowFilter): boolean {
  if (filter === 'granted') return kind === 'granted' || kind === 'ending-soon'
  if (filter === 'ending-soon') return kind === 'ending-soon'
  if (filter === 'not-granted') return kind === 'not-granted' || kind === 'ended'
  return true
}

/**
 * Keep the rows that match a filter chip and a search text, in their original order.
 *
 * `granted` keeps rows with current access (including ending soon), `ending-soon`
 * keeps only rows whose access stops soon, and `not-granted` keeps rows without
 * current access, including ended ones. Search is a case-insensitive substring match
 * on `title` and `category`; blank search keeps everything.
 *
 * @param rows - Rows to filter; not modified.
 * @param options - The filter chip and the search text.
 * @param nowSeconds - Current time in unix seconds, supplied by the caller.
 * @returns A new array with the matching rows.
 */
export function filterRows<T extends FilterableRow> (
  rows: readonly T[],
  options: {
    /** Group of row kinds to keep. */
    filter: RowFilter
    /** Text to look for in title and category; blank matches all. */
    search: string
  },
  nowSeconds: number
): T[] {
  const needle = options.search.trim().toLowerCase()
  return rows.filter(row => {
    const { kind } = grantRowState(row.grant, nowSeconds)
    const text = `${row.title}\n${row.category}`.toLowerCase()
    return matchesFilter(kind, options.filter) && text.includes(needle)
  })
}

/**
 * The calendar day of a moment in a fixed UTC offset, for bounding date pickers.
 *
 * @param nowSeconds - Moment in unix seconds, supplied by the caller.
 * @param utcOffsetMinutes - Offset of the calendar from UTC in minutes (420 for UTC+7).
 * @returns The day as `YYYY-MM-DD`.
 *
 * @example
 * ```ts
 * calendarDay(Date.UTC(2026, 10, 1, 20) / 1000, 420) // '2026-11-02'
 * ```
 */
export function calendarDay (nowSeconds: number, utcOffsetMinutes: number): string {
  return new Date((nowSeconds + utcOffsetMinutes * 60) * 1000).toISOString().slice(0, 10)
}
