import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Provider } from 'react-redux'
import { MemoryRouter } from 'react-router-dom/cjs/react-router-dom'
import { createStore } from 'redux'
import DekartMenu from './DekartMenu'

function renderMenu (user) {
  const state = {
    env: { authEnabled: true },
    connection: { userDefined: true },
    user,
    workspace: { readOnly: false }
  }
  render(
    <Provider store={createStore(() => state)}>
      <MemoryRouter>
        <DekartMenu />
      </MemoryRouter>
    </Provider>
  )
  fireEvent.mouseEnter(document.querySelector('.ant-menu-submenu-title'))
}

describe('DekartMenu Connections item', () => {
  it('is hidden from a Viewer', async () => {
    renderMenu({ stream: {}, isViewer: true, isPlayground: false })
    await screen.findByText('My Maps')
    expect(screen.queryByText('Connections')).toBeNull()
  })

  it('is hidden from an Editor', async () => {
    renderMenu({ stream: {}, isViewer: false, isAdmin: false, isPlayground: false })
    await screen.findByText('My Maps')
    expect(screen.queryByText('Connections')).toBeNull()
  })

  it('is shown to an Admin', async () => {
    renderMenu({ stream: {}, isViewer: false, isAdmin: true, isPlayground: false })
    expect(await screen.findByText('Connections')).toBeTruthy()
  })
})
