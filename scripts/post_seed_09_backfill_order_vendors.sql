-- Back-fill vendor attribution on orders placed before it was recorded.
--
-- Older orders carry only product ids in their items snapshot: no per-line
-- vendor, and for grocery no vendor at all on the order row. They were
-- invisible to every Partner account. Both are recoverable from
-- food_products, which now names a restaurant or an outlet for every product.
--
-- Idempotent: lines that already carry attribution are left untouched, and
-- order columns are only filled where they are empty.

-- 1. Per-line vendor, so a multi-vendor order can be split by scope.
UPDATE orders o
SET items = sub.items,
    last_updated_on = NOW()
FROM (
    SELECT o2.id,
           jsonb_agg(
               CASE
                   WHEN line ? 'outlet_id' AND line ? 'restaurant_id' THEN line
                   ELSE line || jsonb_build_object(
                       'outlet_id',     COALESCE(btrim(p.outlet_id), ''),
                       'restaurant_id', COALESCE(btrim(p.restaurant_id), ''))
               END
               ORDER BY idx) AS items
    FROM orders o2
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(o2.items, '[]'::jsonb))
               WITH ORDINALITY AS t(line, idx)
    LEFT JOIN food_products p ON btrim(p.id) = btrim(line->>'product_id')
    WHERE jsonb_typeof(o2.items) = 'array'
    GROUP BY o2.id
    HAVING bool_or(NOT (line ? 'outlet_id' AND line ? 'restaurant_id'))
) sub
WHERE o.id = sub.id;

-- 2. The order's own vendor columns, from the first attributed line.
UPDATE orders o
SET restaurant_id = COALESCE(NULLIF(o.restaurant_id, ''), NULLIF(sub.rid, '')),
    outlet_id     = COALESCE(NULLIF(o.outlet_id, ''),     NULLIF(sub.oid, '')),
    last_updated_on = NOW()
FROM (
    SELECT o2.id,
           COALESCE(min(NULLIF(btrim(line->>'restaurant_id'), '')), '') AS rid,
           COALESCE(min(NULLIF(btrim(line->>'outlet_id'), '')), '')     AS oid
    FROM orders o2
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(o2.items, '[]'::jsonb)) AS line
    WHERE jsonb_typeof(o2.items) = 'array'
    GROUP BY o2.id
) sub
WHERE o.id = sub.id
  -- Only where this actually fills something in, so a re-run is a no-op.
  AND (((o.restaurant_id IS NULL OR o.restaurant_id = '') AND sub.rid <> '')
    OR ((o.outlet_id IS NULL OR o.outlet_id = '') AND sub.oid <> ''));

-- 3. Any line still without a vendor (a fixture order whose product ids were
--    never real) inherits the order's own vendor, so no line is left dangling.
UPDATE orders o
SET items = sub.items,
    last_updated_on = NOW()
FROM (
    SELECT o2.id,
           jsonb_agg(
               CASE
                   WHEN COALESCE(NULLIF(btrim(line->>'outlet_id'), ''),
                                 NULLIF(btrim(line->>'restaurant_id'), '')) IS NOT NULL
                   THEN line
                   ELSE line || jsonb_build_object(
                       'outlet_id',     COALESCE(btrim(o2.outlet_id), ''),
                       'restaurant_id', COALESCE(btrim(o2.restaurant_id), ''))
               END
               ORDER BY idx) AS items
    FROM orders o2
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(o2.items, '[]'::jsonb))
               WITH ORDINALITY AS t(line, idx)
    WHERE jsonb_typeof(o2.items) = 'array'
      AND COALESCE(NULLIF(btrim(o2.outlet_id), ''),
                   NULLIF(btrim(o2.restaurant_id), '')) IS NOT NULL
    GROUP BY o2.id
    HAVING bool_or(COALESCE(NULLIF(btrim(line->>'outlet_id'), ''),
                            NULLIF(btrim(line->>'restaurant_id'), '')) IS NULL)
) sub
WHERE o.id = sub.id;

SELECT count(*) FILTER (WHERE (restaurant_id IS NULL OR restaurant_id = '')
                          AND (outlet_id IS NULL OR outlet_id = '')) AS orders_with_no_vendor,
       count(*) AS orders_total
FROM orders;
