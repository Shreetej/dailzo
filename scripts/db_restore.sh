#!/usr/bin/env bash
# Restores a dump into the hosted database, once.
#
# Takes the connection string rather than reading .env, so there is no way to
# point it at the local database by accident, and names the target before
# doing anything — this overwrites every table in the dump.
#
# Usage:
#   ./scripts/db_restore.sh dailzo-20261016.dump.sql "$DATABASE_PUBLIC_URL"
#
# On Railway: copy DATABASE_PUBLIC_URL from the Postgres service's Variables
# tab (the internal DATABASE_URL only resolves from inside Railway's network).
set -euo pipefail

DUMP="${1:-}"
TARGET="${2:-}"

if [ -z "$DUMP" ] || [ -z "$TARGET" ]; then
    echo "usage: $0 <dump.sql> <postgres-connection-url>" >&2
    exit 1
fi
if [ ! -f "$DUMP" ]; then
    echo "no such dump: $DUMP" >&2
    exit 1
fi

# A bare host:port is the easy mistake here, and psql does not call it out:
# it treats the argument as a DATABASE NAME and fails with
# `database "autorack.proxy.rlwy.net:13603" does not exist`, which reads like
# the database is missing rather than like the argument is the wrong shape.
case "$TARGET" in
    postgres://*|postgresql://*) ;;
    *)
        cat >&2 <<MSG
That is not a connection URL: $TARGET

It needs the whole thing — scheme, user, password, host, port and database:

  postgresql://postgres:<password>@<host>:<port>/railway

On Railway: open the Postgres service → Variables → DATABASE_PUBLIC_URL
(the "Connect" panel shows it too). Use the PUBLIC one; the internal
DATABASE_URL only resolves from inside Railway's network.

Then:
  $0 $DUMP "<that URL>"
MSG
        exit 1
        ;;
esac

# Everything after @, up to the next / or ? — i.e. host[:port].
HOST="$(printf '%s' "$TARGET" | sed -E 's#^[^@]*@##; s#[/?].*$##')"
if [ -z "$HOST" ] || [ "$HOST" = "$TARGET" ]; then
    echo "Could not read a host out of that URL — nothing was changed." >&2
    exit 1
fi

# Show where this is about to go, with the password hidden.
SAFE="$(printf '%s' "$TARGET" | sed -E 's#(//[^:]+:)[^@]+@#\1xxxxx@#')"
echo "Restoring $DUMP into:"
echo "  $SAFE"
printf 'Type the host to confirm (%s): ' "$HOST"
read -r CONFIRM
if [ "$CONFIRM" != "$HOST" ]; then
    echo "That did not match $HOST — nothing was changed." >&2
    exit 1
fi

# ON_ERROR_STOP so a failed statement aborts instead of leaving the database
# half-populated and looking fine.
psql "$TARGET" -v ON_ERROR_STOP=1 -f "$DUMP"

echo
echo "Restored. Checking the invariants the data has to satisfy:"
psql "$TARGET" -c "
SELECT 'restaurants without an owner' AS check, count(*) FROM restaurants WHERE created_by IS NULL OR created_by = ''
UNION ALL SELECT 'products without a vendor', count(*) FROM food_products
    WHERE COALESCE(NULLIF(btrim(restaurant_id),''), NULLIF(btrim(outlet_id),'')) IS NULL
UNION ALL SELECT 'orders without a vendor', count(*) FROM orders
    WHERE (restaurant_id IS NULL OR restaurant_id = '') AND (outlet_id IS NULL OR outlet_id = '');"
