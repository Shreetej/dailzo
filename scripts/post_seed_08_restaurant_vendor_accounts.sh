#!/usr/bin/env bash
# Give every seeded restaurant an owning vendor account.
#
# The catalogue shipped with 20 restaurants whose restaurants.created_by was
# NULL. Vendor scope is resolved from that column, so an order for one of
# those restaurants belonged to nobody: no Partner-app account could list it,
# accept it or be paid for it. This creates one restaurant-vendor login per
# restaurant, points the restaurant at it, and writes the registration row the
# Partner app expects so the account lands on the dashboard rather than back
# in onboarding.
#
# Idempotent: an account that already exists is reused, and ownership is only
# filled in where it is missing.
#
# Usage:
#   DAILZO_VENDOR_PASSWORD='<password>' ./scripts/post_seed_08_restaurant_vendor_accounts.sh
# The backend must be running (it hashes the passwords); DB settings come
# from .env.
set -euo pipefail

cd "$(dirname "$0")/.."

API="${DAILZO_API:-http://localhost:2193/api/v1}"
PASSWORD="${DAILZO_VENDOR_PASSWORD:-}"
if [ -z "$PASSWORD" ]; then
    echo "DAILZO_VENDOR_PASSWORD is required (no default: it is a real credential)." >&2
    exit 1
fi

# .env is written with spaces around "=" on some lines, which `source` reads
# as a command. Strip them while loading.
set -a
eval "$(sed -E 's/[[:space:]]*=[[:space:]]*/=/; s/^[[:space:]]*//' .env | grep -E '^[A-Za-z_][A-Za-z0-9_]*=')"
set +a
export PGPASSWORD="$DB_PASSWORD"
psql_() { psql -h "${DB_HOST:-localhost}" -U "${DB_USER:-postgres}" \
               -d "${DB_NAME:-dailzo}" -p "${DB_PORT:-5432}" -qtA "$@"; }

if ! curl -fsS -m 5 "$API/health" >/dev/null; then
    echo "Backend is not answering at $API — start it first (./run-backend.sh --bg)." >&2
    exit 1
fi

printf '%-26s %-22s %-34s %s\n' RESTAURANT_ID EMAIL USER_ID RESULT

# Every restaurant, not only the unowned ones: a restaurant that already has
# an owner may still be missing the registration rows added below.
psql_ -c "SELECT id || E'\t' || name || E'\t' || COALESCE(created_by, '')
          FROM restaurants ORDER BY name;" |
while IFS=$'\t' read -r rid name owner; do
    [ -n "$rid" ] || continue

    if [ -n "$owner" ]; then
        # Already owned (a real vendor, or an earlier run) — keep that account.
        uid="$owner"
        email="$(psql_ -c "SELECT COALESCE(email, '') FROM users WHERE id = '$uid';")"
        result=owned
    else
        # RST_the_pizza_house -> the_pizza_house@dailzo.test
        slug="$(printf '%s' "$rid" | sed -E 's/^(RST|REST)_?//I' | tr '[:upper:]' '[:lower:]')"
        email="${slug}@dailzo.test"

        body="$(curl -fsS -X POST "$API/auth/signup" \
            -H 'Content-Type: application/json' \
            -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\",\"user_type\":\"restaurant\"}" \
            2>/dev/null || true)"
        uid="$(printf '%s' "$body" | python3 -c 'import sys,json
