-- Migration: 007_product_restaurant_link.sql
-- Description: Link products to a restaurant so restaurant menus are server-driven.
--             (This DB's food_products was auto-created without the ERD's `restaurant`
--              column; add an explicit restaurant_id used by catalog browse + seeding.)
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/007_product_restaurant_link.sql
-- Date: 2026-07-07

BEGIN;

ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS restaurant_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_food_products_restaurant ON public.food_products (restaurant_id);

-- Restaurants may not have all display columns in this DB; ensure the ones the
-- storefront needs exist.
ALTER TABLE public.restaurants ADD COLUMN IF NOT EXISTS image_path     VARCHAR(500);
ALTER TABLE public.restaurants ADD COLUMN IF NOT EXISTS cuisines       TEXT;
ALTER TABLE public.restaurants ADD COLUMN IF NOT EXISTS delivery_time  VARCHAR(30);
ALTER TABLE public.restaurants ADD COLUMN IF NOT EXISTS location       VARCHAR(255);

COMMIT;
