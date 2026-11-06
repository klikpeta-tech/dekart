import { useMemo } from 'react'
import { useSelector, useStore } from 'react-redux'
import { AiAssistantPanel } from '@openassistant/kepler-assistant'
import styles from './AiAssistantDock.module.css'

// AiAssistantDock shows the Kepler AI assistant beside the map while its map control is active.
export default function AiAssistantDock () {
  const store = useStore()
  const open = useSelector(state => Boolean(state.keplerGl.kepler?.uiState?.mapControls?.aiAssistant?.active))
  // The assistant reads the map through accessors so it never assumes Dekart's state shape.
  const stateAccessors = useMemo(() => ({
    getVisState: () => store.getState().keplerGl.kepler?.visState,
    getMapBoundary: () => store.getState().aiAssistant?.keplerGl?.mapBoundary
  }), [store])
  if (!open) {
    return null
  }
  return (
    <div className={styles.dock} data-testid='ai-assistant-panel' data-map-edit-panel>
      <AiAssistantPanel reduxStore={store} stateAccessors={stateAccessors} />
    </div>
  )
}
