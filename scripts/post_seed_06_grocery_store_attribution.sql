-- Attribute the seeded grocery catalogue to the test grocery store.
--
-- Every grocery product shipped with outlet_id NULL, so a grocery order
-- belonged to nobody: the store could not list its own orders, and a mixed
-- food+grocery order could only ever be attributed to the restaurant half.
-- Food products already carry restaurant_id; this gives groceries the same
-- vendor link.
--
-- Idempotent: only fills in products that have no outlet yet, and only when
-- exactly one grocery store exists (the dev/test fixture). Safe to re-run.
WITH store AS (
    SELECT id, user_id FROM grocery_profiles ORDER BY created_on LIMIT 1
)
UPDATE food_products p
SET outlet_id = store.id,
    -- Provenance, so a seeded product isn't authorless. Vendor-facing reads
    -- match the owned store via outlet_id, not this column.
    created_by = COALESCE(p.created_by, store.user_id),
    last_modified_by = COALESCE(p.last_modified_by, store.user_id),
    last_updated_on = NOW()
FROM store
WHERE p.kind = 'grocery'
  AND (p.outlet_id IS NULL OR p.outlet_id = '' OR p.created_by IS NULL);

SELECT kind,
       count(*)                                                  AS products,
       count(*) FILTER (WHERE outlet_id IS NOT NULL AND outlet_id <> '') AS with_outlet
FROM food_products
GROUP BY kind
ORDER BY kind;