try: print((json.load(sys.stdin).get("data") or {}).get("id",""))
except Exception: print("")')"

        if [ -n "$uid" ]; then
            result=created
        else
            # Signed up on an earlier run but never linked — reuse it.
            uid="$(psql_ -c "SELECT id FROM users WHERE email = '$email';")"
            result=reused
            [ -n "$uid" ] || { printf '%-26s %-22s %-34s %s\n' "$rid" "$email" "-" FAILED; continue; }
        fi

        psql_ -c "UPDATE restaurants
                  SET created_by = '$uid', last_modified_by = '$uid', last_updated_on = NOW()
                  WHERE id = '$rid' AND (created_by IS NULL OR created_by = '');" >/dev/null
    fi

    # The row the Partner app's login gate reads (GET /vendor/registration).
    # Without it the vendor logs in and lands back on the registration wizard,
    # never on the dashboard.
    psql_ -c "INSERT INTO vendor_registrations (user_id, user_type, payload, status, created_on, last_updated_on)
              VALUES ('$uid', 'restaurant', jsonb_build_object(
                  'email', '$email',
                  'user_type', 'restaurant',
                  'first_name', \$n\$$name Owner\$n\$,
                  'restaurant_name', \$n\$$name\$n\$,
                  'restaurant_type', 'dine_in',
                  'cuisine_type', 'indian',
                  'city', 'Mumbai',
                  'address', \$n\$$name, Test Street\$n\$,
                  'pincode', '400001',
                  'mobile_no', '9800000000'
              ), 'approved', NOW(), NOW())
              ON CONFLICT (user_id) DO UPDATE
                  SET payload = EXCLUDED.payload || vendor_registrations.payload,
                      status = 'approved',
                      last_updated_on = NOW();" >/dev/null

    # The restaurant-side registration the onboarding screens read back.
    # Every key RestaurantRegistrationData declares `required` must be
    # present: the Partner app parses this payload with a freezed fromJson,
    # and one missing key throws, leaving the dashboard on its placeholder
    # "Restaurant Name" with no outlets.
    psql_ -c "INSERT INTO restaurant_registrations (id, user_id, payload, created_on, last_updated_on)
              VALUES ('$rid', '$uid', jsonb_build_object(
                  'id', '$rid',
                  'email', '$email',
                  'user_type', 'restaurant',
                  'restaurant_name', \$n\$$name\$n\$,
                  'restaurant_display_name', \$n\$$name\$n\$,
                  'owner_full_name', \$n\$$name Owner\$n\$,
                  'mobile_no', '9800000000',
                  'business_license_number', 'LIC-' || upper(left(md5('$rid'), 8)),
                  'vat_number', 'VAT-' || upper(left(md5('$rid'), 8)),
                  'bank_name', 'Test Bank',
                  'bank_account_number', '0000000000',
                  'bank_swift_code', 'TESTSWFT',
                  'outlets', '[]'::jsonb,
                  'account_status', 'approved',
                  'is_service_enabled', true,
                  'registration_completed', true,
                  'onboarding_fee', '0',
                  'onboarding_fee_status', 'waived'
              ), NOW(), NOW())
              -- Fill in missing keys only; anything the vendor has already
              -- set wins, so a re-run never overwrites real data.
              ON CONFLICT (id) DO UPDATE
                  SET payload = EXCLUDED.payload || restaurant_registrations.payload,
                      last_updated_on = NOW();" >/dev/null

    # Provenance on the restaurant's dishes. Vendor-facing product reads
    # match the owned business as well, so this is not what makes the menu
    # visible — it just stops every seeded product looking authorless.
    psql_ -c "UPDATE food_products
              SET created_by = '$uid', last_modified_by = '$uid'
              WHERE restaurant_id = '$rid' AND created_by IS NULL;" >/dev/null

    # One outlet per restaurant, built from the restaurant's own row.
    #
    # The Partner app's Orders tab lists outlets and will not show a single
    # order until one exists ("No outlets available"), and the dashboard
    # header reads its city from the first outlet. Seeded restaurants had
    # none, so their vendor could see the order count but never the orders.
    oid="OUT_$(printf '%s' "$rid" | sed -E 's/^(RST|REST)_?//')"
    psql_ -c "INSERT INTO vendor_outlets (id, restaurant_id, outlet_type, menu, cuisines,
                                          cost_for_two, avg_delivery_time, address,
                                          packaging_charge, operating_hours, is_active,
                                          created_on, last_updated_on, created_by)
              SELECT '$oid', r.id, 'delivery', 'both',
                     COALESCE(string_to_array(regexp_replace(r.cuisines, ', *', ',', 'g'), ','), '{}'),
                     200,
                     COALESCE(NULLIF(r.delivery_time, ''), '30-40'),
                     jsonb_build_object(
                         'street',  COALESCE(NULLIF(r.address, ''), r.name),
                         'city',    COALESCE(NULLIF(r.location, ''), 'Mumbai'),
                         'state',   'Maharashtra',
                         'zipCode', '400001',
                         'latitude',  '19.0760',
                         'longitude', '72.8777',
                         'country', 'India',
                         'restaurant_photo_url', COALESCE(r.image_path, '')),
                     jsonb_build_object('min_charge', 0, 'max_charge', 0),
                     (SELECT jsonb_object_agg(d, jsonb_build_object('open', '09:00', 'close', '23:00'))
                      FROM unnest(ARRAY['monday','tuesday','wednesday','thursday',
                                        'friday','saturday','sunday']) AS d),
                     true, NOW(), NOW(), '$uid'
              FROM restaurants r WHERE r.id = '$rid'
              ON CONFLICT (id) DO NOTHING;" >/dev/null

    # The dashboard reads outlets from the registration payload, not from
    # /restaurant/:id/outlets, so the same outlet has to appear there too.
    psql_ -c "UPDATE restaurant_registrations rr
              SET payload = jsonb_set(rr.payload, '{outlets}', (
                      SELECT jsonb_agg(jsonb_build_object(
                          'id', o.id,
                          'outlet_type', o.outlet_type,
                          'menu', o.menu,
                          'cuisines', to_jsonb(o.cuisines),
                          'cost_for_two', o.cost_for_two,
                          'avg_delivery_time', o.avg_delivery_time,
                          'address', o.address,
                          'packaging_charge', o.packaging_charge,
                          'operating_hours', o.operating_hours,
                          'is_active', o.is_active))
                      FROM vendor_outlets o WHERE o.restaurant_id = '$rid'),
                      true),
                  last_updated_on = NOW()
              WHERE rr.id = '$rid'
                AND EXISTS (SELECT 1 FROM vendor_outlets o WHERE o.restaurant_id = '$rid')
                AND COALESCE(jsonb_array_length(rr.payload->'outlets'), 0) = 0;" >/dev/null

    printf '%-26s %-22s %-34s %s\n' "$rid" "$email" "$uid" "$result"
done

echo
psql_ -c "SELECT count(*) || ' restaurant(s) still without an owner'
          FROM restaurants WHERE created_by IS NULL OR created_by = '';"
