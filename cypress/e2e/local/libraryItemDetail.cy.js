/* eslint-disable no-undef */
import {
  AddLibraryDatasetRequest,
  AddLibraryDatasetResponse,
  CountLibrarySelectionRequest,
  CountLibrarySelectionResponse,
  GetLibraryItemResponse,
  GetLibrarySampleResponse,
  LibraryArea,
  LibraryBrand,
  LibraryCategory,
  LibraryCategoryGroup,
  LibraryCount,
  LibraryItem,
  LibrarySampleRow,
  LibraryShape,
  LibraryTaxonomy,
  ListLibraryAreasRequest,
  ListLibraryAreasResponse,
  ListLibraryItemsResponse,
  RunQueryResponse,
  UpdateLibraryDatasetSelectionRequest,
  UpdateLibraryDatasetSelectionResponse
} from 'dekart-proto/dekart_pb'
import { createReport } from './duckdbHelpers'
import { ROLE_EDITOR, grpcWebError, grpcWebReply, grpcWebRequest, libraryConnectionId, libraryItem, setRole, workspaceId } from './libraryStubs'

// Hash of a map without parameters: the job a library dataset shows is the one with this hash.
const noParamsHash = 'd41d8cd98f00b204e9800998ecf8427e'
const itemId = 'places'

const place = libraryItem({ id: itemId, title: 'Tempat Usaha 2025', category: 'Places', kind: LibraryShape.Kind.KIND_POINTS, areaLevels: ['provinsi', 'kabkota', 'kecamatan'], rowCount: 0, hasTaxonomy: true })
place.setDescriptionMd('**Tempat usaha** di seluruh Indonesia.\n\nDiperbarui setiap kuartal.')
place.setAttribution('Sumber: OpenStreetMap')
place.setCadence('Kuartalan')
place.setSourceUpdatedAt(Date.UTC(2026, 9, 1) / 1000)
place.setColumnsList(['nama', 'alamat', 'lat', 'lon'])
place.setMaxRows(1000000)

const areas = {
  '': [['31', 'DKI Jakarta', 9000], ['32', 'Jawa Barat', 21000]],
  32: [['3273', 'Kota Bandung', 2100]],
  3273: [['3273010', 'Sukasari', 120]]
}

function taxonomy (rowCounts = [1200, 900]) {
  const group = new LibraryCategoryGroup()
  group.setId('makanan')
  group.setTitle('Makanan')
  const categories = [['kafe', 'Kafe'], ['kopi', 'Kedai kopi']].map(([id, title], index) => {
    const category = new LibraryCategory()
    category.setId(id)
    category.setGroupId('makanan')
    category.setTitle(title)
    category.setRowCount(rowCounts[index])
    return category
  })
  const brand = new LibraryBrand()
  brand.setId('starbucks')
  brand.setCategoryId('kopi')
  brand.setTitle('Starbucks')
  const result = new LibraryTaxonomy()
  result.setGroupsList([group])
  result.setCategoriesList(categories)
  result.setBrandsList([brand])
  return result
}

