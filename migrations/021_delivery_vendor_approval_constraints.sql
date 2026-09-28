-- delivery_vendor (fleet/agency) is a new approvable entity type alongside
-- delivery/grocery/restaurant — relax the CHECK constraints that previously
-- hardcoded just those three.
ALTER TABLE approvals DROP CONSTRAINT IF EXISTS approvals_entity_type_check;
ALTER TABLE approvals ADD CONSTRAINT approvals_entity_type_check
    CHECK (entity_type IN ('delivery', 'grocery', 'restaurant', 'delivery_vendor'));

ALTER TABLE partner_suspensions DROP CONSTRAINT IF EXISTS partner_suspensions_partner_type_check;
ALTER TABLE partner_suspensions ADD CONSTRAINT partner_suspensions_partner_type_check
    CHECK (partner_type IN ('delivery', 'grocery', 'restaurant', 'delivery_vendor'));
