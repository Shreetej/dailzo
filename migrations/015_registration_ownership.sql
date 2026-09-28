-- Migration: 015_registration_ownership.sql
-- Description: Tie restaurant_registrations rows to the authenticated user
--              that created them, so the registration endpoints (previously
--              unauthenticated, IDOR-able by guessing restaurant_id) can
--              require a JWT and verify ownership.
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/015_registration_ownership.sql
-- Date: 2026-07-25

BEGIN;

ALTER TABLE public.restaurant_registrations ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_restaurant_registrations_user_id
    ON public.restaurant_registrations (user_id);

COMMIT;
