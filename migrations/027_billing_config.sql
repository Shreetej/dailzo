-- Server-owned billing policy, editable by admin.
--
-- Delivery fee, handling fee and VAT were constants inside the customer app
-- (handlingFee 10.0, deliveryPartnerFee 30.0, tax * .05) and were SENT to the
-- server, which accepted them verbatim at checkout. Two consequences: the
-- numbers could only change by shipping a new app build, and the amount
-- charged was whatever the device claimed.
--
-- VAT is the clearest example of the damage: the app hardcoded 5% while the
-- platform's default market is Saudi Arabia at 15%.
--
-- One row per market. The app never computes money from these; it asks the
-- server for a quote (GET /cart/quote), and checkout re-computes with the same
-- function, so the preview and the charge can never disagree.
CREATE TABLE IF NOT EXISTS public.billing_config (
    region_code             VARCHAR(2) PRIMARY KEY,
    country_name            VARCHAR(100) NOT NULL,
    currency_code           VARCHAR(3)  NOT NULL,
    -- Fraction, not percent: 0.1500 = 15%.
    vat_rate                NUMERIC(6,4) NOT NULL DEFAULT 0,
    handling_fee            NUMERIC(10,2) NOT NULL DEFAULT 0,
    delivery_fee            NUMERIC(10,2) NOT NULL DEFAULT 0,
    -- Order subtotal at or above which delivery is free. NULL = never free.
    free_delivery_threshold NUMERIC(10,2),
    -- Charged when the subtotal is under min_order_amount.
    small_order_fee         NUMERIC(10,2) NOT NULL DEFAULT 0,
    min_order_amount        NUMERIC(10,2) NOT NULL DEFAULT 0,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_on              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_on         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_modified_by        VARCHAR(255)
);

-- Seed the three markets the app's region picker already offers. Values match
-- what the client used to hardcode, EXCEPT the Saudi VAT rate, which is
-- corrected from the app's 5% to the statutory 15%.
INSERT INTO public.billing_config
    (region_code, country_name, currency_code, vat_rate, handling_fee,
     delivery_fee, free_delivery_threshold, small_order_fee, min_order_amount)
VALUES
    ('SA', 'Saudi Arabia',         'SAR', 0.1500, 10.00, 30.00, 150.00, 5.00, 25.00),
    ('AE', 'United Arab Emirates', 'AED', 0.0500, 10.00, 30.00, 150.00, 5.00, 25.00),
    ('SD', 'Sudan',                'SDG', 0.0000, 10.00, 30.00, NULL,   5.00, 25.00)
ON CONFLICT (region_code) DO NOTHING;
