import { useEffect, useMemo, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { LibraryBrowser, areaLevelOfCode, emptyLibraryQuery } from './dark/data-library/index'
import LibraryDetailPane from './LibraryDetailPane'
import { loadLibraryProvinces } from './actions/library'
import { toCatalogueItem } from './lib/libraryItems'
import styles from './Dataset.module.css'

// Name of the report parameter that holds the map's BPS area code.
const AREA_PARAM = 'kode_wilayah'

// Replaces the dataset selector inside an empty dataset tab with the Data Library browser and the item detail it opens.
export default function LibraryPane ({ datasetId, onBack }) {
  const dispatch = useDispatch()
  const items = useSelector(state => state.library.items)
  const searchTerms = useSelector(state => state.library.searchTerms)
  const provinces = useSelector(state => state.library.provinces)
  const datasets = useSelector(state => state.dataset.list)
  const areaParam = useSelector(state => state.queryParams.values[AREA_PARAM])
  const [itemId, setItemId] = useState('')
  // The map's area setting presets the area select when it is a valid BPS code.
  const [query, setQuery] = useState(() => emptyLibraryQuery(areaLevelOfCode(areaParam || '') ? areaParam : ''))

  useEffect(() => {
    if (provinces.length === 0) {
      dispatch(loadLibraryProvinces())
    }
  }, [dispatch, provinces.length])

  const catalogue = useMemo(
    () => items.map(item => toCatalogueItem(item, searchTerms[item.id])),
    [items, searchTerms]
  )
  const areaOptions = useMemo(
    () => provinces.map(({ code, name }) => ({ code, name })),
    [provinces]
  )
  const addedItemIds = datasets.filter(dataset => dataset.library).map(dataset => dataset.library.itemId)

  if (itemId) {
    return <LibraryDetailPane itemId={itemId} datasetId={datasetId} mode='add' onBack={() => setItemId('')} />
  }

  return (
    <div className={styles.libraryPane}>
      <LibraryBrowser
        items={catalogue}
        query={query}
        onQueryChange={setQuery}
        areaOptions={areaOptions}
        addedItemIds={addedItemIds}
        addedLabel='On this map'
        onSelectItem={setItemId}
        onBack={onBack}
      />
    </div>
  )
}
