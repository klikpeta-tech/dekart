import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useDispatch } from 'react-redux'
import {
  AreaPicker,
  CategoryPicker,
  LibraryItemDetail,
  addButtonLabel,
  formatEstimate,
  tabName,
  validateSelection
} from './dark/data-library/index'
import { addLibraryDataset, countLibrarySelection, fetchLibraryAreas, fetchLibraryItem, fetchLibrarySample, updateLibrarySelection } from './actions/library'
import { toItemDetail, toSelection, toTaxonomy } from './lib/libraryItems'
import { track } from './lib/tracking'
import styles from './Dataset.module.css'

// gRPC code of a compare-and-set refusal: the dataset changed since the client last saw it.
const CODE_CANCELED = 1
const COUNT_DELAY_MS = 250

function counts (list = []) {
  return Object.fromEntries(list.map(({ id, rowCount }) => [id, rowCount]))
}

// Describes one library item and lets an editor pick an area and categories from it.
// mode 'add' binds the empty dataset; 'area' and 'category' change one part of a bound dataset; 'view' only describes the item.
export default function LibraryDetailPane ({ itemId, datasetId, binding, mode, onBack, onDone }) {
  const dispatch = useDispatch()
  const [detail, setDetail] = useState(null)
  const [selection, setSelection] = useState(() => toSelection(binding?.selection))
  const [areaName, setAreaName] = useState('')
  const [estimate, setEstimate] = useState(null)
  const [sample, setSample] = useState(null)
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)
  // Counts sample requests and selection changes, so a reply for an older selection is dropped.
  const sampleRequest = useRef(0)
  const showArea = mode === 'add' || mode === 'area'
  const showCategory = mode === 'add' || mode === 'category'

  useEffect(() => {
    let cancelled = false
    dispatch(fetchLibraryItem(itemId)).then(({ response, error }) => {
      if (cancelled) {
        return
      }
      if (error) {
        setNotice(error)
      } else {
        setDetail({
          item: toItemDetail(response.item),
          taxonomy: toTaxonomy(response.taxonomy),
          // Counts over the whole country, shown only while no area is picked and no estimate exists.
          nationwide: counts(response.taxonomy?.categoriesList)
        })
      }
    })
    return () => { cancelled = true }
  }, [dispatch, itemId])

  const item = detail?.item
  const issues = useMemo(() => item && mode !== 'view' ? validateSelection(item, selection, estimate?.rowCount ?? null) : [], [item, mode, selection, estimate])
  // The estimate is requested once the selection passes the local checks.
  const countable = item && mode !== 'view' && validateSelection(item, selection, null).length === 0

  useEffect(() => {
    setEstimate(null)
    if (!countable) {
      return undefined
    }
    let cancelled = false
    const timer = setTimeout(() => {
      dispatch(countLibrarySelection(itemId, selection)).then(({ response, error }) => {
        if (cancelled) {
          return
        }
        if (error) {
          setNotice(error)
        } else {
          setNotice('')
          setEstimate(response)
        }
      })
    }, COUNT_DELAY_MS)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [dispatch, itemId, countable, selection])

  // Any change of the selection drops the preview and the last refusal, which no longer describe it.
  const changeSelection = (change) => {
    sampleRequest.current += 1
    setSample(null)
    setNotice('')
    setSelection(current => ({ ...current, ...change }))
  }
  const loadAreas = useCallback(async (parentCode) => {
    const { response, error } = await dispatch(fetchLibraryAreas(parentCode, itemId))
    if (error) {
      throw new Error(error)
    }
    return response.areasList.map(({ code, name, rowCount }) => ({ code, name, rowCount }))
  }, [dispatch, itemId])

  if (!item) {
    return (
      <div className={styles.libraryPane}>
        {notice
          ? <LibraryItemMessage message={notice} onBack={onBack} />
          : <div className={styles.libraryLoading}>Loading…</div>}
      </div>
    )
  }

  const { taxonomy } = detail
  const pickedIds = selection.brandIds.length > 0 ? selection.brandIds : selection.categoryIds
  const titles = new Map([...taxonomy.categories, ...taxonomy.brands].map(({ id, title }) => [id, title]))
  const name = tabName(item.title, pickedIds.map(id => titles.get(id) ?? id), areaName)
  // Nationwide counts only describe an unrestricted area; once an estimate exists, a category it omits has no rows there.
  const categoryCounts = estimate
    ? { ...Object.fromEntries(taxonomy.categories.map(({ id }) => [id, 0])), ...counts(estimate.byCategoryList) }
    : (selection.areaCode === '' ? detail.nationwide : {})
  const ready = issues.length === 0 && estimate !== null && !busy

  const submit = async () => {
    setBusy(true)
    setNotice('')
    track(mode === 'add' ? 'AddLibraryDataset' : 'UpdateLibraryDatasetSelection', { datasetId, itemId })
    const { error, code } = mode === 'add'
      ? await dispatch(addLibraryDataset(datasetId, itemId, selection))
      : await dispatch(updateLibrarySelection(datasetId, binding.version, selection))
    if (error) {
      setBusy(false)
      setNotice(code === CODE_CANCELED ? 'This dataset was changed by someone else. Check the selection and try again.' : error)
    } else if (mode !== 'add') {
      onDone()
    }
    // After a successful add the report stream replaces this pane with the dataset's status.
  }
  const previewSample = async () => {
    const request = ++sampleRequest.current
    const { response, error } = await dispatch(fetchLibrarySample(itemId, selection))
    if (request !== sampleRequest.current) {
      return
    }
    if (error) {
      setNotice(error)
    } else {
      setSample({ columns: response.columnsList, rows: response.rowsList.map(row => row.valuesList) })
    }
  }

  return (
    <div className={styles.libraryPane}>
      <LibraryItemDetail
        item={item}
        issues={issues}
        notice={notice}
        onBack={onBack}
        primary={mode === 'view'
          ? undefined
          : {
              label: mode === 'add'
                ? addButtonLabel(item.shape.kind, estimate?.rowCount ?? null)
                : `Update to ${estimate ? formatEstimate(item.shape.kind, estimate.rowCount) : 'new selection'}`,
              disabled: !ready,
              onClick: submit
            }}
        onPreviewSample={mode === 'add' ? previewSample : undefined}
        previewDisabled={issues.some(issue => issue.code !== 'empty' && issue.code !== 'too-many')}
        sample={sample}
      >
        {showArea && (
          <AreaPicker
            levels={item.areaLevels}
            value={selection.areaCode}
            onChange={areaCode => changeSelection({ areaCode })}
            loadAreas={loadAreas}
            onNameChange={setAreaName}
          />
        )}
        {showCategory && item.hasTaxonomy && (
          <CategoryPicker
            taxonomy={taxonomy}
            categoryIds={selection.categoryIds}
            brandIds={selection.brandIds}
            categoryCounts={categoryCounts}
            brandCounts={counts(estimate?.byBrandList)}
            onChange={next => changeSelection(next)}
          />
        )}
        {showArea && <div className={styles.libraryHint}>Tab name: {name}</div>}
      </LibraryItemDetail>
    </div>
  )
}

function LibraryItemMessage ({ message, onBack }) {
  return (
    <div className={styles.libraryLoading}>
      <div role='alert'>{message}</div>
      <button type='button' onClick={onBack}>Back</button>
    </div>
  )
}
