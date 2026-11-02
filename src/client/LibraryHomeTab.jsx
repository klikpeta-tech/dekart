import { useEffect, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import Alert from 'antd/es/alert'
import Input from 'antd/es/input'
import Radio from 'antd/es/radio'
import Spin from 'antd/es/spin'
import { SearchOutlined } from '@ant-design/icons'
import { filterRows } from './dark/access-grants/index'
import LibraryGrantRow from './LibraryGrantRow'
import styles from './LibraryHomeTab.module.css'
import { dismissLibraryGrantsError, listLibraryGrants } from './actions/libraryGrants'
import { toAccessGrant } from './lib/libraryGrantInput'

// LibraryHomeTab lets a workspace Admin grant catalogue datasets to the current workspace.
// The server also requires the caller to be Klikpeta staff and refuses otherwise.
export default function LibraryHomeTab () {
  const dispatch = useDispatch()
  const { rows, loaded, error } = useSelector(state => state.libraryGrants)
  const readOnly = useSelector(state => state.workspace.readOnly)
  const [filter, setFilter] = useState('all')
  const [search, setSearch] = useState('')
  useEffect(() => {
    dispatch(listLibraryGrants())
  }, [dispatch])

  const nowSeconds = Math.floor(Date.now() / 1000)
  const visibleRows = filterRows(
    rows.map(row => ({ title: row.item.title, category: row.item.category, grant: toAccessGrant(row.grant), row })),
    { filter, search },
    nowSeconds
  )

  return (
    <div className={styles.tab}>
      {error
        ? (
          <Alert
            type='error'
            showIcon
            closable
            message={error}
            onClose={() => dispatch(dismissLibraryGrantsError())}
          />
          )
        : null}
      <div className={styles.toolbar}>
        <Input
          placeholder='Search datasets...'
          prefix={<SearchOutlined />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          allowClear
          className={styles.search}
        />
        <Radio.Group value={filter} onChange={(e) => setFilter(e.target.value)}>
          <Radio.Button value='all'>All</Radio.Button>
          <Radio.Button value='granted'>Granted</Radio.Button>
          <Radio.Button value='ending-soon'>Ending soon</Radio.Button>
          <Radio.Button value='not-granted'>Not granted</Radio.Button>
        </Radio.Group>
      </div>
      {loaded
        ? (
          <div className={styles.list}>
            {visibleRows.map(({ row }) => (
              <LibraryGrantRow key={row.item.id} row={row} nowSeconds={nowSeconds} readOnly={readOnly} />
            ))}
            {visibleRows.length === 0 && !error ? <div className={styles.empty}>No datasets found</div> : null}
          </div>
          )
        : <Spin />}
    </div>
  )
}
