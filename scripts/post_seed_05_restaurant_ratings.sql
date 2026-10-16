-- Demo customer ratings.
--
-- The customer app used to display a flat "4.2 (100 ratings)" for every
-- restaurant — a client-side fallback, not data. With that removed, these
-- give the apps something real to show; the aggregate on restaurants
-- (rating / total_ratings) is recomputed from the ratings rows below.
INSERT INTO public.ratings (id, rating, comment, user_id, entity_type, entity_id, created_on, last_updated_on)
SELECT
    'RTNG_' || md5(btrim(r.id) || ':' || g.n),
    -- Deterministic spread of 3–5 stars so each restaurant differs.
    3 + ((abs(hashtext(btrim(r.id) || g.n::text)) % 3)),
    NULL,
    u.id,
    'restaurant',
    btrim(r.id),
    CURRENT_TIMESTAMP - (g.n || ' days')::interval,
    CURRENT_TIMESTAMP
FROM public.restaurants r
CROSS JOIN generate_series(1, 12) AS g(n)
CROSS JOIN LATERAL (
    SELECT id FROM public.users WHERE user_type = 'customer' LIMIT 1
) u
WHERE r.is_active
ON CONFLICT (id) DO NOTHING;

-- Keep the denormalised aggregate on restaurants in step.
UPDATE public.restaurants r
SET rating = agg.avg_rating,
    total_ratings = agg.cnt
FROM (
    SELECT entity_id, round(avg(rating)::numeric, 1)::float8 AS avg_rating, count(*) AS cnt
    FROM public.ratings
    WHERE entity_type = 'restaurant'
    GROUP BY entity_id
) agg
WHERE btrim(r.id) = agg.entity_id;
