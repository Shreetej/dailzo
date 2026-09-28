-- Backs the vendor app's "Store Settings" section on the grocery profile
-- screen (GET /grocery/settings), which previously had no backend columns
-- to read from at all — the handler didn't exist and the route wasn't
-- registered, so the app 404'd and silently fell back to mock data.
ALTER TABLE grocery_profiles
    ADD COLUMN IF NOT EXISTS home_delivery BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS accepts_online_payment BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS store_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS store_size TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS service_area TEXT NOT NULL DEFAULT '';
