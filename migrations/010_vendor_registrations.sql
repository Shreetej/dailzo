-- 010_vendor_registrations.sql
-- Unified vendor onboarding store for ALL partner types (restaurant, grocery,
-- delivery). The Partner app collects a multi-step profile per role; we persist
-- the whole submission as JSONB keyed by the authenticated user, so a
-- re-registration upserts and GET can tell whether onboarding is complete.

CREATE TABLE IF NOT EXISTS vendor_registrations (
    user_id         VARCHAR(255) PRIMARY KEY,
    user_type       VARCHAR(30)  NOT NULL,
    payload         JSONB        NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'submitted',
    created_on      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vendor_registrations_type ON vendor_registrations (user_type);
