import '@ant-design/v5-patch-for-react-19'
import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './index.css'
import { createStore, applyMiddleware, compose } from 'redux'
import { Provider } from 'react-redux'
import ConfigProvider from 'antd/es/config-provider'
import { antdTheme } from './antdTheme'
import { thunk } from 'redux-thunk'
import reducer from './reducers/rootReducer'
import { enhanceReduxMiddleware } from '@kepler.gl/reducers'
import { setTrackingStore, setTrackEventToServer } from './lib/tracking'
import { trackEventToServer } from './actions/tracking'

const store = createStore(
  reducer,
  compose(
    applyMiddleware(...enhanceReduxMiddleware([thunk]))
  )
)

// Initialize tracking with Redux store for Dekart Cloud
setTrackingStore(store)
setTrackEventToServer(trackEventToServer)

createRoot(document.getElementById('root')).render(
  <Provider store={store}>
    <ConfigProvider theme={antdTheme}>
      <App />
    </ConfigProvider>
  </Provider>
)
