import { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { LibraryDatasetStatus } from './dark/data-library/index'
import { runWarehouseQuery } from './actions/query'
import LibraryDetailPane from './LibraryDetailPane'
import { libraryDatasetState } from './lib/libraryItems'
import styles from './Dataset.module.css'

// Replaces the query editor of a library dataset: a status bar with actions, and no SQL or connection.
export default function LibraryDatasetPane ({ dataset }) {
  const dispatch = useDispatch()
  // 'area' and 'category' change that part of the selection; 'view' describes the item; null shows the status.
  const [opened, setOpened] = useState(null)
  const binding = dataset.library
  const report = useSelector(state => state.report)
  const readOnly = useSelector(state => state.workspace.readOnly)
  const edit = useSelector(state => state.reportStatus.edit)
  const hash = useSelector(state => state.queryParams.hash)
  const job = useSelector(state => state.queryJobs.find(j => j.queryId === dataset.queryId && j.queryParamsHash === hash))
  const pending = useSelector(state => Boolean(state.queryExecutionsPending[dataset.queryId]))
  const canChange = report.canWrite && edit && !readOnly

  if (opened) {
    return (
      <LibraryDetailPane
        itemId={binding.itemId}
        datasetId={dataset.id}
        binding={binding}
        mode={opened}
        onBack={() => setOpened(null)}
        onDone={() => setOpened(null)}
      />
    )
  }

  // The server rebuilds the SQL from the binding, so the query text sent here is ignored.
  const refresh = () => dispatch(runWarehouseQuery(dataset.queryId, ''))
  return (
    <div className={styles.libraryPane}>
      <LibraryDatasetStatus
        title={dataset.name}
        state={libraryDatasetState(binding, job, pending)}
        error={job?.jobError}
        accessEndedAt={binding.accessEndsAt}
        canChange={canChange}
        canRefresh={report.canRefresh && !readOnly}
        hasCategories={binding.selection?.categoryIdsList?.length > 0}
        onChangeCategory={() => setOpened('category')}
        onChangeArea={() => setOpened('area')}
        onRefresh={refresh}
        onRetry={refresh}
        onOpenLibrary={canChange ? () => setOpened('view') : undefined}
      />
    </div>
  )
}
