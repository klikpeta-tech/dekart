import { useEffect, useState } from 'react'
import Button from 'antd/es/button'
import Result from 'antd/es/result'
import { useDispatch, useSelector } from 'react-redux'
import { ApiTwoTone, ArrowLeftOutlined } from '@ant-design/icons'
import { useHistory } from 'react-router-dom/cjs/react-router-dom'
import { ConnectionType } from 'dekart-proto/dekart_pb'
import { DatasourceIcon } from './Datasource'
import { track } from './lib/tracking'
import { newConnection, newConnectionScreen } from './actions/connection'
import BigQueryConnectionTypeSelectorModal from './BigQueryConnectionTypeSelectorModal'
import styles from './HomePage.module.css'

function ConnectionTypeSelectorBottom () {
  const dispatch = useDispatch()
  const showCancel = useSelector(state => state.connection.list).length > 0
  const newScreen = useSelector(state => state.connection.screen)
  const isCloud = useSelector(state => state.env.isCloud)
  const history = useHistory()
  const showBack = showCancel || newScreen
  // Cloud keeps the back navigation in the selector footer.
  if (isCloud) {
    return (
      <div className={styles.connectionSelectorFooter}>
        {showBack
          ? (
            <Button
              type='text'
              icon={<ArrowLeftOutlined />}
              className={styles.connectionSelectorBack}
              onClick={() => {
                track('ReturnFromConnectionSelector')
                if (newScreen) {
                  dispatch(newConnectionScreen(false))
                } else {
                  history.push('/')
                }
              }}
            >Back
            </Button>
            )
          : null}
      </div>
    )
  }
  if (showCancel) {
    return (
      <div>
        <Button
          onClick={() => {
            track('ReturnFromConnectionSelector')
            if (newScreen) {
              dispatch(newConnectionScreen(false))
            } else {
              history.push('/')
            }
          }}
          icon={<ArrowLeftOutlined />}
        >Back
        </Button>
      </div>
    )
  }
  return null
}

// Render the connection options and route users to the selected connector setup.
function ConnectionTypeSelector () {
  const dispatch = useDispatch()
  const [bigqueryModalOpen, setBigqueryModalOpen] = useState(false)
  const connectionCards = [
    {
      key: 'bigquery',
      title: 'BigQuery',
      icon: <DatasourceIcon type={ConnectionType.CONNECTION_TYPE_BIGQUERY} />,
      handleClick: () => {
        track('ConnectionTypeSelectorBigQuery')
        setBigqueryModalOpen(true)
      }
    },
    {
      key: 'snowflake',
      title: 'Snowflake',
      icon: <DatasourceIcon type={ConnectionType.CONNECTION_TYPE_SNOWFLAKE} />,
      handleClick: () => {
        track('ConnectionTypeSelectorSnowflake')
        dispatch(newConnection(ConnectionType.CONNECTION_TYPE_SNOWFLAKE))
      }
    },
    {
      key: 'wherobots',
      title: 'Wherobots',
      icon: <DatasourceIcon type={ConnectionType.CONNECTION_TYPE_WHEROBOTS} />,
      handleClick: () => {
        track('ConnectionTypeSelectorWherobots')
        dispatch(newConnection(ConnectionType.CONNECTION_TYPE_WHEROBOTS))
      }
    }
  ]
  connectionCards.push({
    key: 'postgres',
    title: 'Postgres',
    icon: <DatasourceIcon type={ConnectionType.CONNECTION_TYPE_POSTGRES} />,
    handleClick: () => {
      track('ConnectionTypeSelectorPostgres')
      dispatch(newConnection(ConnectionType.CONNECTION_TYPE_POSTGRES))
    }
  })
  useEffect(() => {
    track('ConnectionTypeSelector')
  }, [])
  return (
    <>
      <div className={styles.connectionTypeSelector}>
        <BigQueryConnectionTypeSelectorModal open={bigqueryModalOpen} onClose={() => setBigqueryModalOpen(false)} />
        {connectionCards.map(card => (
          <button
            id={`dekart-connection-type-card-${card.key}`}
            key={card.key}
            type='button'
            className={styles.connectionTypeCard}
            onClick={card.handleClick}
            disabled={Boolean(card.disabled)}
            title={card.disabled ? card.disabledTitle : ''}
          >
            <div className={styles.connectionTypeCardIcon}>{card.icon}</div>
            <div className={styles.connectionTypeCardTitle}>{card.title}</div>
            {card.subtitle ? <div className={styles.connectionTypeCardSubtitle}>{card.subtitle}</div> : null}
            {card.hideConnectCta
              ? null
              : (
                <div className={styles.connectionTypeCardCta}>
                  <span className={styles.connectionTypeCardCtaLabel}>Connect</span>
                </div>
                )}
          </button>
        ))}
      </div>
      <ConnectionTypeSelectorBottom />
    </>
  )
}

export default function CreateConnection () {
  return (
    <>
      <Result
        status='success'
        icon={<ApiTwoTone />}
        title='Connect your warehouse.'
        subTitle={<>We run queries there; nothing is copied to Klikpeta.</>}
      />
      <ConnectionTypeSelector />
    </>
  )
}
