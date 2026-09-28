-- 011_i18n_and_admin_config.sql
-- Server-driven i18n (translate-on-write) + admin-configurable sections.
--
-- Three concerns, one migration:
--   1. content_translations — generic per-(entity,field,lang) translation store.
--      Filled automatically on write (source='auto') and overridable by an
--      admin/vendor edit (source='manual'), which auto-translation never clobbers.
--   2. content_sections    — admin-configurable home/dashboard content blocks,
--      rendered server-driven so sections can be added/removed without a release.
--   3. option_items        — admin-configurable enum/dropdown lists (cuisine,
--      outlet type, vehicle, shift, menu type) replacing hardcoded app lists.
--
-- Titles/labels for all three flow through content_translations by
-- (entity_type, entity_id, field), so admin-added content is auto-translated.

-- ── 1. Generic translation store ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS public.content_translations (
    id           VARCHAR(64) PRIMARY KEY,
    entity_type  VARCHAR(40)  NOT NULL,          -- 'category','product','section','option'
    entity_id    VARCHAR(64)  NOT NULL,
    field        VARCHAR(40)  NOT NULL,          -- 'name','description','title','subtitle','label'
    lang         VARCHAR(8)   NOT NULL,          -- BCP-47 base tag: 'en','ar'
    value        TEXT         NOT NULL,
    source       VARCHAR(10)  NOT NULL DEFAULT 'auto' CHECK (source IN ('auto','manual')),
    updated_on   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (entity_type, entity_id, field, lang)
);
CREATE INDEX IF NOT EXISTS idx_ctr_lookup
    ON public.content_translations (entity_type, entity_id, lang);

-- ── 2. Admin-configurable content sections (home/dashboard blocks) ───────────
CREATE TABLE IF NOT EXISTS public.content_sections (
    id            VARCHAR(50) PRIMARY KEY,
    surface       VARCHAR(40) NOT NULL,          -- 'customer_home','customer_grocery','vendor_dashboard'
    section_key   VARCHAR(60) NOT NULL,          -- stable key within a surface
    type          VARCHAR(30) NOT NULL,          -- 'banner','featured','tiles','carousel','promo'
    title         TEXT,                          -- base (English); translated via content_translations
    subtitle      TEXT,
    payload       JSONB       DEFAULT '{}'::jsonb, -- type-specific data (image_url, cta, item ids…)
    city          VARCHAR(60),                   -- nullable: optional city scoping
    sort_order    INT         DEFAULT 0,
    is_active     BOOLEAN     DEFAULT true,
    created_on    TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (surface, section_key)
);
CREATE INDEX IF NOT EXISTS idx_sections_surface
    ON public.content_sections (surface, is_active, sort_order);

-- ── 3. Admin-configurable option/enum lists ─────────────────────────────────
CREATE TABLE IF NOT EXISTS public.option_items (
    id          VARCHAR(50) PRIMARY KEY,
    set_kind    VARCHAR(40) NOT NULL,            -- 'cuisine','outlet_type','vehicle','shift','menu_type'
    code        VARCHAR(60) NOT NULL,            -- stable logic value the app persists
    label       TEXT        NOT NULL,            -- base (English); translated via content_translations
    sort_order  INT         DEFAULT 0,
    is_active   BOOLEAN     DEFAULT true,
    created_on  TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (set_kind, code)
);
CREATE INDEX IF NOT EXISTS idx_options_kind
    ON public.option_items (set_kind, is_active, sort_order);

