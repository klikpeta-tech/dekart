import React from 'react'
import styles from './WorkspaceReadOnlyBanner.module.css'
import { useSelector } from 'react-redux'
import { useLocation } from 'react-router-dom'
import { GetWorkspaceResponse } from 'dekart-proto/dekart_pb'

export default function WorkspaceReadOnlyBanner () {
  const readOnly = useSelector(state => state.workspace.readOnly)
  const readOnlyReason = useSelector(state => state.workspace.readOnlyReason)
  const location = useLocation()

  // Device authorization is a standalone flow, keep it free of workspace banners.
  if (!readOnly || location.pathname === '/device/authorize') {
    return null
  }
  const licenseExpired = readOnlyReason === GetWorkspaceResponse.ReadOnlyReason.READ_ONLY_REASON_LICENSE_KEY_EXPIRED
  const headline = licenseExpired ? 'License key expired' : 'Workspace is read-only.'
  const detail = licenseExpired ? '' : 'Your maps stay viewable.'

  return (
    <div className={styles.banner} role='status'>
      <div className={styles.message}>
        <div className={styles.headline}>{headline}</div>
        {detail ? <div className={styles.detail}>{detail}</div> : null}
      </div>
    </div>
  )
}
