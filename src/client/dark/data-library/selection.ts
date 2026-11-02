import { areaLevelOfCode } from './filterItems'
import { areaLevelLabel, formatCount, formatEstimate } from './format'
import type { ItemDetail, Selection, SelectionIssue, ShapeKind } from './types'

/**
 * Checks a selection against what its item allows, so a button can be disabled and say why.
 *
 * Reports at most one issue per rule, in this order: a malformed area code (`area-invalid`); an area level the item
 * cannot be narrowed to (`area-level`); no category on an item with a taxonomy (`category-required`); an estimate of
 * zero rows (`empty`); an estimate over the item's row limit (`too-many`). The empty area code (whole coverage) is
 * always allowed. The estimate rules apply only when `estimate` is a number.
 *
 * @param item - The item's limits: `areaLevels`, `hasTaxonomy` and `maxRows`.
 * @param selection - What the user picked.
 * @param estimate - Estimated number of rows, or `null` while it is unknown.
 * @returns The issues; an empty list means the selection can be added.
 */
export function validateSelection (item: Pick<ItemDetail, 'areaLevels' | 'hasTaxonomy' | 'maxRows'>, selection: Selection, estimate: number | null): SelectionIssue[] {
  const issues: SelectionIssue[] = []
  if (selection.areaCode !== '') {
    const level = areaLevelOfCode(selection.areaCode)
    if (level === undefined) {
      issues.push({ code: 'area-invalid', message: 'The area code must have 2, 4 or 7 digits.' })
    } else if (!item.areaLevels.includes(level)) {
      const allowed = item.areaLevels.map(areaLevelLabel).join(' or ')
      issues.push({ code: 'area-level', message: allowed === '' ? 'This dataset cannot be narrowed to an area.' : `This dataset can only be narrowed to ${allowed}.` })
    }
  }
  if (item.hasTaxonomy && selection.categoryIds.length === 0) {
    issues.push({ code: 'category-required', message: 'Pick at least one category.' })
  }
  if (estimate !== null && estimate <= 0) {
    issues.push({ code: 'empty', message: 'No rows match this selection.' })
  } else if (estimate !== null && item.maxRows > 0 && estimate > item.maxRows) {
    issues.push({ code: 'too-many', message: `${formatCount(estimate)} rows is over the limit of ${formatCount(item.maxRows)}. Narrow the area or the categories.` })
  }
  return issues
}

/**
 * Label of the button that adds a selection to a map.
 *
 * @param kind - What one row is.
 * @param estimate - Estimated number of rows, or `null` while it is unknown.
 * @returns For example `Add 2.100 places to map`, or `Add to map` when the estimate is unknown.
 */
export function addButtonLabel (kind: ShapeKind, estimate: number | null): string {
  return estimate === null ? 'Add to map' : `Add ${formatEstimate(kind, estimate)} to map`
}

/**
 * Names the tab of a dataset made from a selection: `<Title> · <Category or brand> +n · <Area>`.
 *
 * Only the first picked title is shown; the others are counted. Without picks the middle part is left out.
 *
 * @param title - The item title.
 * @param pickedTitles - Titles of the picked brands, or of the picked categories when no brand is picked, in display order.
 * @param areaName - Name of the picked area; `Indonesia` is used when it is empty (whole coverage).
 * @returns For example `Tempat Usaha · Kedai kopi +1 · Kota Bandung`. Titles are used as given: no escaping is applied or needed for text.
 */
export function tabName (title: string, pickedTitles: readonly string[], areaName: string): string {
  const parts = [title]
  if (pickedTitles.length === 1) parts.push(pickedTitles[0])
  else if (pickedTitles.length > 1) parts.push(`${pickedTitles[0]} +${pickedTitles.length - 1}`)
  parts.push(areaName === '' ? 'Indonesia' : areaName)
  return parts.join(' · ')
}
