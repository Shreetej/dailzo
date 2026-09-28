-- Delivery vendors can now "delegate ahead" a complaint they can't resolve
-- themselves to platform admin — a distinct status from 'investigating'
-- (which doesn't say who is investigating) so it's unambiguous in the
-- admin queue that a fleet specifically handed this off.
ALTER TABLE complaints DROP CONSTRAINT IF EXISTS complaints_status_check;
ALTER TABLE complaints ADD CONSTRAINT complaints_status_check
    CHECK (status IN ('open', 'investigating', 'resolved', 'closed', 'escalated'));
