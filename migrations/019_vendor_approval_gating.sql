-- Vendor registration currently reaches nowhere: submitting via
-- POST /vendor/register only ever wrote to vendor_registrations, a table
-- nothing else joins against. Neither grocery_profiles nor restaurants ever
-- got a row, the admin approvals queue (`approvals`) was never populated by
-- anything, and no code path anywhere checked an approval/KYC status before
-- letting a vendor create products or appear to customers.
--
-- restaurants previously had no active/KYC status columns at all. Default
-- is_active to true and kyc_status to 'verified' so this migration doesn't
-- retroactively hide every existing restaurant (seed data, and anything
-- created via the raw ops-only POST /restaurant) — the new self-service
-- registration path (repository/registration_repository.go
-- createOperationalProfile) explicitly inserts new rows with both set to
-- the unapproved state instead.
ALTER TABLE restaurants
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS kyc_status VARCHAR(20) NOT NULL DEFAULT 'verified';

CREATE INDEX IF NOT EXISTS idx_restaurants_created_by ON restaurants (created_by);
CREATE INDEX IF NOT EXISTS idx_restaurants_is_active ON restaurants (is_active);
CREATE INDEX IF NOT EXISTS idx_grocery_profiles_is_active ON grocery_profiles (is_active);
