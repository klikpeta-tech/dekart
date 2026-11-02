# Data Library server foundation

Server side of the Klikpeta Data Library (ENG-2164). The design and decisions are in ENG-2098; this page records the contract the client lanes build against.

## Pieces

| Piece | Where |
| --- | --- |
| Proto messages and RPCs | `proto/dekart.proto`, "Data Library" blocks |
| Grant log and dataset bindings | `sqlite/klikpeta-migrations/000002_data_library.up.sql` (fork track, `klikpeta_schema_migrations`) |
| Grant store, binding store, expiry, catalogue interface | `src/server/library` |
| Pure SQL builder (dark library) | `src/server/dark/libraryquery` (generated `README.md`) |
| gRPC handlers | `src/server/dekart/library.go` (client), `library_admin.go` (admin) |

## Rules the handlers enforce

- Grants are per workspace and item, with a required end date. Access lasts through 23:59:59 Asia/Jakarta of that day. The latest log row (highest `seq`) wins.
- The grant checked for a map is the grant of the map's own workspace, not the caller's.
- Reads (`ListLibraryItems`, `GetLibraryItem`, `ListLibraryAreas`, `CountLibrarySelection`, `GetLibrarySample`) hide items without an active grant: an unknown item, an ungranted one and an ended one all answer `NotFound`.
- `AddLibraryDataset` and `UpdateLibraryDatasetSelection` need write access to the map; an ended grant answers `FailedPrecondition`, an ungranted category `PermissionDenied`, a selection over the item's `max_rows` `FailedPrecondition`, a stale `expected_version` `Canceled`.
- Admin commands (`ListLibraryGrants`, `GetLibraryTaxonomy`, `SetLibraryGrant`, `RevokeLibraryGrant`, `SetLibraryGrantCategories`) need the workspace Admin role and the caller's email to equal `DEKART_DEFAULT_WORKSPACE_ADMIN` (Klikpeta staff). An unset setting refuses everyone.
- Items with a taxonomy (Places) need at least one granted category; other items take none.
- `DEKART_LIBRARY_CONNECTION_ID` names the staff-owned library connection. `CreateQuery`, `CreateFile`, `UpdateQuery`, `RunQuery` (including the MCP run path) and `RunAllQueries` refuse it, so user SQL never runs on it.
- Every library command answers `Unavailable` until the catalogue reader is wired in with `Server.SetLibraryCatalogue`.

## Not in this slice

The catalogue reader against the library database, the stream decoration (`Dataset.library`, blanking the library connection id and SQL), the `RunQuery` choke point that rebuilds the SQL and checks the grant on every run, re-running after `UpdateLibraryDatasetSelection`, the `ForkReport` binding copy, and a Postgres twin of the fork migration (the fork track runs on SQLite metadata only).
