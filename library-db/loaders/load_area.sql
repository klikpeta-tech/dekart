-- Load lib_catalog.area from the CSV made by build_area_csv.py.
-- Re-runnable: rows upsert by code, parents load before children.
-- Run with psql from library-db/loaders/, owner role:
--   psql "$LIBRARY_ADMIN_URL" -X -v ON_ERROR_STOP=1 -v csv=area.csv -f load_area.sql
BEGIN;

CREATE TEMP TABLE area_load (
  kode text, level text, nama text, parent_kode text, boundary_vintage text
) ON COMMIT DROP;

\set copycmd '\\copy area_load FROM ' :'csv' ' WITH (FORMAT csv, HEADER true)'
:copycmd

INSERT INTO lib_catalog.area (kode, level, nama, parent_kode, boundary_vintage)
SELECT kode, level, nama, NULLIF(parent_kode, ''), boundary_vintage
FROM area_load
ORDER BY CASE level WHEN 'provinsi' THEN 0 WHEN 'kabkota' THEN 1 ELSE 2 END, kode
ON CONFLICT (kode) DO UPDATE
  SET level = EXCLUDED.level, nama = EXCLUDED.nama,
      parent_kode = EXCLUDED.parent_kode, boundary_vintage = EXCLUDED.boundary_vintage;

DO $$
DECLARE n_prov int; n_kab int; n_kec int;
BEGIN
  SELECT count(*) FILTER (WHERE level = 'provinsi'),
         count(*) FILTER (WHERE level = 'kabkota'),
         count(*) FILTER (WHERE level = 'kecamatan')
    INTO n_prov, n_kab, n_kec
  FROM lib_catalog.area WHERE boundary_vintage = 'bps2020';
  IF (n_prov, n_kab, n_kec) <> (34, 514, 7031) THEN
    RAISE EXCEPTION 'unexpected bps2020 counts: % / % / %', n_prov, n_kab, n_kec;
  END IF;
END
$$;

COMMIT;
