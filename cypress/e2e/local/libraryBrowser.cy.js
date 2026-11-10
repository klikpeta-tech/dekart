/* eslint-disable no-undef */
import {
  GetLibraryItemResponse,
  LibraryArea,
  LibraryCategory,
  LibraryShape,
  LibraryTaxonomy,
  ListLibraryAreasResponse,
  ListLibraryItemsResponse
} from 'dekart-proto/dekart_pb'
import { createReport } from './duckdbHelpers'
import { ROLE_ADMIN, ROLE_EDITOR, ROLE_VIEWER, grpcWebReply, libraryConnectionId, libraryItem, setRole, workspaceId } from './libraryStubs'

// libraryRows selects the dataset rows of the browser, each marked by its access badge.
function libraryRows () {
  return cy.get('li').filter(':contains("Available")')
}

const grantedItems = [
  libraryItem({ id: 'places', title: 'Titik Fasilitas Kesehatan 2025', category: 'Places', kind: LibraryShape.Kind.KIND_POINTS, areaLevels: ['provinsi', 'kabkota', 'kecamatan'], rowCount: 120000, hasTaxonomy: true }),
  libraryItem({ id: 'roads', title: 'Jaringan Jalan 2024', category: 'Infrastructure', kind: LibraryShape.Kind.KIND_LINES, areaLevels: ['kabkota'], rowCount: 480000 }),
  libraryItem({ id: 'boundaries', title: 'Batas Kabupaten Kota 2024', category: 'Boundaries', kind: LibraryShape.Kind.KIND_AREAS, adminLevel: 'kabkota', areaLevels: ['provinsi', 'kabkota'], rowCount: 514 }),
  libraryItem({ id: 'density', title: 'Grid Kepadatan H3 2024', category: 'Demographics', kind: LibraryShape.Kind.KIND_H3, h3Resolution: 8, areaLevels: ['provinsi'], rowCount: 90000 })
]

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

  it('keeps the library connection out of the SQL sources and never marks it as the default', () => {
    const email = `library-connection-${runId}@example.com`
    cy.setDevClaimsEmail(email)
    cy.visit('/')
    setRole(email, ROLE_ADMIN)
    stubLibrary(grantedItems, 12)
    // The staff-owned library connection is the only connection of the workspace.
    cy.exec(`sqlite3 data/dekart.db "INSERT OR REPLACE INTO connections (id, connection_name, workspace_id, connection_type, postgres_host, postgres_username, postgres_database, postgres_port) VALUES ('${libraryConnectionId}', 'Klikpeta Data Library', '${workspaceId}', 5, 'library.example.internal', 'reader', 'library', 5432)"`)
    createReport()

    cy.get('#dekart-library', { timeout: 30000 }).should('be.visible').and('not.be.disabled')
    cy.contains('Upload File').should('be.visible')
    cy.contains('Run SQL directly').should('not.exist')
    cy.contains('Klikpeta Data Library').should('not.exist')

    cy.visit('/connections')
    cy.contains('Klikpeta Data Library', { timeout: 30000 }).should('be.visible')
    cy.contains('Reserved for the Data Library').should('be.visible')
    cy.contains('Used by default').should('not.exist')
    cy.contains('button', 'Set default').should('not.exist')
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

    // A viewer has no Editing/Viewing switch and only gets the read-only Copy Link share button.
    cy.contains('button', 'Copy Link', { timeout: 30000 }).should('be.visible')
    cy.contains('.ant-select', /Viewing|Editing/).should('not.exist')
    // The dataset selector has rendered, so the missing library row is not a loading artifact.
    cy.contains('Upload File').should('be.visible')
    cy.get('#dekart-library').should('not.exist')
    // The library is never requested for a viewer.
    cy.get('@listLibraryItems.all').should('have.length', 1)
  })

  after(() => {
    // The connection row would change what the dataset selector offers in later specs of the lane.
    cy.exec(`sqlite3 data/dekart.db "DELETE FROM connections WHERE id='${libraryConnectionId}'"`)
  })
})
