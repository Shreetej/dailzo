-- Product customisation: grouped options with price deltas.
--
-- The customer app already showed a "Portion / Spice level / Add-ons" sheet,
-- but those groups were const lists hardcoded in the Flutter screen with
-- invented prices, and nothing about the customer's choice could reach the
-- server: cart_items held only (product_id, variant_id, quantity), and
-- Checkout re-priced each line straight from food_products. The result was an
-- order that recorded the BASE price while the payment row recorded the price
-- the customer actually approved — a silent mismatch (SAR 227.50 vs 367.50 on
-- a single customised item) that also lost the selection the kitchen needed.
--
-- product_variants is deliberately left alone: it stores an ABSOLUTE price per
-- variant row and is already used by the vendor's variant flow and by
-- Checkout's `pv.price` branch. Options here are DELTAS layered on top of
-- whichever base price applies.

-- A group of choices on one product ("Size", "Add-ons").
CREATE TABLE IF NOT EXISTS public.product_option_groups (
    id              VARCHAR(64) PRIMARY KEY,
    product_id      CHAR(36) NOT NULL REFERENCES public.food_products(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    -- 'single' renders as radio buttons (pick exactly one), 'multi' as
    -- checkboxes (pick any number).
    selection       VARCHAR(10) NOT NULL DEFAULT 'single'
                    CHECK (selection IN ('single', 'multi')),
    is_required     BOOLEAN NOT NULL DEFAULT false,
    sort_order      INT NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_on      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- One choice within a group. price_delta is added to the line's base price;
-- 0 is normal (a spice level costs nothing).
CREATE TABLE IF NOT EXISTS public.product_options (
    id              VARCHAR(64) PRIMARY KEY,
    group_id        VARCHAR(64) NOT NULL REFERENCES public.product_option_groups(id) ON DELETE CASCADE,
    label           VARCHAR(100) NOT NULL,
    price_delta     NUMERIC(10,2) NOT NULL DEFAULT 0,
    sort_order      INT NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_on      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_option_groups_product ON public.product_option_groups(product_id);
CREATE INDEX IF NOT EXISTS idx_options_group ON public.product_options(group_id);

-- The customer's picks, carried with the cart line so Checkout can price them
-- server-side. A JSONB array of product_options.id, mirroring how orders.items
-- already stores its line payload.
ALTER TABLE public.cart_items
    ADD COLUMN IF NOT EXISTS selected_option_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Two lines for the same product with DIFFERENT options are different lines,
-- so the uniqueness that AddCartItem's ON CONFLICT relies on has to include
-- the selection. The old index keyed only (user_id, cart_type, product_id,
-- variant_id), which would have merged "Large + extra cheese" into "Regular".
DROP INDEX IF EXISTS public.uq_cart_items_line;
CREATE UNIQUE INDEX IF NOT EXISTS uq_cart_items_line
    ON public.cart_items (
        user_id,
        cart_type,
        product_id,
        COALESCE(variant_id, ''::bpchar),
        md5(selected_option_ids::text)
    );
