import { defineConfig } from 'vitest/config'

export default defineConfig({
  // Vitest does not load the React plugin, so JSX components need the automatic runtime.
  esbuild: { jsx: 'automatic' },
  test: {
    globals: true,
    environment: 'jsdom',
    deps: {
      // Optimized CommonJS entries already provide Vite's default-export interop.
      interopDefault: false,
      optimizer: {
        web: {
          enabled: true,
          // Bundle Kepler's CommonJS/ESM graph while preserving Node-only loaders.
          include: [
            '@kepler.gl/actions',
            '@kepler.gl/charts',
            '@kepler.gl/cloud-providers',
            '@kepler.gl/common-utils',
            '@kepler.gl/components',
            '@kepler.gl/constants',
            '@kepler.gl/deckgl-layers',
            '@kepler.gl/effects',
            '@kepler.gl/layers',
            '@kepler.gl/localization',
            '@kepler.gl/processors',
            '@kepler.gl/reducers',
            '@kepler.gl/schemas',
            '@kepler.gl/styles',
            '@kepler.gl/table',
            '@kepler.gl/tasks',
            '@kepler.gl/tasks-core',
            '@kepler.gl/utils',
            'antd/es/menu',
            'dekart-proto/dekart_pb', 'dekart-proto/dekart_pb_service'
          ],
          exclude: ['apache-arrow', 'thrift', 'parquet-wasm', 'h3-js']
        }
      }
    },
    setupFiles: './src/setupTests.js' // Optional: Use your existing setupTests.js
  }
})
