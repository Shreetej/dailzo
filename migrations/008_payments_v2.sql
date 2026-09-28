-- Migration: 008_payments_v2.sql
-- Description: Payments redesign for KSA/GCC (see docs/PAYMENTS_REDESIGN.md).
--              Immediate full capture at placement, COD first-class, explicit
--              payment state machine, append-only ledger, configurable
--              reconciliation, ZATCA invoice scaffold. Extends the existing
--              payments/refunds tables (does NOT drop them).
-- Apply: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/008_payments_v2.sql
-- Date: 2026-07-10

BEGIN;

-- ── payments: augment for the v2 model ──────────────────────────────────────
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS method                 VARCHAR(30);       -- cod|mada|visa|mastercard|applepay|tabby|tamara|stcpay
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS gateway                VARCHAR(30);       -- hyperpay|paytabs|cod|mock
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS gateway_txn_id         VARCHAR(255);
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS currency               CHAR(3) DEFAULT 'SAR';
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS amount_captured        NUMERIC(12,2) DEFAULT 0;
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS amount_refunded        NUMERIC(12,2) DEFAULT 0;
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS idempotency_key        VARCHAR(255);      -- server-side dedupe of the create-payment call
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS client_idempotency_key VARCHAR(255);      -- client-generated per checkout attempt (#9)
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS is_cod                 BOOLEAN DEFAULT FALSE;
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS cod_collected_by       VARCHAR(255);      -- delivery person user id
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS cod_collected_at       TIMESTAMPTZ;
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS cod_reconciled_at      TIMESTAMPTZ;
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS three_ds_status        VARCHAR(30);       -- none|required|pending|passed|failed
ALTER TABLE public.payments ADD COLUMN IF NOT EXISTS metadata               JSONB;

-- Server-side idempotency: a given create-payment key maps to exactly one payment.
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_idempotency_key
    ON public.payments (idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payments_order       ON public.payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_status      ON public.payments (status);
CREATE INDEX IF NOT EXISTS idx_payments_gateway_txn ON public.payments (gateway_txn_id);

-- ── refunds: augment ─────────────────────────────────────────────────────────
ALTER TABLE public.refunds ADD COLUMN IF NOT EXISTS order_id          VARCHAR(255);
ALTER TABLE public.refunds ADD COLUMN IF NOT EXISTS gateway_refund_id VARCHAR(255);

-- ── payment_events: audit trail + webhook idempotency ───────────────────────
CREATE TABLE IF NOT EXISTS public.payment_events (
    id                 VARCHAR(255) PRIMARY KEY,
    payment_id         VARCHAR(255) NOT NULL REFERENCES public.payments(id),
    type               VARCHAR(30)  NOT NULL,      -- authorize|capture|refund|void|webhook|3ds|cod_collect|cod_reconcile
    gateway_event_id   VARCHAR(255),               -- dedupe key for provider webhooks (unique when present)
    from_status        VARCHAR(30),
    to_status          VARCHAR(30),
    amount             NUMERIC(12,2),
    payload            JSONB,
    signature_verified BOOLEAN DEFAULT FALSE,
    occurred_on        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_events_gateway_event
    ON public.payment_events (gateway_event_id) WHERE gateway_event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_events_payment ON public.payment_events (payment_id);

-- ── payment_ledger: append-only money movement ───────────────────────────────
CREATE TABLE IF NOT EXISTS public.payment_ledger (
    id                VARCHAR(255) PRIMARY KEY,
    payment_id        VARCHAR(255) NOT NULL REFERENCES public.payments(id),
    order_id          VARCHAR(255),
    entry_type        VARCHAR(30)  NOT NULL,       -- capture|refund|cod_collect|fee|payout|adjustment
    direction         VARCHAR(10)  NOT NULL,       -- credit|debit
    amount            NUMERIC(12,2) NOT NULL,
    currency          CHAR(3) NOT NULL DEFAULT 'SAR',
    gateway_txn_id    VARCHAR(255),
    reconciliation_id VARCHAR(255),
    occurred_on       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ledger_payment    ON public.payment_ledger (payment_id);
CREATE INDEX IF NOT EXISTS idx_ledger_occurred   ON public.payment_ledger (occurred_on);
CREATE INDEX IF NOT EXISTS idx_ledger_recon      ON public.payment_ledger (reconciliation_id);

-- ── reconciliation_config: singleton, controls cadence ──────────────────────
CREATE TABLE IF NOT EXISTS public.reconciliation_config (
    id           VARCHAR(255) PRIMARY KEY,
    cadence      VARCHAR(10) NOT NULL DEFAULT 'daily',   -- daily|weekly|monthly
    run_at_hour  INT NOT NULL DEFAULT 2,                 -- local hour to run
    timezone     VARCHAR(64) NOT NULL DEFAULT 'Asia/Riyadh',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_on  TIMESTAMPTZ,
    next_run_on  TIMESTAMPTZ,
    updated_on   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO public.reconciliation_config (id, cadence, run_at_hour, timezone, enabled)
    VALUES ('RECFG_SINGLETON', 'daily', 2, 'Asia/Riyadh', TRUE)
    ON CONFLICT (id) DO NOTHING;

-- ── reconciliation_runs: one row per executed reconciliation window ──────────
CREATE TABLE IF NOT EXISTS public.reconciliation_runs (
    id                   VARCHAR(255) PRIMARY KEY,
    cadence              VARCHAR(10) NOT NULL,
    period_start         TIMESTAMPTZ NOT NULL,
    period_end           TIMESTAMPTZ NOT NULL,
    status               VARCHAR(20) NOT NULL DEFAULT 'running',  -- running|completed|failed
    expected_total       NUMERIC(12,2) DEFAULT 0,
    gateway_settled_total NUMERIC(12,2) DEFAULT 0,
    cod_collected_total  NUMERIC(12,2) DEFAULT 0,
    discrepancy_total    NUMERIC(12,2) DEFAULT 0,
    matched_count        INT DEFAULT 0,
    unmatched_count      INT DEFAULT 0,
    report               JSONB,
    created_on           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_on         TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_recon_runs_period ON public.reconciliation_runs (period_start, period_end);

-- ── invoices: ZATCA e-invoicing scaffold (see docs §7 TODO) ─────────────────
CREATE TABLE IF NOT EXISTS public.invoices (
    id             VARCHAR(255) PRIMARY KEY,
    order_id       VARCHAR(255) REFERENCES public.orders(id),
    payment_id     VARCHAR(255) REFERENCES public.payments(id),
    invoice_number VARCHAR(64),
    zatca_uuid     VARCHAR(64),
    zatca_pih      TEXT,                    -- previous invoice hash (PIH chain)
    qr_base64      TEXT,
    status         VARCHAR(20) NOT NULL DEFAULT 'draft',  -- draft|cleared|reported|failed
    subtotal       NUMERIC(12,2) DEFAULT 0,
    vat_amount     NUMERIC(12,2) DEFAULT 0,
    total          NUMERIC(12,2) DEFAULT 0,
    currency       CHAR(3) NOT NULL DEFAULT 'SAR',
    xml_path       VARCHAR(500),
    cleared_at     TIMESTAMPTZ,
    created_on     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invoices_order ON public.invoices (order_id);

COMMIT;
