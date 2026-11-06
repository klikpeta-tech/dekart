/* eslint-disable no-undef */

// uploadActiveDataset completes the visible local file-upload flow for the selected dataset.
function uploadActiveDataset (fixture) {
  cy.intercept('POST', '**/Dekart/CreateFile').as('createFile')
  cy.intercept('POST', '**/api/v1/file/*/upload-sessions').as('startUploadSession')
  cy.intercept('PUT', '**/api/v1/file/*/upload-sessions/*/parts/*').as('uploadPart')
  cy.intercept('POST', '**/api/v1/file/*/upload-sessions/*/complete').as('completeUploadSession')
  cy.contains('button', 'Upload File', { timeout: 20000 }).click({ force: true })
  cy.wait('@createFile', { timeout: 60000 })
  cy.get('input[type="file"]', { timeout: 20000 }).selectFile(fixture, { force: true })
  cy.contains('button', 'Upload').click()
  cy.wait('@startUploadSession', { timeout: 60000 })
  cy.wait('@uploadPart', { timeout: 60000 })
  cy.wait('@completeUploadSession', { timeout: 120000 })
  cy.contains('Ready', { timeout: 120000 }).should('be.visible')
}

describe('Kepler 3.3 map controls', () => {
  it('offers annotations, charts, effects and the AI assistant and keeps Filter out in sync with the chart widgets', () => {
    cy.visit('/')
    cy.get('body', { timeout: 20000 }).then(($body) => {
      // The local lane may start before or after its one-time file-upload setup.
      if ($body.text().includes('Ready to connect')) {
        cy.contains('button', 'Use file upload').click()
      } else {
        cy.get('button#dekart-create-report').click()
      }
    })
    uploadActiveDataset('cypress/fixtures/sample.csv')
    cy.get('[data-testid="widgets-tab"]', { timeout: 120000 }).then(button => {
      if (button.attr('aria-expanded') !== 'true') cy.wrap(button).click()
    })
    cy.get('[data-testid="number-value"]', { timeout: 120000 }).should('have.text', '8,276')

    // The map scale sits beside the open control pane and returns to the map edge when the pane collapses.
    cy.get('[aria-label="Map panels"]').then($pane => {
      const paneRight = $pane[0].getBoundingClientRect().right
      cy.get('.map-scale').should($scale => {
        expect($scale[0].getBoundingClientRect().left).to.be.at.least(paneRight)
      })
      cy.get('[aria-label="Collapse control pane"]').click()
      cy.get('.map-scale').should($scale => {
        expect($scale[0].getBoundingClientRect().left).to.be.lessThan(paneRight)
      })
      cy.get('[data-testid="widgets-tab"]').click()
    })

    // Each control of the Kepler demo map opens its own panel.
    cy.get('.map-control-button.toggle-annotation', { timeout: 60000 }).click()
    cy.contains('Annotations', { timeout: 20000 }).should('be.visible')
    cy.get('.map-control-button.toggle-annotation').click()
    cy.get('.map-control-button.toggle-chart-panel').click()
    cy.contains('Charts', { timeout: 20000 }).should('be.visible')
    cy.get('.map-control-button.toggle-chart-panel').click()
    cy.get('.map-control-button.toggle-effect').click()
    cy.contains('Effects', { timeout: 20000 }).should('be.visible')
    cy.get('.map-control-button.toggle-effect').click()
    cy.get('.map-control-button.toggle-ai-assistant').click()
    cy.get('[data-testid="ai-assistant-panel"]', { timeout: 60000 }).should('be.visible')
      .find('textarea, input').should('exist')
    cy.get('.map-control-button.toggle-ai-assistant').click()
    cy.get('[data-testid="ai-assistant-panel"]').should('not.exist')

    // Filter out keeps the rows a range filter rejects, on the map and in the chart widgets.
    cy.openLayerPanel()
    cy.get('[data-for="filter-nav"]').click()
    cy.contains('button', 'Add Filter').click()
    cy.contains('Select a field').click()
    cy.contains('.list__item, [class*="list-item"]', 'district').click()
    cy.get('.kg-range-slider__input').first().clear().type('10').blur()
    cy.get('[data-testid="widgets-tab"]').click()
    cy.get('[data-testid="number-value"]', { timeout: 60000 }).should('have.text', '4,383')
    cy.get('[data-testid="map-settings-tab"]').click()
    cy.get('.filter-panel__filter-out input').click({ force: true })
    cy.get('[data-testid="widgets-tab"]').click()
    cy.get('[data-testid="number-value"]', { timeout: 60000 }).should('have.text', '3,893')
  })
})
