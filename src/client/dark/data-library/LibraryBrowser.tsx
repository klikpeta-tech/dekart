import type { ReactElement } from 'react'
import { ArrowLeftOutlined, SearchOutlined } from '@ant-design/icons'
import { countItemsByCategory, emptyLibraryQuery, filterCatalogueItems, shapeKindsOf } from './filterItems'
import { shapeKindLabel } from './format'
import { LibraryItemRow } from './LibraryItemRow'
import styles from './LibraryBrowser.module.css'
import type { AreaOption, CatalogueItem, LibraryQuery, ShapeKind } from './types'

/** Props of {@link LibraryBrowser}. */
export interface LibraryBrowserProps {
  /** The whole catalogue to browse, or `null` while it loads (skeleton rows are shown). */
  items: readonly CatalogueItem[] | null
  /** Current filter state. The browser is controlled: it renders this and reports changes. */
  query: LibraryQuery
  /** Called with the complete next query whenever the user changes a filter. */
  onQueryChange: (query: LibraryQuery) => void
  /** Regions offered by the area select, besides "All areas". A `query.areaCode` that is not listed is added as its code. */
  areaOptions?: readonly AreaOption[]
  /** Ids of items already in use; they show the added badge. */
  addedItemIds?: readonly string[]
  /** Text of the added badge. Defaults to `In use`. */
  addedLabel?: string
  /** Called with the item id when a row is activated. Without it rows are not interactive. */
  onSelectItem?: (itemId: string) => void
  /** Called when the back button is pressed. Without it no back button is shown. */
  onBack?: () => void
}

function Skeleton (): ReactElement {
  return (
    <ul className={styles.list} aria-busy='true' aria-label='Loading datasets'>
      {[0, 1, 2].map(index => <li key={index} className={styles.skeletonRow} data-testid='library-skeleton' />)}
    </ul>
  )
}

/**
 * Searchable, filterable list of catalogue items with category chips, an area select and a shape select.
 *
 * Filtering and ordering follow {@link filterCatalogueItems}; chip counts follow {@link countItemsByCategory}. The shape
 * select lists only the kinds present in `items`. While `items` is `null` it shows skeleton rows; when nothing matches it
 * shows an empty state with a button that clears every filter.
 *
 * @param props - See {@link LibraryBrowserProps}.
 * @returns The browser element.
 */
export function LibraryBrowser ({ items, query, onQueryChange, areaOptions = [], addedItemIds = [], addedLabel, onSelectItem, onBack }: LibraryBrowserProps): ReactElement {
  const set = (change: Partial<LibraryQuery>): void => onQueryChange({ ...query, ...change })
  const all = items ?? []
  const shown = filterCatalogueItems(all, query)
  const counts = countItemsByCategory(all, query)
  const chips = query.category !== '' && !counts.some(c => c.category === query.category)
    ? [...counts, { category: query.category, count: 0 }]
    : counts
  const total = counts.reduce((sum, c) => sum + c.count, 0)
  const areas = query.areaCode !== '' && !areaOptions.some(a => a.code === query.areaCode)
    ? [...areaOptions, { code: query.areaCode, name: `Area ${query.areaCode}` }]
    : areaOptions

  return (
    <div className={styles.browser}>
      <div className={styles.header}>
        {onBack !== undefined && (
          <button type='button' className={styles.back} onClick={onBack} aria-label='Back'>
            <ArrowLeftOutlined />
          </button>
        )}
        <span className={styles.headerTitle}>Data Library</span>
      </div>
      <label className={styles.search}>
        <SearchOutlined />
        <input
          type='search'
          placeholder='Search datasets and categories'
          aria-label='Search datasets'
          value={query.search}
          onChange={event => set({ search: event.target.value })}
        />
      </label>
      <div className={styles.filters}>
        <label className={styles.filter}>
          <span>Area</span>
          <select aria-label='Area' value={query.areaCode} onChange={event => set({ areaCode: event.target.value })}>
            <option value=''>All areas</option>
            {areas.map(area => <option key={area.code} value={area.code}>{area.name}</option>)}
          </select>
        </label>
        <label className={styles.filter}>
          <span>Shape</span>
          <select aria-label='Shape' value={query.shape} onChange={event => set({ shape: event.target.value as ShapeKind | '' })}>
            <option value=''>All shapes</option>
            {shapeKindsOf(all).map(kind => <option key={kind} value={kind}>{shapeKindLabel(kind)}</option>)}
          </select>
        </label>
      </div>
      {items !== null && chips.length > 0 && (
        <div className={styles.chips} role='group' aria-label='Categories'>
          <button type='button' className={query.category === '' ? styles.chipOn : styles.chip} aria-pressed={query.category === ''} onClick={() => set({ category: '' })}>
            All ({total})
          </button>
          {chips.map(({ category, count }) => (
            <button
              key={category}
              type='button'
              className={query.category === category ? styles.chipOn : styles.chip}
              aria-pressed={query.category === category}
              onClick={() => set({ category })}
            >
              {category} ({count})
            </button>
          ))}
        </div>
      )}
      {items === null
        ? <Skeleton />
        : shown.length > 0
          ? (
            <ul className={styles.list}>
              {shown.map(item => (
                <LibraryItemRow key={item.id} item={item} added={addedItemIds.includes(item.id)} addedLabel={addedLabel} onSelect={onSelectItem} />
              ))}
            </ul>
            )
          : (
            <div className={styles.empty}>
              <div>{all.length === 0 ? 'No datasets are available yet.' : 'No datasets match your filters.'}</div>
              {all.length > 0 && (
                <button type='button' className={styles.clear} onClick={() => onQueryChange(emptyLibraryQuery())}>
                  Clear filters
                </button>
              )}
            </div>
            )}
    </div>
  )
}
