-- Seed customisation groups for restaurant (food) products.
--
-- These groups used to be const lists compiled into the customer app's
-- customisation screen, so every dish offered the same choices at invented
-- prices and nothing the customer picked could be priced or fulfilled. Moving
-- them into the catalog makes them real vendor-owned data: editable per
-- product, priced once on the server, and carried onto the order.
--
-- Idempotent: safe to re-run. Deterministic ids are derived from the product
-- id so a second run updates in place instead of duplicating.

-- ─── Portion (pick one) ──────────────────────────────────────────────────────
INSERT INTO public.product_option_groups (id, product_id, name, selection, is_required, sort_order)
SELECT 'OPTG_' || md5(btrim(fp.id) || ':portion'), fp.id, 'Portion', 'single', true, 1
FROM public.food_products fp
WHERE fp.kind = 'food' AND fp.is_active
ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name, selection = EXCLUDED.selection,
        is_required = EXCLUDED.is_required, sort_order = EXCLUDED.sort_order,
        last_updated_on = CURRENT_TIMESTAMP;

INSERT INTO public.product_options (id, group_id, label, price_delta, sort_order)
SELECT 'OPT_' || md5(btrim(fp.id) || ':portion:' || o.label),
       'OPTG_' || md5(btrim(fp.id) || ':portion'),
       o.label, o.delta, o.ord
FROM public.food_products fp
CROSS JOIN (VALUES ('Regular', 0.00, 1), ('Large', 90.00, 2)) AS o(label, delta, ord)
WHERE fp.kind = 'food' AND fp.is_active
ON CONFLICT (id) DO UPDATE
    SET label = EXCLUDED.label, price_delta = EXCLUDED.price_delta,
        sort_order = EXCLUDED.sort_order, last_updated_on = CURRENT_TIMESTAMP;

-- ─── Spice level (pick one, no price impact) ─────────────────────────────────
INSERT INTO public.product_option_groups (id, product_id, name, selection, is_required, sort_order)
SELECT 'OPTG_' || md5(btrim(fp.id) || ':spice'), fp.id, 'Spice level', 'single', true, 2
FROM public.food_products fp
WHERE fp.kind = 'food' AND fp.is_active
ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name, selection = EXCLUDED.selection,
        is_required = EXCLUDED.is_required, sort_order = EXCLUDED.sort_order,
        last_updated_on = CURRENT_TIMESTAMP;

INSERT INTO public.product_options (id, group_id, label, price_delta, sort_order)
SELECT 'OPT_' || md5(btrim(fp.id) || ':spice:' || o.label),
       'OPTG_' || md5(btrim(fp.id) || ':spice'),
       o.label, 0.00, o.ord
FROM public.food_products fp
CROSS JOIN (VALUES ('Mild', 1), ('Medium', 2), ('Spicy', 3)) AS o(label, ord)
WHERE fp.kind = 'food' AND fp.is_active
ON CONFLICT (id) DO UPDATE
    SET label = EXCLUDED.label, price_delta = EXCLUDED.price_delta,
        sort_order = EXCLUDED.sort_order, last_updated_on = CURRENT_TIMESTAMP;

-- ─── Add-ons (pick any) ──────────────────────────────────────────────────────
INSERT INTO public.product_option_groups (id, product_id, name, selection, is_required, sort_order)
SELECT 'OPTG_' || md5(btrim(fp.id) || ':addons'), fp.id, 'Add-ons', 'multi', false, 3
FROM public.food_products fp
WHERE fp.kind = 'food' AND fp.is_active
ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name, selection = EXCLUDED.selection,
        is_required = EXCLUDED.is_required, sort_order = EXCLUDED.sort_order,
        last_updated_on = CURRENT_TIMESTAMP;

INSERT INTO public.product_options (id, group_id, label, price_delta, sort_order)
SELECT 'OPT_' || md5(btrim(fp.id) || ':addons:' || o.label),
       'OPTG_' || md5(btrim(fp.id) || ':addons'),
       o.label, o.delta, o.ord
FROM public.food_products fp
CROSS JOIN (VALUES
    ('Extra gravy',  40.00, 1),
    ('Extra cheese', 50.00, 2),
    ('Side salad',   60.00, 3)
) AS o(label, delta, ord)
WHERE fp.kind = 'food' AND fp.is_active
ON CONFLICT (id) DO UPDATE
    SET label = EXCLUDED.label, price_delta = EXCLUDED.price_delta,
        sort_order = EXCLUDED.sort_order, last_updated_on = CURRENT_TIMESTAMP;
