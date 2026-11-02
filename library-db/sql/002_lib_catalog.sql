-- Data Library database, step 2: catalogue tables in lib_catalog.
--
-- Re-runnable (CREATE ... IF NOT EXISTS). Changing an existing table needs a new,
-- numbered file, never an edit here once applied to a shared database.

-- One row per dataset offered in the library.
CREATE TABLE IF NOT EXISTS lib_catalog.item (
  id                text PRIMARY KEY,
  title             text NOT NULL,
  description_md    text NOT NULL DEFAULT '',
  category          text NOT NULL DEFAULT '',
  provider          text NOT NULL DEFAULT '',
  attribution       text NOT NULL DEFAULT '',
  shape_kind        text NOT NULL CHECK (shape_kind IN ('points', 'lines', 'areas', 'h3')),
  admin_level       text,                       -- for areas: provinsi, kabkota, kecamatan
  h3_resolution     int,                        -- for h3
  coverage_summary  text NOT NULL DEFAULT '',
  row_count         bigint,
  source_updated_on date,
  cadence           text NOT NULL DEFAULT '',
  columns           jsonb NOT NULL DEFAULT '[]'::jsonb,
  view_name         text NOT NULL,              -- lib.<view_name>
  area_levels       text[] NOT NULL DEFAULT '{}',
  has_taxonomy      boolean NOT NULL DEFAULT false,
  max_rows          int,                        -- per-dataset row cap Dekart enforces
  layer_style       jsonb,                      -- layer template applied on first load
  status            text NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'retired')),
  sort_order        int NOT NULL DEFAULT 0,
  release_id        text,                       -- current release
  CONSTRAINT item_area_levels_valid
    CHECK (area_levels <@ ARRAY['provinsi', 'kabkota', 'kecamatan']::text[]),
  CONSTRAINT item_view_name_safe CHECK (view_name ~ '^[a-z][a-z0-9_]*$')
);

-- Release history per item (promote, roll back, retire; Pustaka plan 3.4).
CREATE TABLE IF NOT EXISTS lib_catalog.release (
  release_id        text NOT NULL,
  item_id           text NOT NULL REFERENCES lib_catalog.item (id),
  published_on      date,
  source_vintage    text NOT NULL DEFAULT '',
  source_checksum   text NOT NULL DEFAULT '',
  row_count         bigint,
  qa_report_url     text NOT NULL DEFAULT '',
  changelog_md      text NOT NULL DEFAULT '',
  status            text NOT NULL DEFAULT 'staged'
                    CHECK (status IN ('staged', 'current', 'archived', 'withdrawn')),
  PRIMARY KEY (item_id, release_id)
);

-- BPS administrative areas. Codes are BPS codes without the "ID" prefix:
-- provinsi 2 digits, kabkota 4, kecamatan 7. bps2020 first.
CREATE TABLE IF NOT EXISTS lib_catalog.area (
  kode              text PRIMARY KEY,
  level             text NOT NULL CHECK (level IN ('provinsi', 'kabkota', 'kecamatan')),
  nama              text NOT NULL,
  parent_kode       text REFERENCES lib_catalog.area (kode),
  boundary_vintage  text NOT NULL,
  CONSTRAINT area_kode_matches_level CHECK (
    (level = 'provinsi'  AND kode ~ '^[0-9]{2}$' AND parent_kode IS NULL) OR
    (level = 'kabkota'   AND kode ~ '^[0-9]{4}$' AND parent_kode = left(kode, 2)) OR
    (level = 'kecamatan' AND kode ~ '^[0-9]{7}$' AND parent_kode = left(kode, 4))
  )
);
CREATE INDEX IF NOT EXISTS area_parent_idx ON lib_catalog.area (parent_kode);
CREATE INDEX IF NOT EXISTS area_level_nama_idx ON lib_catalog.area (level, nama);

-- Row counts of an item per area, for the estimate before a dataset is added.
CREATE TABLE IF NOT EXISTS lib_catalog.item_area_count (
  item_id  text   NOT NULL REFERENCES lib_catalog.item (id),
  kode     text   NOT NULL REFERENCES lib_catalog.area (kode),
  n        bigint NOT NULL,
  PRIMARY KEY (item_id, kode)
);
CREATE INDEX IF NOT EXISTS item_area_count_kode_idx ON lib_catalog.item_area_count (kode);

-- Taxonomy for Places: groups, categories, brands.
CREATE TABLE IF NOT EXISTS lib_catalog.category_group (
  id          text PRIMARY KEY,
  title       text NOT NULL,
  sort_order  int  NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS lib_catalog.category (
  id          text PRIMARY KEY,
  group_id    text NOT NULL REFERENCES lib_catalog.category_group (id),
  title       text NOT NULL,
  sort_order  int  NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS category_group_idx ON lib_catalog.category (group_id);

-- Includes one 'independen' bucket per category for unbranded places.
CREATE TABLE IF NOT EXISTS lib_catalog.brand (
  id           text PRIMARY KEY,
  category_id  text NOT NULL REFERENCES lib_catalog.category (id),
  title        text NOT NULL
);
CREATE INDEX IF NOT EXISTS brand_category_idx ON lib_catalog.brand (category_id);

-- POI counts at kecamatan grain; kab/kota and provinsi roll up by code prefix.
CREATE TABLE IF NOT EXISTS lib_catalog.poi_count (
  kode_kecamatan  text NOT NULL REFERENCES lib_catalog.area (kode),
  category_id     text NOT NULL REFERENCES lib_catalog.category (id),
  brand_id        text NOT NULL REFERENCES lib_catalog.brand (id),
  n               int  NOT NULL,
  PRIMARY KEY (kode_kecamatan, category_id, brand_id)
);
CREATE INDEX IF NOT EXISTS poi_count_category_idx ON lib_catalog.poi_count (category_id, brand_id);

-- The reader sees every catalogue table. ALTER DEFAULT PRIVILEGES in 001 covers
-- tables created later by the same owner; this covers re-runs after a role change.
GRANT SELECT ON ALL TABLES IN SCHEMA lib_catalog TO dekart_library_reader;
