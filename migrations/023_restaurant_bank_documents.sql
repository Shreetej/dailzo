-- Restaurant registration previously collected owner/restaurant/location/legal
-- info only — bank payout details and KYC documents were captured by a form
-- (bank_details_screen.dart / document_info_screen.dart) that nothing in the
-- actual login/registration flow ever routed to, so they were never stored.
-- Mirrors grocery_profiles' existing bank_details/kyc_documents JSONB columns.

ALTER TABLE restaurants
    ADD COLUMN IF NOT EXISTS bank_details JSONB,
    ADD COLUMN IF NOT EXISTS kyc_documents JSONB;
