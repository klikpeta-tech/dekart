import { describe, expect, it } from 'vitest'
import { calendarDay, endingSoonDays, filterRows, grantRowState } from './index'
import type { Grant, RowFilter } from './index'

const NOW = 1_800_000_000
const DAY = 86400
const active = (days: number): Grant => ({ status: 'active', endsOn: '2027-01-31', endsAt: NOW + days * DAY })

describe('grantRowState', () => {
  it('shows a grant with plenty of time as granted', () => {
    expect(grantRowState(active(endingSoonDays + 1), NOW))
      .toEqual({ kind: 'granted', label: `Ends in ${endingSoonDays + 1} days`, daysLeft: endingSoonDays + 1 })
  })

  it('shows a grant at the ending-soon boundary as ending soon', () => {
    expect(grantRowState(active(endingSoonDays), NOW).kind).toBe('ending-soon')
  })

  it('rounds a partial last day up and uses the singular', () => {
    const grant = { ...active(0), endsAt: NOW + 60 }
    expect(grantRowState(grant, NOW)).toEqual({ kind: 'ending-soon', label: 'Ends in 1 day', daysLeft: 1 })
  })

  it('shows an ended grant with its end date', () => {
    const grant: Grant = { status: 'ended', endsOn: '2026-11-12', endsAt: NOW - DAY }
    expect(grantRowState(grant, NOW)).toEqual({ kind: 'ended', label: 'Ended 2026-11-12', daysLeft: 0 })
  })

  it('does not trust an active status whose end has passed', () => {
    const grant: Grant = { status: 'active', endsOn: '2026-11-12', endsAt: NOW }
    expect(grantRowState(grant, NOW).kind).toBe('ended')
  })

  it('shows a never-granted row as not granted', () => {
    expect(grantRowState({ status: 'none', endsOn: '', endsAt: 0 }, NOW))
      .toEqual({ kind: 'not-granted', label: 'Not granted', daysLeft: 0 })
  })

  it('shows a revoked grant as not granted even when an end date is kept', () => {
    const grant: Grant = { status: 'revoked', endsOn: '2027-01-31', endsAt: NOW + 30 * DAY }
    expect(grantRowState(grant, NOW).kind).toBe('not-granted')
  })
})

describe('filterRows', () => {
  const ended: Grant = { status: 'ended', endsOn: '2026-01-01', endsAt: NOW - DAY }
  const never: Grant = { status: 'none', endsOn: '', endsAt: 0 }
  const rows = [
    { title: 'Batas Provinsi', category: 'Administrasi', grant: active(60) },
    { title: 'Tempat & Usaha', category: 'POI', grant: active(3) },
    { title: 'Jalan Nasional', category: 'Transportasi', grant: ended },
    { title: 'Kepadatan Penduduk', category: 'Demografi', grant: never }
  ]
  const titles = (filter: RowFilter, search = ''): string[] =>
    filterRows(rows, { filter, search }, NOW).map(row => row.title)

  it('keeps every row for all', () => {
    expect(titles('all')).toHaveLength(4)
  })

  it('keeps rows with current access for granted, including ending soon', () => {
    expect(titles('granted')).toEqual(['Batas Provinsi', 'Tempat & Usaha'])
  })

  it('keeps only rows ending soon', () => {
    expect(titles('ending-soon')).toEqual(['Tempat & Usaha'])
  })

  it('keeps ended and never-granted rows for not granted', () => {
    expect(titles('not-granted')).toEqual(['Jalan Nasional', 'Kepadatan Penduduk'])
  })

  it('searches title and category case-insensitively and combines with the chip', () => {
    expect(titles('all', '  poi ')).toEqual(['Tempat & Usaha'])
    expect(titles('granted', 'jalan')).toEqual([])
  })
})

describe('calendarDay', () => {
  it('returns the day in the given offset, crossing midnight UTC', () => {
    expect(calendarDay(Date.UTC(2026, 10, 1, 20) / 1000, 420)).toBe('2026-11-02')
    expect(calendarDay(Date.UTC(2026, 10, 1, 20) / 1000, 0)).toBe('2026-11-01')
  })
})
