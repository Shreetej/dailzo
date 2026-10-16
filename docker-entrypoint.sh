#!/bin/sh
# Applies pending migrations, then starts the server.
#
# Running them here rather than inside the server keeps one invariant: the
# schema is current before the first request is served, and a migration that
# fails stops the deploy instead of leaving a half-migrated database serving
# traffic. cmd/migrate tracks what it has applied in schema_migrations, so
# this is a no-op on a redeploy that changed no SQL.
set -e

if [ "${RUN_MIGRATIONS:-true}" = "true" ]; then
    echo "==> applying database migrations"
    /app/migrate -all
else
    echo "==> RUN_MIGRATIONS=false, skipping migrations"
fi

echo "==> starting dailzo backend"
exec /app/server
