# Test accounts

Every seeded business on a dev database has a Partner-app login. Before this,
20 of the 21 restaurants had no owner at all (`restaurants.created_by` was
NULL) and no grocery product named a store, so an order for any of them was
attributable to nobody: no account could list it, accept it or be paid for it.

All of these are **local fixtures**. They exist only in a dev database seeded
from `scripts/`, and nothing here should ever be created against a real
deployment.

## Password

One password is shared by every account below. It is the value of
`DAILZO_VENDOR_PASSWORD` passed when the seed was run — deliberately not
written down here, so the repo carries no credential. Ask whoever seeded the
database, or re-run the seed with a password of your choosing (step 3 below);
re-running only creates missing accounts, it does not reset existing ones.

## Accounts

| Email | Role | Business |
| --- | --- | --- |
| `admin@dailzo.test` | admin | — |
| `customer@dailzo.test` | customer | — |
| `rider@dailzo.test` | delivery_partner | — |
| `fleet@dailzo.test` | delivery_vendor | — |
| `grocery@dailzo.test` | grocery | Dailzo Test Mart (all 934 grocery products) |
| `restaurant@dailzo.test` | restaurant | Dailzo Test Kitchen |
| `bhajiwala@dailzo.test` | restaurant | Bhajiwala |
| `burger_point@dailzo.test` | restaurant | Burger Point |
| `darjeeling_momos@dailzo.test` | restaurant | Darjeeling Momos |
| `frozon_bottle@dailzo.test` | restaurant | Frozon Bottle |
| `gupta_sandwich@dailzo.test` | restaurant | Gupta Sandwich |
| `kolkata_rolls@dailzo.test` | restaurant | Kolkata Rolls |
| `mcdonald_s@dailzo.test` | restaurant | McDonald's |
| `momo_delight@dailzo.test` | restaurant | Momo Delight |
| `momo_magic@dailzo.test` | restaurant | Momo Magic |
| `momos_king@dailzo.test` | restaurant | Momos King |
| `natural_s_ice_cream@dailzo.test` | restaurant | Natural's Ice Cream |
| `oho_shawarma@dailzo.test` | restaurant | Oho Shawarma |
| `pav_bhaji_corner@dailzo.test` | restaurant | Pav Bhaji Corner |
| `sambar_delights@dailzo.test` | restaurant | Sambar Delights |
| `south_spice@dailzo.test` | restaurant | South Spice |
| `temptation_chinese@dailzo.test` | restaurant | Temptation Chinese |
| `the_biryani_house@dailzo.test` | restaurant | The Biryani House |
| `the_burger_beats@dailzo.test` | restaurant | The Burger Beats |
| `the_curry_bowl@dailzo.test` | restaurant | The Curry Bowl |
| `the_pizza_house@dailzo.test` | restaurant | The Pizza House |

The restaurant email is the restaurant id without its `RST_` prefix, so a new
restaurant's login is predictable.

Each restaurant vendor gets:

- a `users` row (`user_type` `restaurant`),
- `restaurants.created_by` pointing at it — this is what order scoping reads,
- a `vendor_registrations` row with `status = 'approved'`, which is what the
  Partner app's login gate reads (without it the vendor logs in and lands back
  on the registration wizard),
- a `restaurant_registrations` payload carrying every field the app's
  `RestaurantRegistrationData` declares `required` — one missing key makes its
  `fromJson` throw and the dashboard falls back to "Restaurant Name",
- one outlet in `vendor_outlets`, built from the restaurant's own address,
  cuisines and delivery time. The Orders tab lists outlets and shows nothing
  at all until one exists.

## Seeding

Run in order, against a database that already has the base seed, with the
backend running (it hashes the passwords):

1. `psql -f scripts/post_seed_06_grocery_store_attribution.sql` — points every
   grocery product at the grocery store.
2. `DAILZO_VENDOR_PASSWORD='…' ./scripts/post_seed_08_restaurant_vendor_accounts.sh`
   — one vendor login per restaurant, plus registration and outlet rows.
3. `psql -f scripts/post_seed_09_backfill_order_vendors.sql` — gives orders
   placed before per-line attribution existed a vendor, from their products.

All three are idempotent; re-running changes nothing that is already correct.

## Checking the invariant

```sql
SELECT 'restaurants without an owner', count(*) FROM restaurants
WHERE created_by IS NULL OR created_by = ''
UNION ALL SELECT 'products without a vendor', count(*) FROM food_products
WHERE COALESCE(NULLIF(btrim(restaurant_id),''), NULLIF(btrim(outlet_id),'')) IS NULL
UNION ALL SELECT 'orders without a vendor', count(*) FROM orders
WHERE (restaurant_id IS NULL OR restaurant_id = '')
  AND (outlet_id IS NULL OR outlet_id = '');
```

Every row should read 0.
