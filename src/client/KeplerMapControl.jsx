import styled from 'styled-components'
import {
  MapControlFactory,
  EffectControlFactory,
  EffectManagerFactory,
  AnnotationManagerFactory,
  ChartControlFactory
} from '@kepler.gl/components'
import { ChartManagerFactory } from '@kepler.gl/components/charts'
import { getApplicationConfig } from '@kepler.gl/utils'
import { AiAssistantControlFactory } from '@openassistant/kepler-assistant'

const StyledMapControlPanel = styled.div`
  position: relative;
`

// Dekart's fullscreen and location buttons occupy the top 100px of the map corner, so panels start below them.
const StyledMapControlContextPanel = styled.div`
  margin-top: 100px;
  max-height: calc(100% - 100px);
  overflow: hidden;

  display: flex;
  flex-direction: column;
  justify-content: space-between;
  pointer-events: none !important; /* prevent padding from blocking input */
  & > * {
    /* all children should allow input */
    pointer-events: all;
  }
`

const StyledMapControlOverlay = styled.div`
  position: absolute;
  display: flex;
  top: ${props => props.$top}px;
  right: 0;
  z-index: 1;
  pointer-events: none !important; /* prevent padding from blocking input */
  & > * {
    /* all children should allow input */
    pointer-events: all;
  }

  margin-top: ${props => (props.$rightPanelVisible ? props.theme.rightPanelMarginTop : 0)}px;
  margin-right: ${props => (props.$rightPanelVisible ? props.theme.rightPanelMarginRight : 0)}px;
  max-height: calc(100% - ${props => props.theme.rightPanelMarginTop + props.theme.bottomWidgetPaddingBottom}px);

  .map-control {
    ${props => (props.$rightPanelVisible ? 'padding-top: 0px;' : '')}
  }
`

// Kepler's default map control omits charts, effects, annotations and the AI assistant; this mirrors
// the kepler.gl demo composition so every control of the 3.3 demo map is available in Dekart.
function DekartMapControlFactory (EffectControl, EffectManager, AnnotationManager, ChartManager, ChartControl, AiAssistantControl, ...deps) {
  const MapControl = MapControlFactory(...deps)

  return function DekartMapControl (props) {
    const isExport = Boolean(props.isExport)
    const chartsEnabled = Boolean(getApplicationConfig().enableChartsPanel)
    const actionComponents = [
      ...(MapControl.defaultActionComponents ?? []),
      ...(chartsEnabled ? [ChartControl] : []),
      EffectControl,
      AiAssistantControl
    ]
    const showEffects = Boolean(props.mapControls?.effect?.active)
    const showAnnotations = Boolean(props.mapControls?.annotation?.active)
    const showChartsPanel = chartsEnabled && Boolean(props.mapControls?.chart?.active)
    const hasPinnedCharts = !isExport && chartsEnabled && (props.charts || []).some(chart => chart.pinned !== false)
    const showCharts = showChartsPanel || hasPinnedCharts
    const rightPanelVisible = showEffects || showAnnotations || showCharts
    return (
      <StyledMapControlOverlay $top={props.top} $rightPanelVisible={rightPanelVisible}>
        <StyledMapControlPanel>
          <MapControl {...props} top={0} actionComponents={actionComponents} />
        </StyledMapControlPanel>
        <StyledMapControlContextPanel data-map-edit-panel>
          {showAnnotations ? <AnnotationManager /> : null}
          {showCharts ? <ChartManager panelActive={showChartsPanel} isExport={isExport} /> : null}
          {showEffects ? <EffectManager /> : null}
        </StyledMapControlContextPanel>
      </StyledMapControlOverlay>
    )
  }
}
DekartMapControlFactory.deps = [
  EffectControlFactory,
  EffectManagerFactory,
  AnnotationManagerFactory,
  ChartManagerFactory,
  ChartControlFactory,
  AiAssistantControlFactory,
  ...MapControlFactory.deps
]

export { MapControlFactory, DekartMapControlFactory }
