-- Membership figures as admin-editable config.
--
-- The membership screen carried "Buy at Rs. 499 for 3 months" and "On orders
-- above Rs. 199" in its copy: the wrong currency for every market the app
-- serves, and changeable only by shipping a build. They join the referral
-- amounts in option_items so an admin can set them per deployment.
--
-- The values below preserve what the copy used to say; they are starting
-- numbers for the dev fixture, not a pricing decision.
INSERT INTO option_items (id, set_kind, code, label, sort_order, is_active) VALUES
    ('OPTI_membership_price_3m',   'membership', 'price_3months',    '499', 1, true),
    ('OPTI_membership_free_min',   'membership', 'free_delivery_min','199', 2, true)
ON CONFLICT (set_kind, code) DO UPDATE
    SET label = EXCLUDED.label,
        sort_order = EXCLUDED.sort_order,
        is_active = EXCLUDED.is_active;

SELECT set_kind, code, label FROM option_items
WHERE set_kind IN ('referral','membership')
ORDER BY set_kind, sort_order;
