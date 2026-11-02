/* eslint-disable no-undef */
import { LibraryItem, LibraryShape } from 'dekart-proto/dekart_pb'

export const workspaceId = '00000000-0000-0000-0000-000000000000'
export const ROLE_EDITOR = 2
export const ROLE_VIEWER = 3

// setRole replaces the user's default-workspace role, as the MCP role spec does.
export function setRole (email, role) {
  return cy.exec(`sqlite3 data/dekart.db "DELETE FROM workspace_log WHERE email='${email}'; INSERT INTO workspace_log (workspace_id, email, status, authored_by, id, role) VALUES ('${workspaceId}', '${email}', 1, '${email}', lower(hex(randomblob(16))), ${role})"`)
}

// libraryItem builds a granted catalogue entry.
export function libraryItem ({ id, title, category, kind, adminLevel = '', h3Resolution = 0, areaLevels, rowCount, hasTaxonomy = false }) {
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

// grpcWebReply frames a response message the way the Dekart server does, so the client decodes it unchanged.
export function grpcWebReply (message) {
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

// grpcWebError answers a command with a gRPC status, the way the Dekart server refuses one.
export function grpcWebError (code, message) {
  return {
    statusCode: 200,
    headers: {
      'content-type': 'application/grpc-web+proto',
      'grpc-status': String(code),
      'grpc-message': encodeURIComponent(message),
      'access-control-allow-origin': '*',
      'access-control-expose-headers': '*'
    },
    body: ''
  }
}

// grpcWebRequest decodes the message a request carries, so a stub can answer for what the browser asked.
export function grpcWebRequest (req, MessageClass) {
  const body = typeof req.body === 'string' ? Cypress.Buffer.from(req.body, 'binary') : Cypress.Buffer.from(req.body)
  return MessageClass.deserializeBinary(new Uint8Array(body.subarray(5, 5 + body.readUInt32BE(1)))).toObject()
}
