import React, { useEffect, useRef } from 'react'
import Modal from 'antd/es/modal'
import Title from 'antd/es/typography/Title'
import Text from 'antd/es/typography/Text'
import { LockOutlined } from '@ant-design/icons'
import { track } from './lib/tracking'
import styles from './SSOLoginModal.module.css'

export default function SSOLoginModal ({
  visible,
  onClose
}) {
  const wasVisible = useRef(false)

  useEffect(() => {
    if (visible && !wasVisible.current) {
      track('SSOPopupOpened')
    }
    wasVisible.current = visible
  }, [visible])

  const closeModal = () => {
    track('SSOPopupClosed')
    onClose()
  }

  return (
    <Modal
      open={visible}
      title={null}
      footer={null}
      onCancel={closeModal}
      closable
      maskClosable
      width={480}
    >
      <div className={styles.modalContent}>
        <div className={styles.iconWrap}>
          <LockOutlined className={styles.icon} />
        </div>
        <Title level={4} className={styles.heading}>Login requires SSO</Title>
        <Text className={styles.body}>
          Klikpeta is running in anonymous mode. To enable login, teams, and shared maps, your instance admin needs to configure SSO.
        </Text>
      </div>
    </Modal>
  )
}