// stubLibrary answers every library read; count answers with the rows the test sets in `state.rows`.
// Like the server, it counts every granted category of the area, leaving out those in `state.omit` that have no rows there.
// A sample answers after `state.sampleDelay` milliseconds.
function stubLibrary (state) {
  const item = LibraryItem.deserializeBinary(place.serializeBinary())
  item.setRowCount(state.itemRows ?? 2100)
  // Count workflows do not need a remote basemap or Mapbox credentials.
  cy.intercept('GET', 'https://api.mapbox.com/styles/v1/**', {
    version: 8,
    sources: {},
    layers: []
  })
  cy.intercept('POST', '**/Dekart/ListLibraryItems', (req) => {
    const response = new ListLibraryItemsResponse()
    response.setItemsList([item])
    response.setCatalogueSize(1)
    req.reply(grpcWebReply(response))
  })
  cy.intercept('POST', '**/Dekart/GetLibraryItem', (req) => {
    const response = new GetLibraryItemResponse()
    response.setItem(item)
    response.setTaxonomy(taxonomy(state.categoryRows))
    req.reply(grpcWebReply(response))
  })
  cy.intercept('POST', '**/Dekart/ListLibraryAreas', (req) => {
    const { parentCode } = grpcWebRequest(req, ListLibraryAreasRequest)
    const response = new ListLibraryAreasResponse()
    response.setAreasList((areas[parentCode] || []).map(([code, name, rowCount]) => {
      const area = new LibraryArea()
      area.setCode(code)
      area.setName(name)
      area.setRowCount(state.itemRows === 0 ? 0 : rowCount)
      return area
    }))
    req.reply(grpcWebReply(response))
  })
  cy.intercept('POST', '**/Dekart/CountLibrarySelection', (req) => {
    const { selection } = grpcWebRequest(req, CountLibrarySelectionRequest)
    // Like the real count RPC, the fixture refuses an empty category selection.
    if (selection.categoryIdsList.length === 0) {
      req.reply(grpcWebError(3, 'pick at least one category'))
      return
    }
    const response = new CountLibrarySelectionResponse()
    response.setRowCount(state.rows)
    response.setByCategoryList(['kafe', 'kopi'].filter(id => !(state.omit || []).includes(id)).map(id => {
      const count = new LibraryCount()
      count.setId(id)
      count.setRowCount(state.rows)
      return count
    }))
    req.reply(grpcWebReply(response))
  }).as('count')
  cy.intercept('POST', '**/Dekart/GetLibrarySample', (req) => {
    const response = new GetLibrarySampleResponse()
    response.setColumnsList(['nama', 'alamat'])
    response.setRowsList((state.sampleRows ?? [['Kopi <b>Kenangan</b>', 'Jl. Braga 1']]).map(values => {
      const row = new LibrarySampleRow()
      row.setValuesList(values)
      return row
    }))
    req.reply({ ...grpcWebReply(response), delay: state.sampleDelay || 0 })
  }).as('sample')
}

// currentReportId reads the id of the open map from the address.
function currentReportId () {
  return cy.location('pathname').then(path => path.match(/\/reports\/([0-9a-f-]+)/)[1])
}

// seedLibraryDataset does what AddLibraryDataset does on the server: it binds the map's empty dataset to the item.
// The stubs never reach the library database, so the test writes the rows the report stream reads.
function seedLibraryDataset (email, { version = 1, areaCode, name, grantEndsOn = '2099-12-31', jobError = '' }) {
  return currentReportId().then(reportId => {
    const sql = [
      `INSERT OR IGNORE INTO queries (report_id, id, query_text) VALUES ('${reportId}', '${reportId}', '')`,
      `UPDATE datasets SET query_id='${reportId}', connection_id='${libraryConnectionId}', name='${name}' WHERE report_id='${reportId}'`,
      `DELETE FROM library_dataset_bindings WHERE report_id='${reportId}'`,
      `INSERT INTO library_dataset_bindings (dataset_id, report_id, item_id, area_code, category_ids, brand_ids, version, created_by) SELECT id, report_id, '${itemId}', '${areaCode}', json_array('kafe'), json_array(), ${version}, '${email}' FROM datasets WHERE report_id='${reportId}'`,
      `INSERT INTO library_grant_log (workspace_id, item_id, status, ends_on, category_ids, authored_by) VALUES ('${workspaceId}', '${itemId}', 1, '${grantEndsOn}', json_array('kafe'), '${email}')`,
      `DELETE FROM query_jobs WHERE query_id='${reportId}'`
    ]
    if (jobError) {
      sql.push(`INSERT INTO query_jobs (id, query_id, job_status, job_error, query_params_hash) VALUES (lower(hex(randomblob(4))) || '-0000-4000-8000-' || lower(hex(randomblob(6))), '${reportId}', 0, '${jobError}', '${noParamsHash}')`)
    }
    return cy.exec(`sqlite3 data/dekart.db "${sql.join('; ')}"`)
  })
}

// openDetail opens the library from the empty dataset tab and then the item.
function openDetail (rowCount = '2.100') {
  cy.get('#dekart-library', { timeout: 30000 }).should('not.be.disabled').click()
  cy.contains('li', 'Tempat Usaha 2025').should(rowCount === '' ? 'not.contain' : 'contain', 'rows').click()
  cy.contains('h2', 'Tempat Usaha 2025').scrollIntoView().should('be.visible')
  if (rowCount === '') {
    // An unknown count (0) is left out instead of shown as a zero.
    cy.contains('dt', 'Provider').should('be.visible')
    cy.contains('dt', 'Rows').should('not.exist')
  } else {
    cy.contains('dt', 'Rows').parent().find('dd').should('have.text', rowCount)
  }
}

