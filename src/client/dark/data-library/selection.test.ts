import { describe, expect, it } from 'vitest'
import { addButtonLabel, formatDate, formatEstimate, tabName, validateSelection } from './index'
import type { Selection } from './index'

const places = { areaLevels: ['provinsi', 'kabkota', 'kecamatan'], hasTaxonomy: true, maxRows: 1000000 }
const boundaries = { areaLevels: ['kabkota'], hasTaxonomy: false, maxRows: 0 }
const pick = (change: Partial<Selection> = {}): Selection => ({ areaCode: '', categoryIds: [], brandIds: [], ...change })
const codes = (issues: Array<{ code: string }>): string[] => issues.map(issue => issue.code)

describe('validateSelection', () => {
  it('accepts a whole-coverage selection of an item without a taxonomy', () => {
    expect(validateSelection(boundaries, pick(), 514)).toEqual([])
  })

  it('requires a category on items with a taxonomy, and accepts one pick', () => {
    expect(codes(validateSelection(places, pick(), null))).toEqual(['category-required'])
    expect(validateSelection(places, pick({ categoryIds: ['kafe'] }), 2100)).toEqual([])
  })

  it('rejects malformed area codes', () => {
    for (const areaCode of ['3', '327', '32730', '3273010x', 'abcd', '３２']) {
      expect(codes(validateSelection(boundaries, pick({ areaCode }), null))).toEqual(['area-invalid'])
    }
  })

  it('rejects an area level the item cannot be narrowed to and names the allowed one', () => {
    const issues = validateSelection(boundaries, pick({ areaCode: '32' }), null)
    expect(codes(issues)).toEqual(['area-level'])
    expect(issues[0].message).toBe('This dataset can only be narrowed to Kabupaten/Kota.')
    expect(validateSelection(boundaries, pick({ areaCode: '3273' }), null)).toEqual([])
  })

  it('says so when an item cannot be narrowed at all', () => {
    const issues = validateSelection({ ...boundaries, areaLevels: [] }, pick({ areaCode: '32' }), null)
    expect(issues[0].message).toBe('This dataset cannot be narrowed to an area.')
  })

  it('reports an empty selection and one over the row limit, in id-ID format', () => {
    expect(codes(validateSelection(boundaries, pick(), 0))).toEqual(['empty'])
    const [issue] = validateSelection(places, pick({ categoryIds: ['kafe'] }), 1200000)
    expect(issue.code).toBe('too-many')
    expect(issue.message).toBe('1.200.000 rows is over the limit of 1.000.000. Narrow the area or the categories.')
  })

  it('allows exactly the limit and ignores the limit when it is zero', () => {
    expect(validateSelection(places, pick({ categoryIds: ['kafe'] }), 1000000)).toEqual([])
    expect(validateSelection(boundaries, pick(), 99999999)).toEqual([])
  })

  it('collects several issues in rule order', () => {
    expect(codes(validateSelection(places, pick({ areaCode: '3x' }), 0))).toEqual(['area-invalid', 'category-required', 'empty'])
  })
})

describe('estimate and button text', () => {
  it('groups digits the Indonesian way and picks the noun by shape and count', () => {
    expect(formatEstimate('points', 2100)).toBe('2.100 places')
    expect(formatEstimate('points', 1)).toBe('1 place')
    expect(formatEstimate('areas', 1234567)).toBe('1.234.567 areas')
    expect(formatEstimate('h3', 0)).toBe('0 cells')
    expect(formatEstimate('lines', -5)).toBe('0 lines')
    expect(formatEstimate('lines', Number.NaN)).toBe('0 lines')
    expect(formatEstimate('lines', 2.9)).toBe('2 lines')
  })

  it('labels the add button with what it adds', () => {
    expect(addButtonLabel('points', 2100)).toBe('Add 2.100 places to map')
    expect(addButtonLabel('points', null)).toBe('Add to map')
  })

  it('formats dates in Jakarta time', () => {
    // 2026-11-02 18:00 UTC is already 3 November in Jakarta.
    expect(formatDate(Date.UTC(2026, 10, 2, 18) / 1000)).toBe('3 November 2026')
    expect(formatDate(0)).toBe('')
    expect(formatDate(Number.NaN)).toBe('')
  })
})

describe('tabName', () => {
  it('joins title, picks and area', () => {
    expect(tabName('Tempat Usaha', [], 'Kota Bandung')).toBe('Tempat Usaha · Kota Bandung')
    expect(tabName('Tempat Usaha', ['Kedai kopi'], 'Kota Bandung')).toBe('Tempat Usaha · Kedai kopi · Kota Bandung')
    expect(tabName('Tempat Usaha', ['Kedai kopi', 'Kafe', 'Bakery'], 'Kota Bandung')).toBe('Tempat Usaha · Kedai kopi +2 · Kota Bandung')
  })

  it('names the whole coverage Indonesia', () => {
    expect(tabName('Batas Wilayah 2024', [], '')).toBe('Batas Wilayah 2024 · Indonesia')
  })

  it('uses hostile titles as plain text', () => {
    expect(tabName('<b>x</b>', ['"; DROP TABLE'], "O'Brien")).toBe('<b>x</b> · "; DROP TABLE · O\'Brien')
  })
})
