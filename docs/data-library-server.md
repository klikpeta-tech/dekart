# Data Library server foundation

Server side of the Klikpeta Data Library (ENG-2164). The design and decisions are in ENG-2098; this page records the contract the client lanes build against.

## Pieces

| Piece | Where |
| --- | --- |
| Proto messages and RPCs | `proto/dekart.proto`, "Data Library" blocks |
| Grant log and dataset bindings | `sqlite/klikpeta-migrations/000002_data_library.up.sql` (fork track, `klikpeta_schema_migrations`) |
| Grant store, binding store, expiry, catalogue interface, catalogue reader (`PGCatalogue`) | `src/server/library` |
| Pure SQL builder (dark library) | `src/server/dark/libraryquery` (generated `README.md`) |
| gRPC handlers | `src/server/dekart/library.go` (client), `library_admin.go` (admin) |
| Run, stream and fork integration | `src/server/dekart/library_run.go`; hooks in `query.go`, `stream.go`, `mcp.go`, `report.go` |
| Opening the library database | `src/server/dekart/library_catalogue.go`, called from `src/server/main.go` |

## Rules the handlers enforce

- Grants are per workspace and item, with a required end date. Access lasts through 23:59:59 Asia/Jakarta of that day. The latest log row (highest `seq`) wins.
- The grant checked for a map is the grant of the map's own workspace, not the caller's.
- Reads (`ListLibraryItems`, `GetLibraryItem`, `ListLibraryAreas`, `CountLibrarySelection`, `GetLibrarySample`) hide items without an active grant: an unknown item, an ungranted one and an ended one all answer `NotFound`.
- `AddLibraryDataset` and `UpdateLibraryDatasetSelection` need write access to the map; an ended grant answers `FailedPrecondition`, an ungranted category `PermissionDenied`, a selection over the item's `max_rows` `FailedPrecondition`, a stale `expected_version` `Canceled`.
- Admin commands (`ListLibraryGrants`, `GetLibraryTaxonomy`, `SetLibraryGrant`, `RevokeLibraryGrant`, `SetLibraryGrantCategories`) need the workspace Admin role and the caller's email to equal `DEKART_DEFAULT_WORKSPACE_ADMIN` (Klikpeta staff). An unset setting refuses everyone.
- Count metadata of ungranted categories is protected: for taxonomy items `ListLibraryAreas` counts only granted categories and `LibraryItem.row_count` is left unset.
- Items with a taxonomy (Places) need at least one granted category; other items take none.
- `DEKART_LIBRARY_CONNECTION_ID` names the staff-owned library connection. `CreateQuery`, `CreateFile` and `UpdateQuery` refuse it, so user SQL never runs on it.
- Every library command answers `Unavailable` until a catalogue reader is set. `main.go` sets one when `DEKART_LIBRARY_CONNECTION_ID` is configured.

## Catalogue reader

`library.PGCatalogue` reads `lib_catalog` and runs samples on `lib` through the library connection: the connection row's Postgres host, user and server-encrypted password. The database is opened on first use with at most two sessions. Items, taxonomy and areas are cached for 60 seconds; counts, estimates and samples always hit the database. `lib_catalog.item.columns` must be a JSON array of column names. Area, category and brand filters use the view columns `kode_provinsi`, `kode_kabkota`, `kode_kecamatan` (by the item's `area_levels`) and, for taxonomy items, `category_id` and `brand_id`. Estimates come from `item_area_count` and `poi_count`, never from the data views.

## Running a library dataset

`RunQuery` (including the MCP run path), `RunAllQueries` and the first run after `AddLibraryDataset` or `UpdateLibraryDatasetSelection` all end in `runQuery`. For a query on the library connection, the SQL comes only from the dataset's binding: the grant of the map's workspace is checked, then `library.BuildSQL` builds the statement. Stored and supplied text are ignored.

- A missing binding, an ended or revoked grant, or a selection with no granted category left answers `FailedPrecondition` ("library access ended <date>" for an ended grant). The last stored result stays.
- `RunAllQueries` skips such a dataset instead of failing the whole map.
- A failed first run after add or change is logged; the binding is saved and a refresh runs it again.

## Stream and copies

- The report stream (and the MCP report properties tool) sends each library dataset with `Dataset.library` set (item, selection, binding `version`, `access_ends_at`, `access_ended`) and blanks its `connection_id`, `Query.query_text` and `QueryJob.query_text`. A map whose workspace holds no active grant reads as ended.
- `ForkReport` copies each library dataset's binding at version 1 and keeps the library connection on it. A copy made in another workspace than the source's does not reuse the source's stored results, which were read under the source's grants; the copy runs only once its own workspace is granted.

## Not in this slice

A Postgres twin of the fork migration (the fork track runs on SQLite metadata only), so the library commands, stream decoration and binding copy are for SQLite metadata deployments.
