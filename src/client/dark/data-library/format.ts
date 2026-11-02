import type { ItemShape, ShapeKind } from './types'

const countFormat = new Intl.NumberFormat('id-ID')

const kindLabels: Record<ShapeKind, string> = {
  points: 'Points',
  lines: 'Lines',
  areas: 'Areas',
  h3: 'H3 hexagons'
}

const kindNouns: Record<ShapeKind, [string, string]> = {
  points: ['place', 'places'],
  lines: ['line', 'lines'],
  areas: ['area', 'areas'],
  h3: ['cell', 'cells']
}

const dateFormat = new Intl.DateTimeFormat('id-ID', { dateStyle: 'long', timeZone: 'Asia/Jakarta' })

const levelLabels: Record<string, string> = {
  provinsi: 'Provinsi',
  kabkota: 'Kabupaten/Kota',
  kecamatan: 'Kecamatan'
}

/**
 * Formats a whole number with Indonesian digit grouping, for example `2100` becomes `2.100`.
 * Negative and non-finite inputs are formatted as `0`; fractions are rounded down.
 *
 * @param value - The number to format.
 * @returns The formatted text.
 */
export function formatCount (value: number): string {
  return countFormat.format(Number.isFinite(value) && value > 0 ? Math.floor(value) : 0)
}

/**
 * Label of a geometry kind, as used by the shape select.
 *
 * @param kind - The geometry kind.
 * @returns A short English label, for example `Points`.
 */
export function shapeKindLabel (kind: ShapeKind): string {
  return kindLabels[kind]
}

/**
 * Describes what one row is, including the administrative level or H3 resolution when known.
 *
 * @param shape - The item shape.
 * @returns For example `Areas · Kabupaten/Kota` or `H3 hexagons · res 8`. Unknown administrative levels are shown as given.
 */
export function shapeLabel (shape: ItemShape): string {
  const base = kindLabels[shape.kind]
  if (shape.kind === 'areas' && shape.adminLevel !== undefined && shape.adminLevel !== '') return `${base} · ${levelLabels[shape.adminLevel] ?? shape.adminLevel}`
  if (shape.kind === 'h3' && shape.h3Resolution !== undefined && shape.h3Resolution > 0) return `${base} · res ${shape.h3Resolution}`
  return base
}

/**
 * Label of an area level.
 *
 * @param level - `provinsi`, `kabkota` or `kecamatan`.
 * @returns `Provinsi`, `Kabupaten/Kota` or `Kecamatan`; any other level is returned as given.
 */
export function areaLevelLabel (level: string): string {
  return levelLabels[level] ?? level
}

/**
 * Formats a row estimate with the noun that fits the geometry kind, using Indonesian digit grouping.
 *
 * @param kind - What one row is.
 * @param count - Estimated number of rows. Negative and non-finite values count as `0`; fractions are rounded down.
 * @returns For example `2.100 places`, `1 place` or `514 areas`.
 */
export function formatEstimate (kind: ShapeKind, count: number): string {
  const whole = Number.isFinite(count) && count > 0 ? Math.floor(count) : 0
  return `${formatCount(whole)} ${kindNouns[kind][whole === 1 ? 0 : 1]}`
}

/**
 * Formats a point in time as a long Indonesian calendar date in the Asia/Jakarta time zone.
 *
 * @param epochSeconds - Unix time in seconds.
 * @returns For example `2 November 2026`; an empty string when the input is not a positive finite number.
 */
export function formatDate (epochSeconds: number): string {
  return Number.isFinite(epochSeconds) && epochSeconds > 0 ? dateFormat.format(new Date(epochSeconds * 1000)) : ''
}
