-- Adds admin-settable commission_rate to the partner types that didn't have
-- one yet (grocery_profiles already has it, default 15.00 — match that).
ALTER TABLE restaurants ADD COLUMN IF NOT EXISTS commission_rate NUMERIC(5,2) NOT NULL DEFAULT 15.00;
ALTER TABLE delivery_vendor_profiles ADD COLUMN IF NOT EXISTS commission_rate NUMERIC(5,2) NOT NULL DEFAULT 15.00;
