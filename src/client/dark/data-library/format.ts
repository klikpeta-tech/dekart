import type { ItemShape, ShapeKind } from './types'

const countFormat = new Intl.NumberFormat('id-ID')

const kindLabels: Record<ShapeKind, string> = {
  points: 'Points',
  lines: 'Lines',
  areas: 'Areas',
  h3: 'H3 hexagons'
}

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
