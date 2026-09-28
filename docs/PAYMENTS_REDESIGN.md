# Dailzo Payments Redesign — Plan (KSA/GCC)

Status: **in progress** · Owner: backend-led, spans backend + customer app + vendor app
Last updated: 2026-07-10

**Progress:** P1 ✅ (migration 008 applied, models + FSM), P2 ✅ (gateway iface + mock + COD +
immediate-capture + refund + idempotency + ledger — curl-verified), P3 ✅ (reconciliation
cadence config + run + status-poll backstop — curl-verified, discrepancy=0 on mock),
P4 ✅ (webhook signature+idempotency+out-of-order-safe FSM + 3DS resume — curl-verified),
P7 ✅ (ZATCA scaffold: invoice auto-generated on capture/COD-collect with VAT-inclusive
split + Phase-1 TLV QR + PIH hash + fetch endpoint — curl-verified, QR decodes to 5 tags).
P5 ✅ core (customer app): CONFIGURABLE region system — Saudi Arabia→SAR, UAE→AED, Sudan→SDG
(lib/config/region_config.dart, single source of truth), Money.format helper, custom map-based
i18n (AppStrings + delegate, en+ar) with flutter_localizations RTL, settings provider (region+lang,
persisted), settings screen + menu entry, MaterialApp wired (locale/delegates/RTL), cart currency
converted, COD payment wired to POST /payments. intl bumped 0.17→0.20.2 for flutter_localizations.
0 analyze errors. Remaining (task #71): ₹→Money sweep on ~21 other screens + card/BNPL method
picker + on-device SAR/AED/SDG + ar-RTL verify.
P6 ✅ core (vendor/delivery app): backend GET /payments?order_id= (delivery lookup); delivery
"Complete Delivery" cash-collected → collectCod (datasource→repo→notifier) → POST cod/collect
(API-verified: list→resolve cod_pending→collect→cod_collected); admin Reconciliation screen
(cadence chips daily/weekly/monthly + Run now + runs list w/ discrepancy) at /admin/reconciliation;
same configurable region/currency/i18n foundation mirrored (region_config, Money, AppStrings en+ar,
settings provider, MaterialApp locale/RTL, vendor settings screen). 0 analyze errors.

ALL PLANNED PHASES P1–P7 DONE + tails: currency ₹→Money sweep done across BOTH apps (customer +
vendor, both analyze 0 errors — live prices via Money.auto ambient formatter; pure mock-data/demo
copy left as-is), payment-method picker done (GCC methods COD/Mada/Apple Pay/card/Tabby/Tamara,
backendMethodCode mapping, COD + prepaid both POST /payments with Idempotency-Key). EXTERNALLY
BLOCKED (not codeable here): real HyperPay/PayTabs adapter (needs creds — stubbed + interface-ready),
ZATCA Phase-2 cryptographic stamp + clearance (needs CSID — §7 TODO). Remaining verification:
on-device SAR/AED/SDG + ar-RTL walkthrough (needs app rebuilds).

## 0. Decisions (locked)

These are fixed inputs from product; implementation must not silently deviate.

1. **Capture model = immediate full "Sale" at order placement.** No Authorize→Capture hold, no
   auth buffer. The amount charged is exactly the order total shown to the customer at checkout.
2. **No overcharge, no undercharge.** The customer is only ever charged the exact displayed total.
   Any post-placement change (out-of-stock, substitution, cancellation) is handled **refund-only** —
   we never raise a second/higher charge automatically. A substitution that would cost *more* is
   either offered at the same price or requires an explicit new customer-approved charge (out of MVP scope).
3. **COD is first-class.** Separate lifecycle (no gateway), driver cash collection + reconciliation.
   Exact order total is collected; no change-making problem because we collect the displayed total.
4. **Gateway = HyperPay or PayTabs** (single gateway, native SDK, tokenization). Concrete adapter is
   pluggable behind an interface; a **mock adapter** is used until real credentials exist.
5. **Methods:** COD, Mada, Visa, Mastercard, Apple Pay (funded by Mada/Visa/MC), Tabby, Tamara.
   STC Pay = later. Apple Pay is a *presentation* layer, not a settlement network.
6. **Multi-currency + multi-language (Arabic/English, RTL) are in scope** and handled app-side +
   currency stored per payment on the backend.
7. Consequences of #1: auth-expiry handling is **moot** (no holds). BNPL/wallets **converge** with
   cards (all capture immediately; adjust via refund).

## 1. Scope

Backend (system of record) + customer app (checkout UX, i18n, currency) + vendor/delivery app
(COD cash collection + reconciliation view). Real gateway/SDK integration is stubbed behind an
interface until credentials are provisioned; everything else is fully implemented and testable
against a mock adapter.

## 2. Payment state machine

Single `status` enum on `payments`, every transition appended to `payment_events`.

**Prepaid (card / mada / applepay / visa / mastercard / tabby / tamara / stcpay):**
```
created ──▶ processing ──▶ captured ──▶ (partially_refunded ─▶) refunded
   │            │  ▲            │
   │            ▼  │            └─▶ (order-create failure) ─▶ auto-refund  [#11 orphan cleanup]
   │        pending_3ds                                             │
   │            │                                                   ▼
   └──▶ expired │                                              refunded
                └────────▶ failed
```
**COD:**
```
created ──▶ cod_pending ──▶ cod_collected ──▶ cod_reconciled
                 │
                 └─▶ cancelled            (cancel before delivery; refund N/A, nothing captured)
```
Terminal: `refunded`, `cod_reconciled`, `failed`, `expired`, `cancelled`, `voided`.
Illegal transitions are rejected by the FSM (guards against out-of-order webhooks — see #8).

## 3. Data model — migration `008_payments_v2.sql`

Extend existing `payments`/`refunds` (do **not** drop them). Add columns + new tables.

- **payments** (ALTER): `method text`, `gateway text` (hyperpay|paytabs|cod|mock),
  `gateway_txn_id text`, `currency char(3) default 'SAR'`, `amount_captured numeric(12,2) default 0`,
  `amount_refunded numeric(12,2) default 0`, `idempotency_key text UNIQUE`,
  `client_idempotency_key text`, `is_cod bool default false`,
  `cod_collected_by text` (driver user id), `cod_collected_at timestamptz`,
  `cod_reconciled_at timestamptz`, `metadata jsonb`, `three_ds_status text`.
  Widen `status` value set per the FSM.
- **payment_events** (NEW): `id, payment_id FK, type` (authorize|capture|refund|void|webhook|3ds|cod_collect|cod_reconcile),
  `gateway_event_id text UNIQUE` (nullable — dedupe key for webhook idempotency),
  `from_status, to_status, amount numeric(12,2), payload jsonb, signature_verified bool, occurred_on timestamptz`.
- **refunds** (ALTER if needed): ensure `status` (requested|processing|done|failed),
  `gateway_refund_id`, `amount`, `reason`.
- **payment_ledger** (NEW, append-only): `id, payment_id, order_id, entry_type`
  (capture|refund|cod_collect|fee|payout|adjustment), `direction` (credit|debit),
  `amount numeric(12,2), currency char(3), gateway_txn_id, occurred_on, reconciliation_id nullable`.
- **reconciliation_config** (NEW, singleton row): `cadence` (daily|weekly|monthly),
  `run_at_hour int, timezone text, enabled bool, last_run_on, next_run_on`.
- **reconciliation_runs** (NEW): `id, cadence, period_start, period_end, status`
  (running|completed|failed), `expected_total, gateway_settled_total, cod_collected_total,
  discrepancy_total, matched_count, unmatched_count, report jsonb, created_on, completed_on`.
- **invoices** (NEW, ZATCA — see §7): `id, order_id, payment_id, invoice_number, zatca_uuid,
  zatca_pih (previous invoice hash), qr_base64, status` (draft|cleared|reported|failed),
  `subtotal, vat_amount, total numeric(12,2), currency, xml_path, cleared_at, created_on`.

Amounts: `numeric(12,2)` for consistency with existing schema (minor-unit integers noted as a future
option; not adopted now to avoid churn with existing `orders.total_amount`/`payments.amount`).

## 4. Backend components

- `models/payment.go` — Payment, PaymentEvent, Refund, LedgerEntry, ReconRun, ReconConfig, Invoice; enums.
- `payment/fsm.go` — allowed-transition table + `CanTransition(from,to)`; single source of truth.
- `payment/gateway.go` — `Gateway` interface: `Sale(ctx, req) (txnID, status, raw, err)`,
  `Refund(ctx, txnID, amount) (...)`, `VerifyWebhook(sig, body) (event, err)`.
  Adapters: `gateway_mock.go` (deterministic success/fail by amount, used now),
  `gateway_hyperpay.go` / `gateway_paytabs.go` (stubbed, credential-gated, TODO).
- `repository/payment_v2_repository.go` — CRUD for payments/events/ledger/refunds/recon/invoices;
  every state change writes a `payment_events` row + (on capture/refund/cod) a `payment_ledger` row,
  in one tx.
- `controllers/payment_v2_controller.go` + routes (all under `/api/v1`):
  - `POST /payments` — create + immediate Sale (prepaid) or COD-pending. Requires
    `Idempotency-Key` header (client key, #9). Returns payment + (for 3DS) challenge redirect.
  - `POST /payments/webhook/:gateway` — signature-verified, idempotent by `gateway_event_id` (#8 dedupe).
  - `POST /payments/:id/cod/collect` — driver marks cash collected (delivery app).
  - `POST /payments/:id/cod/reconcile` — finance/ops marks reconciled.
  - `POST /payments/:id/refund` — full/partial refund.
  - `GET /payments/:id`, `GET /payments?order_id=`.
  - `GET/PUT /admin/reconciliation/config` — read/update cadence.
  - `GET /admin/reconciliation/runs` — list runs + discrepancies.
- **Jobs** (started in `main.go` as goroutines with ticker; guarded so only one runs):
  - **Status-poll backstop (#8):** every N minutes, find prepaid payments stuck in
    `processing`/`pending_3ds` past a threshold and poll the gateway for the true status; converge.
  - **Reconciliation (§6):** on the configured cadence, build a run comparing ledger vs gateway
    settlement report (mock: internal totals) + COD collected; record discrepancies.
- **Checkout integration:** `StorefrontRepository.Checkout` calls the payment service. For prepaid,
  order is created only after `captured`; if order-create fails post-capture → **auto-refund** (#11).
  For COD, order is created and payment set `cod_pending`.

## 5. COD lifecycle (detail)

1. Customer selects COD → `POST /payments` with method=cod → payment `cod_pending`, order created,
   ledger untouched (no cash yet).
2. Driver delivers, collects exact order total → delivery app `POST /payments/:id/cod/collect`
   → `cod_collected`, ledger `cod_collect` credit, `cod_collected_by/at` set.
3. Ops/finance settles driver float → `POST /payments/:id/cod/reconcile` → `cod_reconciled`,
   linked into the reconciliation run.
4. Cancellation before delivery → `cancelled` (no money moved).

## 6. Ledger + reconciliation (configurable cadence)

- Every capture/refund/cod-collect writes an immutable `payment_ledger` row (credit/debit).
- `reconciliation_config.cadence ∈ {daily, weekly, monthly}` (+ hour + timezone). Job computes the
  period window, sums ledger by type, compares to gateway settlement (mock: sum of captured−refunded)
  and COD collected, writes a `reconciliation_runs` row with `discrepancy_total` + per-txn unmatched list.
- Admin endpoints to change cadence and view runs. Default: daily @ 02:00 Asia/Riyadh.

## 7. ZATCA e-invoicing — TODO checklist (scaffold now, integrate later)

Table `invoices` created now; generation wired as a stub after successful payment webhook/capture.
Full ZATCA (Fatoora) compliance is a separate integration:

- [ ] Register on ZATCA Fatoora portal; obtain CSID (compliance + production cryptographic stamp identifier).
- [ ] Onboard device/EGS unit; generate CSR, get compliance CSID, then production CSID.
- [ ] Generate **UBL 2.1 XML** invoice per order (standard tax invoice B2C simplified).
- [ ] Compute invoice **hash (PIH chain)** — each invoice references previous invoice hash.
- [ ] **Cryptographic stamp** (ECDSA) with the production private key.
- [ ] Generate **TLV QR code** (seller name, VAT reg no, timestamp, invoice total, VAT total, hash, signature, public key).
- [ ] **Phase 2 clearance/reporting**: call ZATCA clearance API (standard) / reporting API (simplified) within required window.
- [ ] Store cleared XML + QR + UUID + status in `invoices`; expose PDF/A-3 with embedded XML.
- [ ] Arabic + English invoice fields (bilingual), SAR, 15% VAT lines.
- [ ] Handle rejection/retry + reporting SLA; alerting on failures.
- [ ] Legal: seller VAT registration number, CR number, address on every invoice.

## 8. Currency & i18n/l10n

- **Backend:** `payments.currency` + `invoices.currency` per record; order carries currency; amounts
  formatted client-side. Default SAR; support AED/others via config.
- **Customer + vendor apps (Flutter):** add `flutter_localizations` + `intl`; ARB files `app_en.arb`,
  `app_ar.arb`; `MaterialApp.supportedLocales=[en, ar]`, `localizationsDelegates`, locale resolution;
  **RTL** handled automatically via `Directionality` from locale (ar → rtl). Currency display via
  `NumberFormat.currency(locale, symbol/name)`. Replace hardcoded `₹`/`SAR` string concatenation with
  a `Money.format(amount, currency, locale)` helper. Audit every price widget.
- Payment-method labels, error messages, invoice copy all localized.

## 9. Security / PCI (#13)

- SDK tokenization only → PCI **SAQ A / A-EP** scope; **no PAN/CVV ever touches our servers or DB**.
- Store only tokens + `gateway_txn_id`. Verify webhook signatures. Rate-limit payment endpoints.
- 3DS/SCA (#12): `pending_3ds` state; SDK drives challenge; backend resumes on gateway callback/webhook.
- Fraud: per-user COD cap + velocity checks (later). Never log payloads with sensitive data.
- Apple Pay note (#13): prefer Mada-funded Apple Pay for lower fees / higher auth; track funding network.
- Tips & delivery fee (#13): modeled as explicit line items inside the captured total.

## 10. Phasing

- **P1 — Backend schema + FSM foundation** (migration 008, models, fsm). ← start here
- **P2 — Payment service + gateway interface + mock + COD** (repo, controller, routes, checkout wiring, orphan cleanup).
- **P3 — Ledger + reconciliation (configurable) + status-poll backstop jobs + admin endpoints.**
- **P4 — Webhook (signature + idempotency) + 3DS state handling.**
- **P5 — Customer app: currency + i18n/RTL + checkout method selection (COD/card/Apple Pay/BNPL) via `Money` helper.**
- **P6 — Vendor/delivery app: COD collect + reconciliation views + i18n.**
- **P7 — ZATCA invoice scaffold wired (stub) + TODO tracked; real integration later.**
- Real HyperPay/PayTabs adapter: credential-gated, after keys provisioned.

## 11. Verification

- `go build ./...` + `go vet` each backend phase; apply migration to dev DB; curl matrix for
  create→capture(mock)→webhook→refund, COD collect→reconcile, reconciliation run cadence, orphan auto-refund.
- `flutter analyze` ×2 apps; on-device: place COD + mock-card order in en + ar (RTL) with SAR formatting.
- Ledger balances reconcile to zero discrepancy on the mock adapter.
