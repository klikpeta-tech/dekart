# library-db

Database setup for the Klikpeta Data Library: the PostGIS database that Dekart's staff-owned library connection reads. It is a separate Neon project from Dekart's own metadata database.

| | |
|---|---|
| Neon project | `klikpeta-geo-library` (id `flat-leaf-69431132`, org Klikpeta) |
| Region / version | `aws-ap-southeast-1` (Singapore), Postgres 17 |
| Database | `library`, default branch `main` only |
| Compute | autoscaling 0.25 to 2 CU, scale to zero after 300 s idle |
| Extension | PostGIS (in `public`) |

Design: ENG-2098 technical plan sections 6.2 and 6.3, Pustaka Data plan sections 2.3 and 3.4.

## Layout

```
sql/001_schemas_and_roles.sql   PostGIS, schemas, dekart_library_reader, privileges
sql/002_lib_catalog.sql         lib_catalog tables
apply.sh                        applies sql/*.sql in order, then verify.sql
verify.sql                      fails loudly if roles, schemas or privileges drifted
smoke_reader.sql                behaviour checks to run as the reader
loaders/build_area_csv.py       builds the area CSV from the pinned bps2020 boundaries
loaders/load_area.sql           loads that CSV into lib_catalog.area
```

Every file is re-runnable. Changing an object that exists in a shared database needs a new numbered file, not an edit to an applied one. Applied files are listed in `lib_catalog.schema_migration`.

## Schemas

| Schema | Holds | Reader |
|---|---|---|
| `data` | One table per dataset, partitioned by `release_id` | no access |
| `lib` | Stable views Dekart queries (`lib.<view_name>`) | `SELECT` |
| `lib_catalog` | `item`, `release`, `area`, `item_area_count`, `category_group`, `category`, `brand`, `poi_count` | `SELECT` |
| `staging` | Releases being loaded and checked | no access |

`lib_catalog.area` holds BPS codes without the `ID` prefix: provinsi 2 digits, kab/kota 4, kecamatan 7. `poi_count` is at kecamatan grain; higher levels roll up by code prefix.

## The reader role

`dekart_library_reader` is the only login role Dekart uses. It has `SELECT` on `lib` and `lib_catalog`, no access to `data` or `staging`, cannot create anything (including temp tables), and its attributes are checked by `verify.sql`. Role settings: `default_transaction_read_only = on`, `statement_timeout = '60s'`, connection limit 5.

Roles are created by this script, never in the Neon console: a console-created role gets `CREATEDB`, `CREATEROLE`, `BYPASSRLS` and `neon_superuser`.

Default privileges make later views and tables in `lib` and `lib_catalog` readable without a re-grant, but only for objects created by the role that ran `001`. The pipeline must create them as that same owner (`library_owner`).

### Reader password

The scripts never set a password, so the role cannot log in until someone sets one. From a trusted terminal, using the owner connection string from the environment:

```sh
psql "$LIBRARY_ADMIN_URL" -c '\password dekart_library_reader'
```

`\password` hashes the password on the client, so it does not appear in the server log. Store the result in the secret store, not in this repository. Then run `smoke_reader.sql` as the reader: the first three queries must succeed and every later statement must fail.

## Apply

```sh
export LIBRARY_ADMIN_URL=...   # owner role of database library; keep it out of files and history
library-db/apply.sh
```

Note for non-superuser owners (Neon): the owner cannot run `ALTER ROLE ... NOSUPERUSER`, and cannot `SET ROLE` to the reader, so the script relies on `CREATE ROLE` defaults for the first and on `verify.sql` privilege probes for the second.

## Loading data

### Area (`lib_catalog.area`)

Source: HDX COD-AB Indonesia (BPS via OCHA), vintage `bps2020`, valid from 2020-04-01, licence CC BY-IGO 3.0, archive SHA-256 `d285a40f9fc4b726b1bda98ee833d52d23d94fe78b6a47c0a769d2e912ff0970`.

```sh
python3 -I library-db/loaders/build_area_csv.py \
  --source <dir with idn_admin1.geojson, idn_admin2.geojson, idn_admin3.geojson> \
  --zip <idn_admin_boundaries.geojson.zip> --out "$TMPDIR/area.csv"
(cd library-db/loaders && psql "$LIBRARY_ADMIN_URL" -X -v ON_ERROR_STOP=1 -v csv="$TMPDIR/area.csv" -f load_area.sql)
```

The builder checks the archive hash, drops placeholder areas (forest, lake and island codes) and expects 34 provinsi, 514 kab/kota and 7,031 kecamatan; the loader checks the same counts after loading and fails if they differ.

### Status of the first data

- The area CSV builds from the pinned local copy and passes the count check (7,579 rows, no orphan parents), but it is **not yet loaded** into the new project: the only available write path in the task that created this folder was the Neon MCP, which cannot stream a 320 KB file, and the connection string must not be pasted into a session. Run the two commands above with the owner connection string.
- The two first items (boundaries and kecamatan BPS 2024) are **not loaded**. The boundary geometry is large and no local copy of a kecamatan BPS 2024 table was found. Their `data`, `lib` and `item` rows belong to the release pipeline.
