import { useEffect, useRef, useState } from 'react'
import type { ReactElement } from 'react'
import { areaLevelOfCode } from './filterItems'
import { areaLevelLabel, formatCount } from './format'
import styles from './LibraryDetail.module.css'
import type { AreaNode } from './types'

/** Props of {@link AreaPicker}. */
export interface AreaPickerProps {
  /** Area levels the item can be narrowed to: any of `provinsi`, `kabkota`, `kecamatan`. Selects are shown down to the deepest of them; nothing is shown when empty. */
  levels: readonly string[]
  /** The picked region code of 2, 4 or 7 digits, or empty for the whole coverage. Any other value is treated as empty. */
  value: string
  /** Called with the complete next region code, or an empty string for the whole coverage. */
  onChange: (code: string) => void
  /** Loads the regions directly below a region code; the empty code loads the top level. Called once per parent code, and again after a failure when the parent changes back. */
  loadAreas: (parentCode: string) => Promise<readonly AreaNode[]>
  /** Called with the display name of the picked region whenever it becomes known, and with an empty string for the whole coverage. */
  onNameChange?: (name: string) => void
}

const levelOrder = ['provinsi', 'kabkota', 'kecamatan']
const codeLengths = [2, 4, 7]

type Loaded = Record<string, readonly AreaNode[] | 'loading' | 'failed'>

function listOf (options: Loaded[string] | undefined): readonly AreaNode[] {
  return typeof options === 'object' ? options : []
}

/**
 * Cascading selects that narrow a selection to a region: province, then district or city, then sub-district.
 *
 * The picker is controlled by a single region code; the code prefixes of 2 and 4 digits identify the parents. Choosing
 * a region at some level clears the levels below it; choosing the "all" option of a level picks its parent. Each
 * option shows the row count of its region when the loader supplies one. Names are rendered as text, never as markup.
 *
 * @param props - See {@link AreaPickerProps}.
 * @returns The picker element, or `null` when `levels` has no known level.
 */
export function AreaPicker ({ levels, value, onChange, loadAreas, onNameChange }: AreaPickerProps): ReactElement | null {
  const [loaded, setLoaded] = useState<Loaded>({})
  const requested = useRef(new Set<string>())
  const alive = useRef(true)
  const loader = useRef(loadAreas)
  loader.current = loadAreas
  const code = areaLevelOfCode(value) === undefined ? '' : value
  const deepest = Math.max(-1, ...levels.map(level => levelOrder.indexOf(level)))
  const shown = codeLengths.slice(0, deepest + 1)
  // The region whose children fill each select; undefined while the select above it has no pick.
  const parents = shown.map((_, index) => index === 0 ? '' : code.length >= codeLengths[index - 1] ? code.slice(0, codeLengths[index - 1]) : undefined)

  const parentKey = parents.filter((parent): parent is string => parent !== undefined).join('|')
  // Loaded lists are kept per parent, so a late answer is stored unless the picker is gone.
  useEffect(() => {
    alive.current = true
    return () => { alive.current = false }
  }, [])
  useEffect(() => {
    if (shown.length === 0) return
    for (const parent of parentKey.split('|')) {
      if (requested.current.has(parent)) continue
      requested.current.add(parent)
      setLoaded(current => ({ ...current, [parent]: 'loading' }))
      loader.current(parent).then(
        areas => { if (alive.current) setLoaded(current => ({ ...current, [parent]: areas })) },
        () => {
          requested.current.delete(parent)
          if (alive.current) setLoaded(current => ({ ...current, [parent]: 'failed' }))
        }
      )
    }
  }, [parentKey, shown.length])

  const pickedParent = code.length === 7 ? code.slice(0, 4) : code.length === 4 ? code.slice(0, 2) : ''
  const pickedOptions = loaded[pickedParent]
  const pickedName = code === '' ? '' : listOf(pickedOptions).find(area => area.code === code)?.name
  useEffect(() => {
    if (pickedName !== undefined) onNameChange?.(pickedName)
  }, [pickedName, onNameChange])

  if (shown.length === 0) return null
  return (
    <div className={styles.picker}>
      <div className={styles.selects}>
        {shown.map((length, index) => {
          const parent = parents[index]
          const options = parent === undefined ? undefined : loaded[parent]
          const selected = code.length >= length ? code.slice(0, length) : ''
          const label = areaLevelLabel(levelOrder[index])
          return (
            <label key={length} className={styles.selectLabel}>
              {label}
              <select
                aria-label={label}
                value={selected}
                disabled={options === 'loading' || options === undefined || options === 'failed'}
                onChange={event => onChange(event.target.value === '' ? parent ?? '' : event.target.value)}
              >
                <option value=''>{index === 0 ? 'All of Indonesia' : `All ${label}`}</option>
                {listOf(options).map(area => (
                  <option key={area.code} value={area.code}>
                    {area.rowCount === undefined ? area.name : `${area.name} (${formatCount(area.rowCount)})`}
                  </option>
                ))}
              </select>
              {options === 'failed' && <span role='alert'>Could not load {label}.</span>}
            </label>
          )
        })}
      </div>
    </div>
  )
}
