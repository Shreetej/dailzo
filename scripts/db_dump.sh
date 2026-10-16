#!/usr/bin/env bash
# Dumps the local development database to a single restorable file.
#
# This is how the hosted database gets its data: the catalogue, the vendor
# accounts and every attribution fix live in rows, not in the repo, so a fresh
# managed Postgres starts empty even though the schema migrates itself.
#
# --no-owner / --no-privileges: the roles on the local box ("shree",
# "postgres") do not exist on a managed instance, and without this every
# ALTER ... OWNER TO in the dump fails on restore.
#
# Usage:
#   ./scripts/db_dump.sh [output.sql]      # defaults to dailzo-<date>.dump.sql
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="${1:-dailzo-$(date +%Y%m%d-%H%M).dump.sql}"

set -a
eval "$(sed -E 's/[[:space:]]*=[[:space:]]*/=/; s/^[[:space:]]*//' .env | grep -E '^[A-Za-z_][A-Za-z0-9_]*=')"
set +a

# The local pg_dump is newer than the managed server this restores into
# (Homebrew ships 18; Railway runs 16/17), and a newer pg_dump emits SET
# statements for parameters an older server has never heard of —
# `transaction_timeout` arrives in 17 and makes psql abort on line 13 of the
# dump. These settings are dump-time safety rails, not data, so dropping the
# ones the target may not know keeps the dump restorable across versions.
PGPASSWORD="$DB_PASSWORD" pg_dump \
    -h "${DB_HOST:-localhost}" \
    -p "${DB_PORT:-5432}" \
    -U "${DB_USER:-postgres}" \
    -d "${DB_NAME:-dailzo}" \
    --no-owner --no-privileges --clean --if-exists \
  | grep -v -E "^SET (transaction_timeout|idle_session_timeout) = " \
  > "$OUT"

echo "Wrote $OUT ($(du -h "$OUT" | cut -f1))"
echo
echo "This file contains real data, including bcrypt password hashes."
echo "Do not commit it."
