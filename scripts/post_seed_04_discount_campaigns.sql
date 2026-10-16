-- Demo discount campaigns, so offer messaging comes from real data.
--
-- The customer app used to label EVERY restaurant "Flat 15% off on all
-- orders" with a fabricated FLAT15 code, written into the catalog mapping —
-- an offer no vendor had agreed to and no backend record supported. Real
-- campaigns live here and are editable from the vendor/admin marketing screens.
--
-- Idempotent: ids are derived from the restaurant id.
INSERT INTO public.discount_campaigns
    (id, restaurant_id, code, type, segment, percent, flat_amount,
     min_order_value, capping_amount, start_date, end_date, status)
SELECT
    'DISC_' || md5(btrim(r.id) || ':launch'),
    r.id,
    'SAVE15',
    'percentage',
    'all',
    15,
    NULL,
    50,
    40,
    to_char(CURRENT_DATE, 'YYYY-MM-DD'),
    to_char(CURRENT_DATE + INTERVAL '90 days', 'YYYY-MM-DD'),
    'active'
FROM public.restaurants r
WHERE r.is_active
ON CONFLICT (id) DO UPDATE
    SET percent = EXCLUDED.percent,
        status = EXCLUDED.status,
        end_date = EXCLUDED.end_date,
        last_updated_on = CURRENT_TIMESTAMP;
