/* eslint-disable no-undef */
import {
  GetLibraryTaxonomyRequest,
  GetLibraryTaxonomyResponse,
  LibraryCategory,
  LibraryCategoryGroup,
  LibraryGrant,
  LibraryGrantRow,
  LibraryItem,
  LibraryTaxonomy,
  ListLibraryGrantsResponse,
  RevokeLibraryGrantRequest,
  RevokeLibraryGrantResponse,
  SetLibraryGrantCategoriesRequest,
  SetLibraryGrantCategoriesResponse,
  SetLibraryGrantRequest,
  SetLibraryGrantResponse
} from 'dekart-proto/dekart_pb'

// The Data Library admin RPCs (ListLibraryGrants, GetLibraryTaxonomy, SetLibraryGrant,
// RevokeLibraryGrant, SetLibraryGrantCategories) are STUBBED with cy.intercept, because the
// local lane has no library database and its caller is not Klikpeta staff. The stub keeps the
// grants in memory so the tab is driven through the real client and asserted through the DOM only.

const DATASET_ID = 'dataset-1'
const PLACES_ID = 'places'
const REFUSAL = 'only Klikpeta staff can manage Data Library access'
const GRPC_HEADERS = {
  'content-type': 'application/grpc-web+proto',
  'access-control-allow-origin': '*',
  'access-control-expose-headers': 'grpc-status,grpc-message'
}

let grants

function newItem (id, title, category, hasTaxonomy) {
  const item = new LibraryItem()
  item.setId(id)
  item.setTitle(title)
  item.setCategory(category)
  item.setProvider('Klikpeta')
  item.setHasTaxonomy(hasTaxonomy)
  return item
}

function newTaxonomy () {
  const taxonomy = new LibraryTaxonomy()
  const groups = [['group-1', 'Makanan'], ['group-2', 'Kesehatan']]
  const categories = [
    ['cat-1', 'group-1', 'Restoran'],
    ['cat-2', 'group-1', 'Kafe'],
    ['cat-3', 'group-2', 'Apotek']
  ]
  taxonomy.setGroupsList(groups.map(([id, title], index) => {
    const group = new LibraryCategoryGroup()
    group.setId(id)
    group.setTitle(title)
    group.setSortOrder(index)
    return group
  }))
  taxonomy.setCategoriesList(categories.map(([id, groupId, title], index) => {
    const category = new LibraryCategory()
    category.setId(id)
    category.setGroupId(groupId)
    category.setTitle(title)
    category.setSortOrder(index)
    category.setRowCount(1000 + index)
    return category
  }))
  return taxonomy
}

function newGrant (itemId, state, endsOn, categoryIds) {
  const grant = new LibraryGrant()
  grant.setItemId(itemId)
  grant.setState(state)
  grant.setEndsOn(endsOn)
  // access lasts until the end of the day in Asia/Jakarta
  grant.setAccessEndsAt(endsOn ? Date.parse(`${endsOn}T23:59:59+07:00`) / 1000 : 0)
  grant.setCategoryIdsList(categoryIds)
  return grant
}

function frame (flag, payload) {
  const bytes = new Uint8Array(payload.length + 5)
  bytes[0] = flag
  new DataView(bytes.buffer).setUint32(1, payload.length)
  bytes.set(payload, 5)
  return bytes
}

function replyOk (req, message) {
  const trailers = new TextEncoder().encode('grpc-status: 0\r\n')
  const data = frame(0, message.serializeBinary())
  const trailer = frame(0x80, trailers)
  const body = new Uint8Array(data.length + trailer.length)
  body.set(data)
  body.set(trailer, data.length)
  // the status travels in the trailer frame: a grpc-status header would make the client read a trailers-only reply without a message
  // an ArrayBuffer body reaches the client byte for byte; a Cypress.Buffer would be sent as JSON
  req.reply({ statusCode: 200, headers: GRPC_HEADERS, body: body.buffer })
}

// replyRefused answers with a trailers-only gRPC PERMISSION_DENIED.
function replyRefused (req) {
  req.reply({
    statusCode: 200,
    headers: { ...GRPC_HEADERS, 'grpc-status': '7', 'grpc-message': REFUSAL },
    body: ''
  })
}

function decodeRequest (req, RequestType) {
  const bytes = Cypress.Buffer.from(req.body)
  return RequestType.deserializeBinary(new Uint8Array(bytes.buffer, bytes.byteOffset + 5, bytes.length - 5))
}

