import ConfigProvider from 'antd/es/config-provider'
import { setTwoToneColor } from '@ant-design/icons'

// antdTheme keeps the antd 4 look on antd 5: same blue, 2px corners, outlined (wireframe) components and v4 text and border colors.
// Geometry that has no token is restored in antdCompat.css.
export const antdTheme = {
  token: {
    colorPrimary: '#1890ff',
    colorInfo: '#1890ff',
    colorLink: '#1890ff',
    colorText: 'rgba(0, 0, 0, 0.85)',
    colorTextHeading: 'rgba(0, 0, 0, 0.85)',
    colorBorderSecondary: '#f0f0f0',
    colorSplit: '#f0f0f0',
    borderRadius: 2,
    wireframe: true
  },
  components: {
    Button: {
      defaultShadow: '0 2px 0 rgba(0, 0, 0, 0.016)',
      primaryShadow: '0 2px 0 rgba(0, 0, 0, 0.045)',
      dangerShadow: '0 2px 0 rgba(0, 0, 0, 0.045)'
    },
    Modal: {
      titleLineHeight: 1.375
    },
    Tabs: {
      cardBg: '#fafafa'
    }
  }
}

// message and Modal.confirm are called outside React, so they only get the theme through the static config.
ConfigProvider.config({ theme: antdTheme })

// two-tone icons without an explicit color used the antd 4 blue
setTwoToneColor('#1890ff')
