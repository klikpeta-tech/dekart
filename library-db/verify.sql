-- Read-only checks of the foundation. Raises an exception on the first failure.
-- Run as the owner role; apply.sh runs it last.
DO $$
DECLARE
  r record;
  missing text;
BEGIN
  SELECT * INTO r FROM pg_roles WHERE rolname = 'dekart_library_reader';
  IF NOT FOUND THEN RAISE EXCEPTION 'role dekart_library_reader is missing'; END IF;
  IF r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolbypassrls OR r.rolreplication THEN
    RAISE EXCEPTION 'dekart_library_reader has an elevated attribute';
  END IF;
  IF NOT r.rolcanlogin OR r.rolconnlimit <> 5 THEN
    RAISE EXCEPTION 'dekart_library_reader must log in with connection limit 5 (got %)', r.rolconnlimit;
  END IF;
  IF pg_has_role('dekart_library_reader', 'neon_superuser', 'MEMBER') THEN
    RAISE EXCEPTION 'dekart_library_reader is a member of neon_superuser';
  END IF;
  IF NOT (r.rolconfig @> ARRAY['default_transaction_read_only=on', 'statement_timeout=60s']) THEN
    RAISE EXCEPTION 'dekart_library_reader role settings are wrong: %', r.rolconfig;
  END IF;

  SELECT string_agg(s, ', ') INTO missing
  FROM unnest(ARRAY['data', 'lib', 'lib_catalog', 'staging']) s
  WHERE NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = s);
  IF missing IS NOT NULL THEN RAISE EXCEPTION 'missing schemas: %', missing; END IF;

  SELECT string_agg(t, ', ') INTO missing
  FROM unnest(ARRAY['item', 'release', 'area', 'item_area_count', 'category_group',
                    'category', 'brand', 'poi_count']) t
  WHERE to_regclass('lib_catalog.' || t) IS NULL;
  IF missing IS NOT NULL THEN RAISE EXCEPTION 'missing lib_catalog tables: %', missing; END IF;

  IF NOT has_schema_privilege('dekart_library_reader', 'lib', 'USAGE')
     OR NOT has_schema_privilege('dekart_library_reader', 'lib_catalog', 'USAGE') THEN
    RAISE EXCEPTION 'reader lacks USAGE on lib or lib_catalog';
  END IF;
  IF has_schema_privilege('dekart_library_reader', 'data', 'USAGE')
     OR has_schema_privilege('dekart_library_reader', 'staging', 'USAGE') THEN
    RAISE EXCEPTION 'reader can reach data or staging';
  END IF;
  IF has_schema_privilege('dekart_library_reader', 'lib', 'CREATE')
     OR has_schema_privilege('dekart_library_reader', 'lib_catalog', 'CREATE')
     OR has_schema_privilege('dekart_library_reader', 'public', 'CREATE') THEN
    RAISE EXCEPTION 'reader can create objects';
  END IF;
  IF has_database_privilege('dekart_library_reader', current_database(), 'TEMPORARY') THEN
    RAISE EXCEPTION 'reader can create temporary tables';
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_database d, aclexplode(d.datacl) a
    WHERE d.datname = current_database() AND a.grantee = 0
  ) THEN
    RAISE EXCEPTION 'PUBLIC still holds a privilege on the database';
  END IF;
END
$$;
