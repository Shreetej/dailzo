-- Delivery vendor (fleet/agency) profiles. Mirrors the grocery/restaurant
-- self-service approval pattern (registration_repository.go
-- createOperationalProfile) but adds a second, delegated approval tier:
-- the platform approves the FLEET entity; the fleet then approves its own
-- individual riders via their own invite code, rather than every rider
-- going straight to the platform's approval queue.
CREATE TABLE IF NOT EXISTS delivery_vendor_profiles (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_name VARCHAR(255) NOT NULL,
    owner_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(20) NOT NULL,
    address TEXT,
    city VARCHAR(100),
    pincode VARCHAR(10),
    gst_number VARCHAR(50),
    invite_code VARCHAR(20) UNIQUE NOT NULL,
    kyc_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    is_active BOOLEAN NOT NULL DEFAULT false,
    rating DECIMAL(3,2) DEFAULT 0.00,
    created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_delivery_vendor_profiles_user_id ON delivery_vendor_profiles (user_id);
CREATE INDEX IF NOT EXISTS idx_delivery_vendor_profiles_invite_code ON delivery_vendor_profiles (invite_code);

-- Individual riders now link to the fleet they registered under (nullable —
-- a rider created before this feature existed, or via any future
-- direct-to-platform path, simply has no fleet), and gain a real approval
-- gate (is_active) distinct from is_online (which only means "currently on
-- shift", not "account approved"). Existing rows default to active so no
-- currently-working rider is retroactively locked out; the new invite-code
-- registration path (registration_repository.go) explicitly inserts new
-- riders with is_active = false until their fleet approves them.
ALTER TABLE delivery_profiles
    ADD COLUMN IF NOT EXISTS vendor_id VARCHAR(50) REFERENCES delivery_vendor_profiles(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_delivery_profiles_vendor_id ON delivery_profiles (vendor_id);
