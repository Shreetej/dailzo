-- Migration: 016_otp_hardening.sql
-- Description: Support failed-attempt lockout on OTP verification (a 6-digit
--              OTP with unlimited guesses is brute-forceable) and efficient
--              per-user send-rate lookups.
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/016_otp_hardening.sql
-- Date: 2026-07-25

BEGIN;

ALTER TABLE public.otps ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_otps_user_created ON public.otps (user_id, created_at);

COMMIT;
