-- Migration: 003_offers_and_registration.sql
-- Description: Offers tables (matching repository/offer_repository.go queries,
--              which were previously broken — no offers tables existed) and
--              restaurant registration storage for the Partner app flow.
-- Date: 2026-07-04

BEGIN;

CREATE TABLE IF NOT EXISTS public.offers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    discount_percent NUMERIC(6,2) NOT NULL DEFAULT 0,
    max_discount_amount NUMERIC(10,2) NOT NULL DEFAULT 0,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS public.offer_conditions (
    id VARCHAR(50) PRIMARY KEY,
    offer_id VARCHAR(50) REFERENCES public.offers(id) ON DELETE CASCADE,
    condition_type VARCHAR(100) NOT NULL,
    value VARCHAR(255) NOT NULL,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS public.offer_applicable_entities (
    id VARCHAR(50) PRIMARY KEY,
    offer_id VARCHAR(50) REFERENCES public.offers(id) ON DELETE CASCADE,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    last_modified_by VARCHAR(255)
);

-- Partner app restaurant registration: the app owns the (large, evolving)
-- payload shape, so store it as JSONB and echo it back.
CREATE TABLE IF NOT EXISTS public.restaurant_registrations (
    id VARCHAR(100) PRIMARY KEY,
    payload JSONB NOT NULL,
    created_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMIT;
