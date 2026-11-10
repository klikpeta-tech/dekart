/* eslint-disable no-undef */
import copy from '../../fixtures/copy.json'

// A map copied across workspaces has its dataset connections blanked on purpose (the copy is a dataset row with a
// NULL connection_id), so the SQL tab has to ask for a connection instead of showing the generic failure banner.
describe('copied map without a dataset connection', () => {
  it('asks the user to choose a connection', () => {
    cy.resetCloudTestDatabase()
    cy.setDevClaimsEmail('copied-map@example.com')
    cy.visit('/')
    cy.ensureTestWorkspace()
    cy.get('button#dekart-create-report').click()
    cy.location('pathname', { timeout: 30000 }).should('match', /\/reports\/[0-9a-f-]{36}\/source$/)
    cy.location('pathname').then((pathname) => {
      const reportId = pathname.split('/')[2]
      cy.psql(`
        INSERT INTO queries (report_id, id, query_text)
        VALUES ('${reportId}', '00000000-0000-0000-0000-0000000000a1', 'select 0 as lat, 0 as lon');
        INSERT INTO datasets (report_id, id, query_id, connection_id)
        VALUES ('${reportId}', '00000000-0000-0000-0000-0000000000b1', '00000000-0000-0000-0000-0000000000a1', NULL);
      `)
    })
    cy.reload()

    cy.get('#dekart-query-status-message', { timeout: 30000 }).should('contain', 'Choose a connection')
    cy.get('#dekart-query-status-hint').should('contain', 'copied from another workspace')
    cy.get(`button:contains("${copy.execute}")`).should('be.disabled')
  })
})
