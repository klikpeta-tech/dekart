import { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import dayjs from 'dayjs'
import Button from 'antd/es/button'
import DatePicker from 'antd/es/date-picker'
import Switch from 'antd/es/switch'
import Tag from 'antd/es/tag'
import { calendarDay, grantRowState } from './dark/access-grants/index'
import LibraryCategoriesModal from './LibraryCategoriesModal'
import styles from './LibraryGrantRow.module.css'
import { revokeLibraryGrant, setLibraryGrant, setLibraryGrantCategories } from './actions/libraryGrants'
import { toAccessGrant } from './lib/libraryGrantInput'

const DAY_FORMAT = 'YYYY-MM-DD'
// access ends at the end of the day in Asia/Jakarta (UTC+7), so the earliest end day is today there
const JAKARTA_UTC_OFFSET_MINUTES = 420

const KIND_COLOR = {
  granted: 'green',
  'ending-soon': 'orange',
  ended: 'red',
  'not-granted': 'default'
}

// initialEndsOn keeps a past end date out of the picker, so a re-grant needs a new date.
function initialEndsOn (grant, today) {
  return grant.endsOn >= today ? grant.endsOn : ''
}

function isPickedDayAllowed (day, today) {
  return day.format(DAY_FORMAT) >= today
}

function onGrantSwitch (checked, { item, endsOn, dispatch, setModal }) {
  if (!checked) {
    dispatch(revokeLibraryGrant(item.id))
  } else if (item.hasTaxonomy) {
    // a taxonomy item needs its categories chosen before it can be granted
    setModal('grant')
  } else {
    dispatch(setLibraryGrant(item.id, endsOn, []))
  }
}

function onModalOk (categoryIds, { modal, item, endsOn, dispatch, setModal }) {
  setModal('')
  if (modal === 'grant') {
    dispatch(setLibraryGrant(item.id, endsOn, categoryIds))
  } else {
    dispatch(setLibraryGrantCategories(item.id, categoryIds))
  }
}

// LibraryGrantRow shows one catalogue item with its grant switch, end date and categories.
export default function LibraryGrantRow ({ row, nowSeconds, readOnly }) {
  const dispatch = useDispatch()
  const saving = useSelector(state => state.libraryGrants.savingItemId === row.item.id)
  const { item, grant } = row
  const today = calendarDay(nowSeconds, JAKARTA_UTC_OFFSET_MINUTES)
  const { kind, label } = grantRowState(toAccessGrant(grant), nowSeconds)
  const granted = kind === 'granted' || kind === 'ending-soon'
  const [draftEndsOn, setDraftEndsOn] = useState(() => initialEndsOn(grant, today))
  const [modal, setModal] = useState('')
  // a granted row always shows the server's date, so a refused change snaps back
  const endsOn = granted ? grant.endsOn : draftEndsOn
  const disabled = readOnly || saving

  return (
    <div className={styles.row} data-testid='library-grant-row'>
      <div className={styles.info}>
        <div className={styles.title}>{item.title}</div>
        <div className={styles.meta}>
          {[item.category, item.provider, item.coverageSummary].filter(Boolean).join(' · ')}
        </div>
      </div>
      <Tag color={KIND_COLOR[kind]} className={styles.state}>{label}</Tag>
      <DatePicker
        className={styles.date}
        placeholder='Access until'
        allowClear={!granted}
        disabled={disabled}
        value={endsOn ? dayjs(endsOn) : null}
        disabledDate={(day) => !isPickedDayAllowed(day, today)}
        onChange={(day) => {
          const picked = day ? day.format(DAY_FORMAT) : ''
          setDraftEndsOn(picked)
          if (granted) {
            dispatch(setLibraryGrant(item.id, picked, grant.categoryIdsList))
          }
        }}
      />
      <Switch
        checked={granted}
        disabled={disabled || (!granted && !endsOn)}
        title={!granted && !endsOn ? 'Pick an end date first' : undefined}
        onChange={(checked) => onGrantSwitch(checked, { item, endsOn, dispatch, setModal })}
      />
      {item.hasTaxonomy && granted
        ? (
          <Button size='small' disabled={disabled} onClick={() => setModal('manage')}>
            Manage categories ({grant.categoryIdsList.length})
          </Button>
          )
        : <span />}
      {modal
        ? (
          <LibraryCategoriesModal
            itemId={item.id}
            title={item.title}
            initial={grant.categoryIdsList}
            okText={modal === 'grant' ? 'Grant access' : 'Save'}
            onOk={(categoryIds) => onModalOk(categoryIds, { modal, item, endsOn, dispatch, setModal })}
            onCancel={() => setModal('')}
          />
          )
        : null}
    </div>
  )
}