-- ── Seed the option lists the vendor app currently hardcodes ─────────────────
-- codes are the exact values the app already persists/compares on, so existing
-- data and logic keep working; labels are the current English display text.
INSERT INTO public.option_items (id, set_kind, code, label, sort_order) VALUES
    ('opt_cuisine_indian',   'cuisine', 'indian',   'Indian',        1),
    ('opt_cuisine_chinese',  'cuisine', 'chinese',  'Chinese',       2),
    ('opt_cuisine_italian',  'cuisine', 'italian',  'Italian',       3),
    ('opt_cuisine_arabic',   'cuisine', 'arabic',   'Arabic',        4),
    ('opt_cuisine_mexican',  'cuisine', 'mexican',  'Mexican',       5),
    ('opt_cuisine_thai',     'cuisine', 'thai',     'Thai',          6),
    ('opt_cuisine_conti',    'cuisine', 'continental','Continental', 7),
    ('opt_otype_fine',       'outlet_type', 'fine_dining',   'Fine Dining',   1),
    ('opt_otype_qsr',        'outlet_type', 'quick_service', 'Quick Service', 2),
    ('opt_otype_cafe',       'outlet_type', 'cafe',          'Cafe',          3),
    ('opt_otype_cloud',      'outlet_type', 'cloud_kitchen', 'Cloud Kitchen', 4),
    ('opt_otype_bakery',     'outlet_type', 'bakery',        'Bakery',        5),
    ('opt_veh_bike',         'vehicle', 'bike',      'Bike',          1),
    ('opt_veh_scooter',      'vehicle', 'scooter',   'Scooter',       2),
    ('opt_veh_car',          'vehicle', 'car',       'Car',           3),
    ('opt_veh_bicycle',      'vehicle', 'bicycle',   'Bicycle',       4),
    ('opt_shift_morning',    'shift',   'morning',   'Morning',       1),
    ('opt_shift_afternoon',  'shift',   'afternoon', 'Afternoon',     2),
    ('opt_shift_evening',    'shift',   'evening',   'Evening',       3),
    ('opt_shift_night',      'shift',   'night',     'Night',         4),
    ('opt_menu_veg',         'menu_type', 'veg',        'Veg Only',       1),
    ('opt_menu_both',        'menu_type', 'veg_nonveg', 'Veg & Non-Veg',  2)
ON CONFLICT (set_kind, code) DO NOTHING;

-- Arabic labels for the seeded options (source='manual' so they are treated as
-- reviewed, not machine output, and never overwritten by auto-translation).
INSERT INTO public.content_translations (id, entity_type, entity_id, field, lang, value, source) VALUES
    ('ctr_opt_cuisine_indian_ar',  'option','opt_cuisine_indian','label','ar','هندي','manual'),
    ('ctr_opt_cuisine_chinese_ar', 'option','opt_cuisine_chinese','label','ar','صيني','manual'),
    ('ctr_opt_cuisine_italian_ar', 'option','opt_cuisine_italian','label','ar','إيطالي','manual'),
    ('ctr_opt_cuisine_arabic_ar',  'option','opt_cuisine_arabic','label','ar','عربي','manual'),
    ('ctr_opt_cuisine_mexican_ar', 'option','opt_cuisine_mexican','label','ar','مكسيكي','manual'),
    ('ctr_opt_cuisine_thai_ar',    'option','opt_cuisine_thai','label','ar','تايلندي','manual'),
    ('ctr_opt_cuisine_conti_ar',   'option','opt_cuisine_conti','label','ar','كونتيننتال','manual'),
    ('ctr_opt_otype_fine_ar',      'option','opt_otype_fine','label','ar','مطعم فاخر','manual'),
    ('ctr_opt_otype_qsr_ar',       'option','opt_otype_qsr','label','ar','خدمة سريعة','manual'),
    ('ctr_opt_otype_cafe_ar',      'option','opt_otype_cafe','label','ar','مقهى','manual'),
    ('ctr_opt_otype_cloud_ar',     'option','opt_otype_cloud','label','ar','مطبخ سحابي','manual'),
    ('ctr_opt_otype_bakery_ar',    'option','opt_otype_bakery','label','ar','مخبز','manual'),
    ('ctr_opt_veh_bike_ar',        'option','opt_veh_bike','label','ar','دراجة نارية','manual'),
    ('ctr_opt_veh_scooter_ar',     'option','opt_veh_scooter','label','ar','سكوتر','manual'),
    ('ctr_opt_veh_car_ar',         'option','opt_veh_car','label','ar','سيارة','manual'),
    ('ctr_opt_veh_bicycle_ar',     'option','opt_veh_bicycle','label','ar','دراجة هوائية','manual'),
    ('ctr_opt_shift_morning_ar',   'option','opt_shift_morning','label','ar','صباحي','manual'),
    ('ctr_opt_shift_afternoon_ar', 'option','opt_shift_afternoon','label','ar','بعد الظهر','manual'),
    ('ctr_opt_shift_evening_ar',   'option','opt_shift_evening','label','ar','مسائي','manual'),
    ('ctr_opt_shift_night_ar',     'option','opt_shift_night','label','ar','ليلي','manual'),
    ('ctr_opt_menu_veg_ar',        'option','opt_menu_veg','label','ar','نباتي فقط','manual'),
    ('ctr_opt_menu_both_ar',       'option','opt_menu_both','label','ar','نباتي وغير نباتي','manual')
ON CONFLICT (entity_type, entity_id, field, lang) DO NOTHING;
