-- 009_vendor_ops_status.sql
-- Vendor operational status + customer search-demand logging.
--   * vendor_store_status: role-agnostic open/closed switch, keyed by the
--     vendor user (covers restaurant AND grocery). Delivery partners use the
--     existing delivery_profiles.is_online flag instead.
--   * search_logs: every customer search query + how many catalog results it
--     matched, so vendors can see UNMET demand (queries that returned nothing).

CREATE TABLE IF NOT EXISTS vendor_store_status (
    user_id         VARCHAR(255) PRIMARY KEY,
    is_open         BOOLEAN      NOT NULL DEFAULT TRUE,
    updated_on      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS search_logs (
    id            VARCHAR(64) PRIMARY KEY,
    user_id       VARCHAR(255),
    query         TEXT        NOT NULL,
    city          VARCHAR(100),
    result_count  INTEGER     NOT NULL DEFAULT 0,
    kind          VARCHAR(20),          -- 'food' | 'grocery' | NULL (all)
    created_on    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_search_logs_query   ON search_logs (lower(query));
CREATE INDEX IF NOT EXISTS idx_search_logs_kind    ON search_logs (kind);
CREATE INDEX IF NOT EXISTS idx_search_logs_created ON search_logs (created_on);
