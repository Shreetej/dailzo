-- Migration: 014_category_department.sql
-- Description: Real department/section grouping for grocery categories (e.g.
--              "Grocery & Kitchen", "Snacks & Drinks", "Beauty & Personal Care"),
--              replacing the customer app's previous array-index-slicing hack
--              which produced mislabeled groupings.
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/014_category_department.sql
-- Date: 2026-07-24

BEGIN;

ALTER TABLE public.categories ADD COLUMN IF NOT EXISTS department VARCHAR(50);

CREATE INDEX IF NOT EXISTS idx_categories_department ON public.categories (department);

COMMIT;
