/* eslint-disable no-undef */
import {
  GetLibraryItemResponse,
  LibraryArea,
  LibraryCategory,
  LibraryItem,
  LibraryShape,
  LibraryTaxonomy,
  ListLibraryAreasResponse,
  ListLibraryItemsResponse
} from 'dekart-proto/dekart_pb'
import { createReport } from './duckdbHelpers'

const workspaceId = '00000000-0000-0000-0000-000000000000'
const ROLE_EDITOR = 2
const ROLE_VIEWER = 3

// setRole replaces the user's default-workspace role, as the MCP role spec does.
function setRole (email, role) {
  return cy.exec(`sqlite3 data/dekart.db "DELETE FROM workspace_log WHERE email='${email}'; INSERT INTO workspace_log (workspace_id, email, status, authored_by, id, role) VALUES ('${workspaceId}', '${email}', 1, '${email}', lower(hex(randomblob(16))), ${role})"`)
}

// libraryRows selects the dataset rows of the browser, each marked by its access badge.
function libraryRows () {
  return cy.get('li').filter(':contains("Available")')
}

// libraryItem builds a granted catalogue entry.
function libraryItem ({ id, title, category, kind, adminLevel = '', h3Resolution = 0, areaLevels, rowCount, hasTaxonomy = false }) {
  const shape = new LibraryShape()
  shape.setKind(kind)
  shape.setAdminLevel(adminLevel)
  shape.setH3Resolution(h3Resolution)
  const item = new LibraryItem()
  item.setId(id)
  item.setTitle(title)
  item.setCategory(category)
  item.setProvider('Klikpeta')
  item.setCoverageSummary('Seluruh Indonesia')
  item.setShape(shape)
  item.setRowCount(rowCount)
  item.setAreaLevelsList(areaLevels)
  item.setHasTaxonomy(hasTaxonomy)
  return item
}

const grantedItems = [
  libraryItem({ id: 'places', title: 'Titik Fasilitas Kesehatan 2025', category: 'Places', kind: LibraryShape.Kind.KIND_POINTS, areaLevels: ['provinsi', 'kabkota', 'kecamatan'], rowCount: 120000, hasTaxonomy: true }),
  libraryItem({ id: 'roads', title: 'Jaringan Jalan 2024', category: 'Infrastructure', kind: LibraryShape.Kind.KIND_LINES, areaLevels: ['kabkota'], rowCount: 480000 }),
  libraryItem({ id: 'boundaries', title: 'Batas Kabupaten Kota 2024', category: 'Boundaries', kind: LibraryShape.Kind.KIND_AREAS, adminLevel: 'kabkota', areaLevels: ['provinsi', 'kabkota'], rowCount: 514 }),
  libraryItem({ id: 'density', title: 'Grid Kepadatan H3 2024', category: 'Demographics', kind: LibraryShape.Kind.KIND_H3, h3Resolution: 8, areaLevels: ['provinsi'], rowCount: 90000 })
]

// grpcWebReply frames a response message the way the Dekart server does, so the client decodes it unchanged.
function grpcWebReply (message) {
  const payload = Cypress.Buffer.from(message.serializeBinary())
  const trailer = Cypress.Buffer.from('grpc-status:0\r\n')
  const body = Cypress.Buffer.alloc(5 + payload.length + 5 + trailer.length)
  body.writeUInt32BE(payload.length, 1)
  payload.copy(body, 5)
  body[5 + payload.length] = 0x80
  body.writeUInt32BE(trailer.length, 5 + payload.length + 1)
  trailer.copy(body, 10 + payload.length)
  return {
    statusCode: 200,
    headers: {
      'content-type': 'application/grpc-web+proto',
      'access-control-allow-origin': '*',
      'access-control-expose-headers': '*'
    },
    body: body.buffer.slice(body.byteOffset, body.byteOffset + body.length)
  }
}

