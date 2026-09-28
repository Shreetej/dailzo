-- Audit trail for admin "Notify vendors" clicks on unmet search demand.
-- No push/SMS delivery infra exists yet in this codebase; this records the
-- action as real/auditable rather than being a no-op button.
CREATE TABLE IF NOT EXISTS unmet_demand_notifications (
    id           VARCHAR(50) PRIMARY KEY,
    query        TEXT NOT NULL,
    notified_by  VARCHAR(255) NOT NULL,
    notified_on  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
