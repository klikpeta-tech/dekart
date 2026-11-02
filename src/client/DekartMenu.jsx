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
  return (
    <div className={styles.dekartMenu}>
      <Menu mode='horizontal' theme='dark'>
        <Menu.SubMenu
          id='dekart-main-menu'
          popupOffset={popupOffset}
          popupClassName={styles.subMenu} title={<MenuOutlined />} key='home' active='yes'
        >
          {authEnabled
            ? (
              <>
                <Menu.Item key='my' onClick={() => track('NavigateToMyMaps')}>
                  <Link to='/'>My Maps</Link>
                </Menu.Item>
                <Menu.Item key='shared' disabled={isPlayground} onClick={() => track('NavigateToSharedMaps')}>
                  <Link to='/shared'>Shared Maps</Link>
                </Menu.Item>
              </>
              )
            : (
              <>
                <Menu.Item key='reports' onClick={() => track('NavigateToMaps')}>
                  <Link to='/'>Maps</Link>
                </Menu.Item>
                <Menu.Item key='shared-disabled' disabled>
                  <span>Shared Maps</span>
                </Menu.Item>
              </>
              )}
          {userDefinedConnection && canViewConnections(user)
            ? (
              <Menu.Item key='connections' onClick={() => track('NavigateToConnections')}>
                <Link to='/connections'>Connections</Link>
              </Menu.Item>
              )
            : null}
          {canManageLibraryAccess(user)
            ? (
              <Menu.Item key='library' onClick={() => track('NavigateToDataLibrary')}>
                <Link to='/library'>Data Library</Link>
              </Menu.Item>
              )
            : null}
          <Menu.Item
            key='create'
            disabled={isViewer || readOnly}
            onClick={() => {
              track('CreateNewMap')
              dispatch(createReport())
            }}
          >
            New Map
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item key='source' onClick={() => track('ClickedSource')}>
            <a target='_blank' rel='noopener noreferrer' href='https://github.com/klikpeta-tech/dekart'>Source - Based on Dekart (AGPL-3.0)</a>
          </Menu.Item>
        </Menu.SubMenu>
      </Menu>
    </div>
  )
}
