import Button from 'antd/es/button'
import Modal from 'antd/es/modal/Modal'
import Tooltip from 'antd/es/tooltip'
import styles from './BigQueryConnectionTypeSelectorModal.module.css'
import { newConnection } from './actions/connection'
import { ConnectionType } from 'dekart-proto/dekart_pb'
import { useDispatch, useSelector } from 'react-redux'
import { useEffect } from 'react'
import { track } from './lib/tracking'

export default function BigQueryConnectionTypeSelectorModal ({ open, onClose }) {
  const dispatch = useDispatch()
  const googleOAuthEnabled = useSelector(state => state.env.googleOAuthEnabled)
  const googleOAuthDisabledTitle = googleOAuthEnabled ? '' : 'Google OAuth is not configured for this instance'
  useEffect(() => {
    if (open) {
      track('OpenBigQueryConnectionTypeSelectorModal')
    }
  }, [open])

  return (
    <Modal
      title='Choose BigQuery Connection Method'
      open={open}
      onCancel={onClose}
      footer={null}
      width={800}
    >
      <div className={styles.connectionOptions}>
        <div className={styles.connectionOptionsItem}>
          <h2>
            Use your Google Account
          </h2>
          <p>
            Recommended if your account already has BigQuery access.
          </p>
          <p>
            No credentials stored in Klikpeta backend.
          </p>

          <p>
            <Tooltip title={googleOAuthDisabledTitle}>
              <span>
                <Button
                  type='primary'
                  disabled={!googleOAuthEnabled}
                  onClick={() => {
                    onClose()
                    dispatch(newConnection(ConnectionType.CONNECTION_TYPE_BIGQUERY))
                    track('ConnectWithGoogle')
                  }}
                >Connect with Google
                </Button>
              </span>
            </Tooltip>
          </p>
        </div>
        <div className={styles.connectionOptionsItem}>
          <h2>
            Use Service Account
          </h2>
          <p>
            Connect to BigQuery with using Service Account Key in JSON format.
          </p>
          <p>Key will be encrypted and stored in Klikpeta backend.</p>

          <p>
            <Button
              type='primary'
              onClick={() => {
                onClose()
                dispatch(newConnection(ConnectionType.CONNECTION_TYPE_BIGQUERY, true))
                track('ConnectWithServiceAccount')
              }}
            >Configure Service Account
            </Button>
          </p>
        </div>
      </div>
    </Modal>
  )
}
