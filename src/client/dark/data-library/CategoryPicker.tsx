import type { ReactElement } from 'react'
import { formatCount } from './format'
import styles from './LibraryDetail.module.css'
import type { Taxonomy } from './types'

/** Props of {@link CategoryPicker}. */
export interface CategoryPickerProps {
  /** The categories the user may pick from. Only these are shown; nothing outside them can be picked. */
  taxonomy: Taxonomy
  /** Ids of the picked categories. */
  categoryIds: readonly string[]
  /** Ids of the picked brands. */
  brandIds: readonly string[]
  /** Row counts by category id, shown next to the titles. Categories without an entry show no count. */
  categoryCounts?: Readonly<Record<string, number>>
  /** Row counts by brand id, shown next to the titles. Brands without an entry show no count. */
  brandCounts?: Readonly<Record<string, number>>
  /** Called with the complete next picks. Brands of a category that is no longer picked are dropped. */
  onChange: (next: { categoryIds: string[], brandIds: string[] }) => void
}

function withCount (title: string, count: number | undefined): string {
  return count === undefined ? title : `${title} (${formatCount(count)})`
}

/**
 * Picks categories, and optionally brands, from grouped lists.
 *
 * Categories are listed in collapsible groups; a group opens by default when it is the first or holds a pick. Brands
 * of the picked categories appear in a second row and narrow the selection further; picking none keeps every brand.
 * At least one category is expected: the picker says so while none is picked, and {@link validateSelection} reports
 * it. Titles are rendered as text, never as markup.
 *
 * @param props - See {@link CategoryPickerProps}.
 * @returns The picker element.
 */
export function CategoryPicker ({ taxonomy, categoryIds, brandIds, categoryCounts = {}, brandCounts = {}, onChange }: CategoryPickerProps): ReactElement {
  const toggleCategory = (id: string): void => {
    const next = categoryIds.includes(id) ? categoryIds.filter(other => other !== id) : [...categoryIds, id]
    const keptBrands = brandIds.filter(brandId => {
      const brand = taxonomy.brands.find(candidate => candidate.id === brandId)
      return brand !== undefined && next.includes(brand.categoryId)
    })
    onChange({ categoryIds: next, brandIds: keptBrands })
  }
  const toggleBrand = (id: string): void => {
    onChange({ categoryIds: [...categoryIds], brandIds: brandIds.includes(id) ? brandIds.filter(other => other !== id) : [...brandIds, id] })
  }
  const brands = taxonomy.brands.filter(brand => categoryIds.includes(brand.categoryId))
  return (
    <div className={styles.picker}>
      <div className={styles.hint}>{categoryIds.length === 0 ? 'Pick at least one category.' : `${categoryIds.length} selected`}</div>
      {taxonomy.groups.map((group, index) => {
        const categories = taxonomy.categories.filter(category => category.groupId === group.id)
        const picked = categories.filter(category => categoryIds.includes(category.id)).length
        return (
          <details key={group.id} className={styles.group} open={index === 0 || picked > 0}>
            <summary>{picked > 0 ? `${group.title} (${picked} selected)` : group.title}</summary>
            <div className={styles.categories}>
              {categories.map(category => (
                <label key={category.id} className={styles.check}>
                  <input type='checkbox' checked={categoryIds.includes(category.id)} onChange={() => toggleCategory(category.id)} />
                  {withCount(category.title, categoryCounts[category.id])}
                </label>
              ))}
            </div>
          </details>
        )
      })}
      {brands.length > 0 && (
        <div>
          <h3 className={styles.sectionTitle}>Brands (optional)</h3>
          <div className={styles.brands} role='group' aria-label='Brands'>
            {brands.map(brand => (
              <button key={brand.id} type='button' className={brandIds.includes(brand.id) ? styles.chipOn : styles.chip} aria-pressed={brandIds.includes(brand.id)} onClick={() => toggleBrand(brand.id)}>
                {withCount(brand.title, brandCounts[brand.id])}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
