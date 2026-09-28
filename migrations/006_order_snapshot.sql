-- Migration: 006_order_snapshot.sql
-- Description: Store an immutable line-item snapshot + checkout metadata on orders,
--              so an order preserves exactly what was bought (name/price/qty) regardless
--              of later catalog changes, and both apps can render it without a brittle
--              product_variant FK join. Also lets checkout consume the server cart directly.
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/006_order_snapshot.sql
-- Date: 2026-07-06

BEGIN;

ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS cart_type        VARCHAR(20);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS items            JSONB;          -- [{product_id,name,price,quantity,variant,image_url}]
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS customer_name    VARCHAR(255);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS restaurant_name  VARCHAR(255);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS delivery_address TEXT;
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS payment_method   VARCHAR(50);
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS tip              NUMERIC(10,2) DEFAULT 0;
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS subtotal         NUMERIC(10,2) DEFAULT 0;
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS delivery_fee     NUMERIC(10,2) DEFAULT 0;
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS tax_amount       NUMERIC(10,2) DEFAULT 0;
ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS image_path       VARCHAR(500);

COMMIT;
