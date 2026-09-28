-- Migration: 017_user_fcm_token.sql
-- Description: Store each user's current FCM device token so server-initiated
--              push notifications (order status, etc.) have somewhere to send
--              to. Previously the client fetched a token and only
--              debugPrint'd it — nothing on the backend ever recorded it.
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/017_user_fcm_token.sql
-- Date: 2026-07-25

BEGIN;

ALTER TABLE public.users ADD COLUMN IF NOT EXISTS fcm_token VARCHAR(255);

COMMIT;
