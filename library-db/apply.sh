#!/usr/bin/env bash
# Apply library-db/sql/*.sql in order to the library database.
# Re-runnable. Needs psql and an owner-role connection string in LIBRARY_ADMIN_URL
# (read from the environment; never put it in a file).
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${LIBRARY_ADMIN_URL:?set LIBRARY_ADMIN_URL to the owner connection string of the library database}"

for f in "$here"/sql/[0-9][0-9][0-9]_*.sql; do
  echo "applying $(basename "$f")"
  psql "$LIBRARY_ADMIN_URL" -X -q -v ON_ERROR_STOP=1 -1 -f "$f"
done

psql "$LIBRARY_ADMIN_URL" -X -q -v ON_ERROR_STOP=1 -f "$here/verify.sql"
echo "ok"
