-- Migration: 002_marketing_and_vendor_ops.sql
-- Description: Discount campaigns, ad campaigns/packs, vendor outlets, and
--              order out-of-stock items for the Partner app.
-- Date: 2026-07-03

BEGIN;

CREATE TABLE IF NOT EXISTS public.discount_campaigns (
    id VARCHAR(50) PRIMARY KEY,
    restaurant_id VARCHAR(255),
    code VARCHAR(50) NOT NULL,
    type VARCHAR(30) NOT NULL,              -- percentage | flatOff | eliteExclusive
    segment VARCHAR(30) NOT NULL,           -- all | newCustomers | returningCustomers | dormantCustomers
    percent INT,
    flat_amount INT,
    min_order_value INT NOT NULL DEFAULT 0,
    capping_amount INT,
    start_date VARCHAR(20) NOT NULL,
    end_date VARCHAR(20),
    status VARCHAR(20) NOT NULL DEFAULT 'upcoming',  -- active | upcoming | inactive
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS public.ad_campaigns (
    id VARCHAR(50) PRIMARY KEY,
    restaurant_id VARCHAR(255),
    kind VARCHAR(20) NOT NULL,              -- quickPack | custom
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'upcoming',
    cpc NUMERIC(8,2) NOT NULL DEFAULT 0,
    target_customers VARCHAR(100) NOT NULL DEFAULT 'All customers',
    timeslot VARCHAR(100) NOT NULL DEFAULT 'All day',
    outlet_ids TEXT[] NOT NULL DEFAULT '{}',
    start_date VARCHAR(20) NOT NULL,
    duration_days INT NOT NULL DEFAULT 0,
    budget_line VARCHAR(200) NOT NULL DEFAULT '',
    clicks INT NOT NULL DEFAULT 0,
    spend INT NOT NULL DEFAULT 0,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS public.ad_packs (
    id VARCHAR(50) PRIMARY KEY,
    tier VARCHAR(20) NOT NULL,              -- recommended | standard | turbo
    description VARCHAR(200) NOT NULL,
    clicks INT NOT NULL,
    price INT NOT NULL,
    cpc NUMERIC(8,2) NOT NULL,
    duration_days INT NOT NULL
);

INSERT INTO public.ad_packs (id, tier, description, clicks, price, cpc, duration_days) VALUES
    ('PACK-REC-300',  'recommended', '300 clicks during all day',  300,  3450,  11.5, 30),
    ('PACK-STD-300',  'standard',    '300 clicks during all day',  300,  3450,  11.5, 30),
    ('PACK-STD-500',  'standard',    '500 clicks during all day',  500,  5750,  11.5, 30),
    ('PACK-TRB-800',  'turbo',       '800 clicks during all day',  800,  9200,  11.5, 30),
    ('PACK-TRB-1000', 'turbo',       '1000 clicks during all day', 1000, 11500, 11.5, 30)
ON CONFLICT (id) DO NOTHING;

-- Vendor outlets as created from the Partner app registration flow.
CREATE TABLE IF NOT EXISTS public.vendor_outlets (
    id VARCHAR(100) PRIMARY KEY,
    restaurant_id VARCHAR(255) NOT NULL,
    outlet_type VARCHAR(50),
    menu VARCHAR(20),
    cuisines TEXT[] NOT NULL DEFAULT '{}',
    cost_for_two INT NOT NULL DEFAULT 0,
    avg_delivery_time VARCHAR(30),
    address JSONB NOT NULL DEFAULT '{}'::jsonb,
    packaging_charge JSONB NOT NULL DEFAULT '{}'::jsonb,
    operating_hours JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

-- Items a vendor marked out of stock on an order (customer is informed to
-- choose alternatives).
CREATE TABLE IF NOT EXISTS public.order_out_of_stock_items (
    id SERIAL PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    product_id VARCHAR(255) NOT NULL,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    UNIQUE (order_id, product_id)
);

COMMIT;
