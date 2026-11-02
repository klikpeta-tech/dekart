-- Data Library database, step 1: extension, schemas, the read-only login role.
--
-- Re-runnable: every statement is idempotent. Run as the database owner
-- (the Neon project owner role), connected to the `library` database.
-- Never run this from the Neon console role wizard: a console-created role gets
-- CREATEDB, CREATEROLE, BYPASSRLS and neon_superuser (the B7 lesson).
--
-- Does not set a password. See README.md, "Reader password".

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE SCHEMA IF NOT EXISTS data;        -- one table per dataset, partitioned by release_id (pipeline and owner only)
CREATE SCHEMA IF NOT EXISTS lib;         -- stable views Dekart queries
CREATE SCHEMA IF NOT EXISTS lib_catalog; -- catalogue Dekart reads
CREATE SCHEMA IF NOT EXISTS staging;     -- releases being loaded and checked (pipeline only)

-- The single login role Dekart's library connection uses.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'dekart_library_reader') THEN
    CREATE ROLE dekart_library_reader LOGIN;
  END IF;
END
$$;

-- Pin the attributes on each run, so drift is corrected rather than kept.
-- NOSUPERUSER is not listed: CREATE ROLE already defaults to it, and a non-superuser
-- owner (Neon) may not run ALTER ROLE ... NOSUPERUSER. verify.sql checks it.
ALTER ROLE dekart_library_reader
  WITH LOGIN NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS NOINHERIT
  CONNECTION LIMIT 5;
ALTER ROLE dekart_library_reader SET default_transaction_read_only = on;
ALTER ROLE dekart_library_reader SET statement_timeout = '60s';

-- Close what PUBLIC gets by default, then open exactly what the reader needs.
DO $$
BEGIN
  EXECUTE format('REVOKE CONNECT, TEMPORARY ON DATABASE %I FROM PUBLIC', current_database());
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO dekart_library_reader', current_database());
END
$$;

-- PostGIS lives in public. Nobody but the owner may create objects there.
-- USAGE stays open so geometry types and spatial_ref_sys resolve.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA data, staging FROM PUBLIC;
REVOKE ALL ON SCHEMA lib, lib_catalog FROM PUBLIC;

GRANT USAGE ON SCHEMA lib, lib_catalog TO dekart_library_reader;

-- Objects the owner creates later in these schemas become readable without a re-grant.
ALTER DEFAULT PRIVILEGES IN SCHEMA lib
  GRANT SELECT ON TABLES TO dekart_library_reader;
ALTER DEFAULT PRIVILEGES IN SCHEMA lib_catalog
  GRANT SELECT ON TABLES TO dekart_library_reader;
