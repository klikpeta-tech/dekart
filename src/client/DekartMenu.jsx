import Menu from 'antd/es/menu'
import styles from './DekartMenu.module.css'
import { useDispatch, useSelector } from 'react-redux'
import { MenuOutlined } from '@ant-design/icons'
import { Link } from 'react-router-dom/cjs/react-router-dom'
import { createReport } from './actions/report'
import { track } from './lib/tracking'
import { canViewConnections } from './lib/connectionsAccess'
import { canManageLibraryAccess } from './lib/libraryAccess'

const popupOffset = [-10, 0]

export default function DekartMenu () {
  const env = useSelector(state => state.env)
  const userDefinedConnection = useSelector(state => state.connection.userDefined)
  const dispatch = useDispatch()
  const { authEnabled } = env
  const isPlayground = useSelector(state => state.user.isPlayground)
  const user = useSelector(state => state.user)
  const { isViewer } = user
  const readOnly = useSelector(state => state.workspace.readOnly)
  // main menu entries, each follows the visibility rules of the page it opens
  const children = [
    ...(authEnabled
      ? [
          { key: 'my', label: <Link to='/'>My Maps</Link>, onClick: () => track('NavigateToMyMaps') },
          { key: 'shared', label: <Link to='/shared'>Shared Maps</Link>, disabled: isPlayground, onClick: () => track('NavigateToSharedMaps') }
        ]
      : [
          { key: 'reports', label: <Link to='/'>Maps</Link>, onClick: () => track('NavigateToMaps') },
          { key: 'shared-disabled', label: <span>Shared Maps</span>, disabled: true }
        ]),
    ...(userDefinedConnection && canViewConnections(user)
      ? [{ key: 'connections', label: <Link to='/connections'>Connections</Link>, onClick: () => track('NavigateToConnections') }]
      : []),
    ...(canManageLibraryAccess(user)
      ? [{ key: 'library', label: <Link to='/library'>Data Library</Link>, onClick: () => track('NavigateToDataLibrary') }]
      : []),
    {
      key: 'create',
      label: 'New Map',
      disabled: isViewer || readOnly,
      onClick: () => {
        track('CreateNewMap')
        dispatch(createReport())
      }
    },
    { type: 'divider' },
    {
      key: 'source',
      label: <a target='_blank' rel='noopener noreferrer' href='https://github.com/klikpeta-tech/dekart'>Source - Based on Dekart (AGPL-3.0)</a>,
      onClick: () => track('ClickedSource')
    }
  ]
  const items = [{
    key: 'home',
    label: <MenuOutlined />,
    popupOffset,
    popupClassName: styles.subMenu,
    children
  }]
  return (
    <div className={styles.dekartMenu}>
      <Menu mode='horizontal' theme='dark' items={items} />
    </div>
  )
}
