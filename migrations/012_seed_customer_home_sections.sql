-- 012_seed_customer_home_sections.sql
-- Seed the customer-home content blocks that were hardcoded in the app's
-- food_screen (promo banner texts + section headers), so an admin can edit them
-- and they localize via content_translations. Idempotent (ON CONFLICT).
INSERT INTO public.content_sections (id, surface, section_key, type, title, subtitle, payload, sort_order, is_active) VALUES
    ('sec_home_promo_primary',   'customer_home', 'promo_primary',   'banner', 'FOODIE WEEKENDS', '', '{}'::jsonb, 1, true),
    ('sec_home_promo_secondary', 'customer_home', 'promo_secondary', 'banner', 'FLAT 150 OFF',    '', '{}'::jsonb, 2, true),
    ('sec_home_promo_cta',       'customer_home', 'promo_cta',       'banner', 'ORDER NOW',       '', '{}'::jsonb, 3, true),
    ('sec_home_todays_featured', 'customer_home', 'todays_featured', 'header', 'Today''s Featured','', '{}'::jsonb, 4, true),
    ('sec_home_whats_on_mind',   'customer_home', 'whats_on_mind',   'header', 'WHAT''S ON YOUR MIND?','', '{}'::jsonb, 5, true)
ON CONFLICT (surface, section_key) DO NOTHING;

-- Arabic (manual = reviewed, never clobbered by auto-translation).
INSERT INTO public.content_translations (id, entity_type, entity_id, field, lang, value, source) VALUES
    ('ctr_sec_home_promo_primary_ar',   'section','sec_home_promo_primary','title','ar','عطلات الذوّاقة','manual'),
    ('ctr_sec_home_promo_secondary_ar', 'section','sec_home_promo_secondary','title','ar','خصم ١٥٠ ثابت','manual'),
    ('ctr_sec_home_promo_cta_ar',       'section','sec_home_promo_cta','title','ar','اطلب الآن','manual'),
    ('ctr_sec_home_todays_featured_ar', 'section','sec_home_todays_featured','title','ar','مميّزات اليوم','manual'),
    ('ctr_sec_home_whats_on_mind_ar',   'section','sec_home_whats_on_mind','title','ar','ماذا يدور في ذهنك؟','manual')
ON CONFLICT (entity_type, entity_id, field, lang) DO NOTHING;
