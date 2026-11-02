-- Behaviour checks, run as dekart_library_reader (not the owner):
--   psql "$LIBRARY_READER_URL" -X -f smoke_reader.sql
-- Each statement after the first group must FAIL; psql prints the error and goes on.
SELECT current_user, current_setting('default_transaction_read_only') AS read_only,
       current_setting('statement_timeout') AS statement_timeout;
SELECT count(*) AS items FROM lib_catalog.item;
SELECT count(*) AS areas FROM lib_catalog.area;

-- expected: permission denied for schema data / staging / public
SELECT 1 FROM data.pg_class_does_not_matter;
SELECT 1 FROM staging.pg_class_does_not_matter;
-- expected: cannot execute ... in a read-only transaction
CREATE TABLE lib.x (a int);
-- expected: permission denied (after switching the default off, in its own statement)
SET default_transaction_read_only = off;
CREATE TABLE lib.x (a int);
CREATE TABLE public.x (a int);
CREATE TEMP TABLE t (a int);