// stubLibrary answers the library RPCs, since no library database is available to the local lane.
function stubLibrary (items, catalogueSize) {
  cy.intercept('POST', '**/Dekart/ListLibraryItems', (req) => {
    const response = new ListLibraryItemsResponse()
    response.setItemsList(items)
    response.setCatalogueSize(catalogueSize)
    req.reply(grpcWebReply(response))
  }).as('listLibraryItems')
  cy.intercept('POST', '**/Dekart/GetLibraryItem', (req) => {
    const category = new LibraryCategory()
    category.setTitle('Apotek')
    const taxonomy = new LibraryTaxonomy()
    taxonomy.setCategoriesList([category])
    const response = new GetLibraryItemResponse()
    response.setItem(items[0])
    response.setTaxonomy(taxonomy)
    req.reply(grpcWebReply(response))
  }).as('getLibraryItem')
  cy.intercept('POST', '**/Dekart/ListLibraryAreas', (req) => {
    const areas = [['31', 'DKI Jakarta'], ['32', 'Jawa Barat']].map(([code, name]) => {
      const area = new LibraryArea()
      area.setCode(code)
      area.setLevel('provinsi')
      area.setName(name)
      return area
    })
    const response = new ListLibraryAreasResponse()
    response.setAreasList(areas)
    req.reply(grpcWebReply(response))
  }).as('listLibraryAreas')
}

describe('data library browser', () => {
  const runId = Date.now()

  it('lets an editor browse granted datasets and filter them by search, area and shape', () => {
    const email = `library-editor-${runId}@example.com`
    cy.setDevClaimsEmail(email)
    cy.visit('/')
    setRole(email, ROLE_EDITOR)
    stubLibrary(grantedItems, 12)
    createReport()

    cy.get('#dekart-library', { timeout: 30000 }).should('be.visible').and('not.be.disabled')
      .and('contain', 'Browse Data Library')
      .and('contain', '12 datasets')
      .and('contain', '4 available to this workspace')
    cy.get('#dekart-library').click()

    cy.contains('Data Library').should('be.visible')
    libraryRows().should('have.length', 4)
    cy.contains('li', 'Titik Fasilitas Kesehatan 2025').should('contain', 'Places').and('contain', 'Points')
    cy.contains('li', 'Batas Kabupaten Kota 2024').should('contain', 'Areas · Kabupaten/Kota')
    cy.contains('li', 'Grid Kepadatan H3 2024').should('contain', 'H3 hexagons · res 8')

    // A search term matches through the granted taxonomy categories of the Places item.
    cy.get('input[aria-label="Search datasets"]').type('apotek')
    libraryRows().should('have.length', 1).and('contain', 'Titik Fasilitas Kesehatan 2025')
    cy.get('input[aria-label="Search datasets"]').clear()
    libraryRows().should('have.length', 4)

    // A province keeps only datasets that can be narrowed to provinces.
    cy.get('select[aria-label="Area"]').select('DKI Jakarta')
    libraryRows().should('have.length', 3)
    cy.contains('li', 'Jaringan Jalan 2024').should('not.exist')

    cy.get('select[aria-label="Shape"]').select('Areas')
    libraryRows().should('have.length', 1).and('contain', 'Batas Kabupaten Kota 2024')
    cy.get('select[aria-label="Shape"]').select('Lines')
    cy.contains('No datasets match your filters.').should('be.visible')
    cy.contains('button', 'Clear filters').click()
    libraryRows().should('have.length', 4)

    cy.get('button[aria-label="Back"]').click()
    cy.get('#dekart-library').should('be.visible')
    cy.screenshot('library-browser')
  })

  it('disables the library row when nothing is granted to the workspace', () => {
    const email = `library-empty-${runId}@example.com`
    cy.setDevClaimsEmail(email)
    cy.visit('/')
    setRole(email, ROLE_EDITOR)
    stubLibrary([], 12)
    createReport()

    cy.get('#dekart-library', { timeout: 30000 }).should('be.visible').and('be.disabled')
      .and('contain', 'Klikpeta is preparing your library')
  })

  it('does not show the library to a viewer', () => {
    const email = `library-viewer-${runId}@example.com`
    cy.setDevClaimsEmail(email)
    cy.visit('/')
    setRole(email, ROLE_EDITOR)
    stubLibrary(grantedItems, 12)
    createReport()
    cy.get('#dekart-library', { timeout: 30000 }).should('be.visible')
    cy.location('pathname').then((path) => {
      setRole(email, ROLE_VIEWER)
      cy.visit(path)
    })

    cy.contains('.ant-select', 'Viewing', { timeout: 30000 }).should('be.visible')
    cy.get('#dekart-library').should('not.exist')
    // The library is never requested for a viewer.
    cy.get('@listLibraryItems.all').should('have.length', 1)
  })
})