describe('data library item detail', () => {
  const runId = Date.now()

  function openMapAsEditor (name, state) {
    const email = `library-${name}-${runId}@example.com`
    cy.setDevClaimsEmail(email)
    cy.visit('/')
    setRole(email, ROLE_EDITOR)
    stubLibrary(state)
    createReport()
    return email
  }

  it('lets an editor describe, narrow and add an item, and refuses what the server refuses', () => {
    const state = { rows: 2100 }
    openMapAsEditor('add', state)
    cy.intercept('POST', '**/Dekart/AddLibraryDataset', (req) => {
      req.reply(grpcWebError(7, 'category is not granted to this workspace'))
    }).as('addRefused')
    openDetail()

    // The detail describes the item without any SQL or connection.
    cy.contains('Tempat usaha di seluruh Indonesia.').scrollIntoView().should('be.visible')
    // The description is Markdown: emphasis is rendered, no literal asterisks are left.
    cy.contains('strong', 'Tempat usaha').should('be.visible')
    cy.contains('**').should('not.exist')
    cy.contains('dt', 'Rows').scrollIntoView().should('be.visible')
    cy.contains('dd', 'Kuartalan').scrollIntoView().should('be.visible')
    cy.contains('1 Oktober 2026').scrollIntoView().should('be.visible')
    cy.contains('li', 'alamat').scrollIntoView().should('be.visible')
    cy.contains('Sumber: OpenStreetMap').scrollIntoView().should('be.visible')
    cy.contains('Tab name: Tempat Usaha 2025 · Indonesia').scrollIntoView().should('be.visible')

    // A category is required before the item can be added.
    cy.contains('Pick at least one category.').scrollIntoView().should('be.visible')
    cy.contains('button', 'Add to map').should('be.disabled')
    cy.contains('button', 'Preview sample').should('be.disabled')

    cy.get('select[aria-label="Provinsi"]').should('contain', 'Jawa Barat (21.000)').select('Jawa Barat (21.000)')
    cy.get('select[aria-label="Kabupaten/Kota"]').should('contain', 'Kota Bandung (2.100)').select('Kota Bandung (2.100)')
    cy.contains('label', 'Kafe').find('input').check()
    cy.contains('label', 'Kedai kopi').find('input').check()
    // Brands sit under their category, so equal brand titles of several categories stay apart.
    cy.get('[role="group"][aria-label="Kedai kopi brands"]').should('contain', 'Kedai kopi').contains('button', 'Starbucks').click()
    cy.contains('Tab name: Tempat Usaha 2025 · Starbucks · Kota Bandung').scrollIntoView().should('be.visible')
    cy.contains('button', 'Add 2.100 places to map').should('not.be.disabled')

    // The preview shows rows as text.
    cy.contains('button', 'Preview sample').click()
    cy.get('table[aria-label="Sample rows"]').should('contain', 'Kopi <b>Kenangan</b>')

    // Over the item's row limit, nothing can be added.
    cy.then(() => { state.rows = 1200000 })
    cy.contains('label', 'Kafe').find('input').uncheck()
    cy.contains('1.200.000 rows is over the limit of 1.000.000.', { timeout: 15000 }).scrollIntoView().should('be.visible')
    cy.contains('button', 'Add 1.200.000 places to map').should('be.disabled')
    cy.then(() => { state.rows = 2100 })
    cy.contains('label', 'Kafe').find('input').check()
    cy.contains('button', 'Add 2.100 places to map').click()

    // The server refuses an ungranted category; the reason is shown and the user can try again.
    cy.wait('@addRefused').then(({ request }) => {
      expect(grpcWebRequest(request, AddLibraryDatasetRequest)).to.deep.include({ itemId })
      expect(grpcWebRequest(request, AddLibraryDatasetRequest).selection).to.deep.equal({
        areaCode: '3273',
        categoryIdsList: ['kopi', 'kafe'],
        brandIdsList: ['starbucks']
      })
    })
    cy.contains('category is not granted to this workspace').scrollIntoView().should('be.visible')
    cy.contains('button', 'Add 2.100 places to map').should('not.be.disabled')
  })

  it('counts categories for the narrowed area and drops a preview of an older selection', () => {
    const state = { rows: 120, omit: ['kopi'], sampleDelay: 1500 }
    openMapAsEditor('picker', state)
    openDetail()

    // Nationwide counts show only while no area is picked; a category without rows in the area shows 0.
    cy.contains('label', 'Kafe (1.200)').scrollIntoView().should('be.visible')
    cy.get('select[aria-label="Provinsi"]').should('contain', 'Jawa Barat').select('Jawa Barat (21.000)')
    cy.contains('label', 'Kafe (120)').should('exist')
    cy.contains('label', 'Kedai kopi (0)').should('exist')
    cy.contains('label', 'Kafe').find('input').should('not.be.checked')
    cy.contains('button', 'Add to map').should('be.disabled')
    cy.contains('label', 'Kafe').find('input').check()

    // A preview that answers after the selection changed is not shown for the new selection.
    cy.contains('button', 'Preview sample').click()
    cy.contains('label', 'Kedai kopi').find('input').check()
    cy.wait('@sample')
    // Waits out the time the browser needs to handle the answer; the table must still be absent.
    cy.wait(500)
    cy.get('table[aria-label="Sample rows"]').should('not.exist')

    // Clearing every category keeps the area's counts without enabling Add.
    cy.contains('label', 'Kafe').find('input').uncheck()
    cy.contains('label', 'Kedai kopi').find('input').uncheck()
    cy.contains('label', 'Kafe (120)').scrollIntoView().should('be.visible')
    cy.contains('label', 'Kedai kopi (0)').should('be.visible')
    cy.contains('button', 'Add to map').should('be.disabled')
  })

  it('shows zero counts and no fabricated sample rows when Places is empty', () => {
    openMapAsEditor('empty', { rows: 0, itemRows: 0, categoryRows: [0, 0], omit: ['kafe', 'kopi'], sampleRows: [] })
    openDetail('')
    cy.contains('label', 'Kafe (0)').should('exist')
    cy.get('select[aria-label="Provinsi"]').should('contain', 'Jawa Barat (0)').select('32')
    cy.contains('label', 'Kafe (0)').should('exist')
    cy.contains('label', 'Kedai kopi (0)').should('exist')
    cy.contains('label', 'Kafe').find('input').check()
    cy.contains('No rows match this selection.').scrollIntoView().should('be.visible')
    cy.contains('button', 'Add 0 places to map').should('be.disabled')
    cy.contains('button', 'Preview sample').click()
    cy.get('table[aria-label="Sample rows"]').should('contain', 'nama').find('tbody tr').should('not.exist')
  })

  it('renders an empty source preview even when aggregate counts exist', () => {
    openMapAsEditor('empty-source', { rows: 120, sampleRows: [] })
    openDetail()
    cy.get('select[aria-label="Provinsi"]').should('contain', 'Jawa Barat').select('32')
    cy.contains('label', 'Kafe (120)').should('exist')
    cy.contains('label', 'Kafe').find('input').check()
    cy.contains('button', 'Add 120 places to map').should('not.be.disabled')
    cy.contains('button', 'Preview sample').click()
    cy.get('table[aria-label="Sample rows"]').should('contain', 'nama').find('tbody tr').should('not.exist')
  })

  it('shows the status of an added dataset and changes its area', () => {
    const email = openMapAsEditor('change', { rows: 120 })
    cy.intercept('POST', '**/Dekart/AddLibraryDataset', (req) => {
      req.reply(grpcWebReply(new AddLibraryDatasetResponse()))
    }).as('add')
    openDetail()
    cy.get('select[aria-label="Provinsi"]').should('contain', 'Jawa Barat').select('Jawa Barat (21.000)')
    cy.contains('label', 'Kafe').find('input').check()
    cy.contains('button', 'Add 120 places to map').click()
    cy.wait('@add')

    // The server would now hold the binding and send it through the report stream.
    seedLibraryDataset(email, { areaCode: '32', name: 'Tempat Usaha 2025 · Kafe · Jawa Barat' })
    cy.reload()
    cy.contains('[role="status"]', 'Loading', { timeout: 30000 }).should('exist')
    cy.contains('h2', 'Tempat Usaha 2025 · Kafe · Jawa Barat').scrollIntoView().should('be.visible')
    cy.contains('button', 'Change category').scrollIntoView().should('be.visible')
    cy.contains('button', 'Open in Library').scrollIntoView().should('be.visible')
    cy.contains('SELECT').should('not.exist')

    // Change area asks for the new selection only, and sends the binding version it saw.
    cy.contains('button', 'Change area').click()
    cy.get('select[aria-label="Provinsi"]').should('have.value', '32')
    cy.get('select[aria-label="Kabupaten/Kota"]').should('contain', 'Kota Bandung').select('Kota Bandung (2.100)')
    cy.contains('label', 'Kafe').should('not.exist')
    cy.intercept('POST', '**/Dekart/UpdateLibraryDatasetSelection', (req) => {
      req.reply(grpcWebError(1, 'binding changed since it was read'))
    }).as('updateStale')
    cy.contains('button', 'Update to 120 places').click()
    cy.wait('@updateStale').then(({ request }) => {
      const body = grpcWebRequest(request, UpdateLibraryDatasetSelectionRequest)
      expect(body.expectedVersion).to.equal(1)
      expect(body.selection).to.deep.equal({ areaCode: '3273', categoryIdsList: ['kafe'], brandIdsList: [] })
    })
    cy.contains('This dataset was changed by someone else.').scrollIntoView().should('be.visible')

    // The retry is accepted; the stream then brings the new binding and name.
    cy.intercept('POST', '**/Dekart/UpdateLibraryDatasetSelection', (req) => {
      const response = new UpdateLibraryDatasetSelectionResponse()
      response.setVersion(2)
      req.reply(grpcWebReply(response))
    }).as('update')
    cy.contains('button', 'Update to 120 places').click()
    cy.wait('@update')
    cy.contains('button', 'Change area').scrollIntoView().should('be.visible')
    seedLibraryDataset(email, { version: 2, areaCode: '3273', name: 'Tempat Usaha 2025 · Kafe · Kota Bandung' })
    cy.reload()
    cy.contains('h2', 'Tempat Usaha 2025 · Kafe · Kota Bandung', { timeout: 30000 }).scrollIntoView().should('be.visible')
  })

  it('changes the category of a bound dataset without naming an area', () => {
    const email = openMapAsEditor('category', { rows: 120 })
    cy.get('#dekart-library', { timeout: 30000 }).should('be.visible')
    seedLibraryDataset(email, { areaCode: '3273', name: 'Tempat Usaha 2025 · Kafe · Kota Bandung' })
    cy.reload()
    cy.contains('button', 'Change category', { timeout: 30000 }).click()
    cy.contains('label', 'Kafe').should('exist')
    cy.contains('Tab name:').should('not.exist')
  })

  it('retries a failed dataset, and shows access ended with refresh refused', () => {
    const email = openMapAsEditor('status', { rows: 120 })
    cy.intercept('POST', '**/Dekart/RunQuery', (req) => {
      req.reply(grpcWebReply(new RunQueryResponse()))
    }).as('run')
    // Reaching the empty tab first makes the report exist; the dataset is then bound the way the server would.
    cy.get('#dekart-library', { timeout: 30000 }).should('be.visible')
    seedLibraryDataset(email, { areaCode: '3273', name: 'Tempat Usaha 2025 · Kafe · Kota Bandung', jobError: 'connection refused' })
    cy.reload()

    cy.contains('[role="status"]', 'Failed: connection refused', { timeout: 30000 }).should('exist')
    cy.contains('button', 'Retry').click()
    cy.wait('@run').its('request.body').should('exist')

    seedLibraryDataset(email, { areaCode: '3273', name: 'Tempat Usaha 2025 · Kafe · Kota Bandung', grantEndsOn: '2020-01-31' })
    cy.reload()
    cy.contains('[role="status"]', 'Access ended 31 Januari 2020. Your Klikpeta contact can extend it.', { timeout: 30000 }).should('exist')
    cy.contains('button', 'Refresh').should('be.disabled')
    cy.contains('button', 'Change area').should('be.disabled')
    cy.contains('button', 'Change category').should('be.disabled')
    cy.contains('button', 'Open in Library').should('not.exist')
    cy.screenshot('library-access-ended')
  })
})
