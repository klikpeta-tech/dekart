import { initApplicationConfig } from '@kepler.gl/utils'

let duckDBAdapter = null

// configureDuckDB installs the shared browser database before report data starts loading.
export async function configureDuckDB () {
  const [{ createSharedDuckDBAdapter }, { DekartDuckDBTable }] = await Promise.all([
    import('./database'),
    import('./table')
  ])
  duckDBAdapter ||= createSharedDuckDBAdapter()
  initApplicationConfig({
    table: DekartDuckDBTable,
    database: duckDBAdapter,
    // Progressive batches map to separate Deck.gl layers and degrade large-map FPS.
    useArrowProgressiveLoading: false,
    // Kepler 3.3 features that Dekart does not offer or persist; the map UI stays as before the upgrade.
    enableAnnotations: false,
    enableDrawOnMapSketches: false,
    enableMapNavigationControl: false,
    enableMapScale: false,
    enableSwipeMode: false,
    enableGlobeView: false,
    enableLayerGroups: false,
    enableCameraBookmarks: false,
    enableCameraTour: false,
    enableMapInfo: false,
    enableChartsPanel: false,
    enableFlowLayer: false,
    enableBitmapLayer: false,
    enableGeohashLayer: false,
    enableFlowFieldLayer: false,
    enableZarrLayer: false,
    enableJsonEditors: false,
    enableLayerJsonEditor: false,
    enableFilterJsonEditor: false,
    enableEffectJsonEditor: false,
    enableAnimationJsonEditor: false,
    enableChartJsonEditor: false
  })
}