function stubLibraryRpcs ({ refuseList = false } = {}) {
  grants = new Map([
    [DATASET_ID, newGrant(DATASET_ID, LibraryGrant.State.STATE_NOT_GRANTED, '', [])],
    [PLACES_ID, newGrant(PLACES_ID, LibraryGrant.State.STATE_NOT_GRANTED, '', [])]
  ])
  const items = [
    newItem(DATASET_ID, 'Batas Wilayah 2024', 'Administrasi', false),
    newItem(PLACES_ID, 'Places Indonesia', 'Places', true)
  ]
  cy.intercept('OPTIONS', '**/Dekart/*Library*', {
    statusCode: 204,
    headers: { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'POST' }
  })
  cy.intercept('POST', '**/Dekart/ListLibraryGrants', (req) => {
    if (refuseList) {
      replyRefused(req)
      return
    }
    const response = new ListLibraryGrantsResponse()
    response.setRowsList(items.map(item => {
      const row = new LibraryGrantRow()
      row.setItem(item)
      row.setGrant(grants.get(item.getId()))
      return row
    }))
    replyOk(req, response)
  })
  cy.intercept('POST', '**/Dekart/GetLibraryTaxonomy', (req) => {
    decodeRequest(req, GetLibraryTaxonomyRequest)
    const response = new GetLibraryTaxonomyResponse()
    response.setTaxonomy(newTaxonomy())
    replyOk(req, response)
  })
  cy.intercept('POST', '**/Dekart/SetLibraryGrant', (req) => {
    const request = decodeRequest(req, SetLibraryGrantRequest)
    const grant = newGrant(request.getItemId(), LibraryGrant.State.STATE_ACTIVE, request.getEndsOn(), request.getCategoryIdsList())
    grants.set(request.getItemId(), grant)
    const response = new SetLibraryGrantResponse()
    response.setGrant(grant)
    replyOk(req, response)
  })
  cy.intercept('POST', '**/Dekart/RevokeLibraryGrant', (req) => {
    const request = decodeRequest(req, RevokeLibraryGrantRequest)
    const previous = grants.get(request.getItemId())
    const grant = newGrant(request.getItemId(), LibraryGrant.State.STATE_REVOKED, previous.getEndsOn(), previous.getCategoryIdsList())
    grants.set(request.getItemId(), grant)
    const response = new RevokeLibraryGrantResponse()
    response.setGrant(grant)
    replyOk(req, response)
  })
  cy.intercept('POST', '**/Dekart/SetLibraryGrantCategories', (req) => {
    const request = decodeRequest(req, SetLibraryGrantCategoriesRequest)
    const previous = grants.get(request.getItemId())
    const grant = newGrant(request.getItemId(), LibraryGrant.State.STATE_ACTIVE, previous.getEndsOn(), request.getCategoryIdsList())
    grants.set(request.getItemId(), grant)
    const response = new SetLibraryGrantCategoriesResponse()
    response.setGrant(grant)
    replyOk(req, response)
  })
}

// futureDay is a day well after today in every timezone, as the picker only allows days from today on.
function futureDay () {
  return new Date(Date.now() + 400 * 24 * 3600 * 1000).toISOString().slice(0, 10)
}

function row (title) {
  return cy.contains('[data-testid="library-grant-row"]', title, { timeout: 30000 })
}

function pickEndDay (title, day) {
  row(title).find('.ant-picker input').type(`${day}{enter}`, { force: true })
  row(title).find('.ant-picker input').should('have.value', day)
}

function openGroup (name) {
  cy.contains('.ant-collapse-header', name).click()
}

function tickCategory (name) {
  cy.contains('.ant-modal label.ant-checkbox-wrapper', name).click()
}

describe('Data Library access tab', () => {
  it('grants and revokes a dataset with an end date', () => {
    stubLibraryRpcs()
    cy.visit('/library')

    row('Batas Wilayah 2024').contains('Not granted').should('be.visible')
    row('Batas Wilayah 2024').find('.ant-switch').should('be.disabled')

    const day = futureDay()
    pickEndDay('Batas Wilayah 2024', day)
    row('Batas Wilayah 2024').find('.ant-switch').click()
    row('Batas Wilayah 2024').contains(/Ends in \d+ days/).should('be.visible')
    row('Batas Wilayah 2024').find('.ant-picker input').should('have.value', day)
    row('Batas Wilayah 2024').find('.ant-switch').should('have.attr', 'aria-checked', 'true')

    cy.reload()
    row('Batas Wilayah 2024').contains(/Ends in \d+ days/).should('be.visible')

    row('Batas Wilayah 2024').find('.ant-switch').click()
    row('Batas Wilayah 2024').contains('Not granted').should('be.visible')
    row('Batas Wilayah 2024').find('.ant-switch').should('have.attr', 'aria-checked', 'false')
  })

  it('grants Places with categories and keeps categories of closed groups when editing', () => {
    stubLibraryRpcs()
    cy.visit('/library')

    pickEndDay('Places Indonesia', futureDay())
    row('Places Indonesia').find('.ant-switch').click()

    cy.contains('.ant-modal-title', 'Categories · Places Indonesia').should('be.visible')
    cy.get('#dekart-library-categories-ok').should('be.disabled')
    cy.contains('Pick at least one category.').should('be.visible')
    openGroup('Makanan')
    openGroup('Kesehatan')
    tickCategory('Restoran')
    tickCategory('Apotek')
    cy.contains('2 selected').should('be.visible')
    cy.get('#dekart-library-categories-ok').click()
    row('Places Indonesia').contains('button', 'Manage categories (2)').should('be.visible')

    // only the first group is opened, so the category granted in the other one must survive the edit
    row('Places Indonesia').contains('button', 'Manage categories (2)').click()
    cy.contains('2 selected').should('be.visible')
    openGroup('Makanan')
    tickCategory('Kafe')
    cy.contains('3 selected').should('be.visible')
    cy.get('#dekart-library-categories-ok').click()
    row('Places Indonesia').contains('button', 'Manage categories (3)').should('be.visible')

    cy.reload()
    row('Places Indonesia').contains('button', 'Manage categories (3)').should('be.visible')
  })

  it('shows the server refusal in the tab only', () => {
    stubLibraryRpcs({ refuseList: true })
    cy.visit('/library')

    cy.get('.ant-alert-error', { timeout: 30000 }).contains(REFUSAL).should('be.visible')
    cy.get('.ant-message-notice').should('not.exist')
    cy.get('.ant-alert-error').find('.ant-alert-close-icon').click()
    cy.get('.ant-alert-error').should('not.exist')
  })
})
