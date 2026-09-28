-- Migration: 005_catalog_hierarchy.sql
-- Description: Normalized category hierarchy (food + grocery), product<->category link,
--              and a server-side persistent cart (per user, survives uninstall/re-login).
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/005_catalog_hierarchy.sql
-- Date: 2026-07-06

BEGIN;

-- ============================================
-- CATEGORY HIERARCHY (covers food menu sections AND grocery aisle/subcategory tree)
-- ============================================
CREATE TABLE IF NOT EXISTS public.categories (
    id            VARCHAR(50) PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    kind          VARCHAR(20) NOT NULL CHECK (kind IN ('food', 'grocery')),
    parent_id     VARCHAR(50) REFERENCES public.categories(id) ON DELETE CASCADE,
    image_url     VARCHAR(500),
    description   TEXT,
    sort_order    INT DEFAULT 0,
    restaurant_id VARCHAR(255),          -- nullable: food menu sections scoped to a restaurant; grocery categories are global
    outlet_id     VARCHAR(50),           -- nullable: optional outlet scoping
    is_active     BOOLEAN DEFAULT true,
    created_on    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by    VARCHAR(255),
    last_modified_by VARCHAR(255)
);

-- ============================================
-- LINK PRODUCTS TO CATEGORIES + kind flag (unifies food menu items and grocery SKUs)
-- ============================================
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS category_id VARCHAR(50);
ALTER TABLE public.food_products ADD COLUMN IF NOT EXISTS kind VARCHAR(20) DEFAULT 'food';
-- FK added guarded (categories must exist first; this migration creates it above)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'food_products_category_id_fkey'
    ) THEN
        ALTER TABLE public.food_products
            ADD CONSTRAINT food_products_category_id_fkey
            FOREIGN KEY (category_id) REFERENCES public.categories(id) ON DELETE SET NULL;
    END IF;
END$$;

-- ============================================
-- SERVER-SIDE PERSISTENT CART (per user; rows ARE the cart, grouped by cart_type)
-- ============================================
CREATE TABLE IF NOT EXISTS public.cart_items (
    id           VARCHAR(64) PRIMARY KEY,
    user_id      VARCHAR(255) NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    cart_type    VARCHAR(20) NOT NULL CHECK (cart_type IN ('food', 'grocery')),
    product_id   CHAR(36) NOT NULL REFERENCES public.food_products(id) ON DELETE CASCADE,
    variant_id   CHAR(36) REFERENCES public.product_variants(id) ON DELETE CASCADE,
    quantity     INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    added_on     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Unique per (user, type, product, variant). COALESCE handles the nullable variant_id
-- so ON CONFLICT upsert works (Postgres treats NULLs as distinct otherwise).
CREATE UNIQUE INDEX IF NOT EXISTS uq_cart_items_line
    ON public.cart_items (user_id, cart_type, product_id, COALESCE(variant_id, ''));

-- ============================================
-- INDEXES
-- ============================================
CREATE INDEX IF NOT EXISTS idx_categories_kind_parent   ON public.categories (kind, parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_restaurant     ON public.categories (restaurant_id);
CREATE INDEX IF NOT EXISTS idx_food_products_category_id ON public.food_products (category_id);
CREATE INDEX IF NOT EXISTS idx_food_products_kind        ON public.food_products (kind);
CREATE INDEX IF NOT EXISTS idx_cart_items_user           ON public.cart_items (user_id, cart_type);

COMMIT;
