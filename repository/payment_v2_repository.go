package repository

import (
	"context"
	"dailzo/models"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// idCounter makes payment ids unique even for multiple inserts within the same
// request/second — GetIdToRecord uses second granularity and collides when a
// single request writes several events/ledger rows (created→processing→captured).
var idCounter uint64

func newPayID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s%d%04d", prefix, time.Now().UnixNano(), n%10000)
}

// PaymentV2Repository implements the payments-redesign persistence
// (see docs/PAYMENTS_REDESIGN.md): immediate-capture payments, COD lifecycle,
// append-only ledger, audit events, and reconciliation.
type PaymentV2Repository struct {
	db *pgxpool.Pool
}

func NewPaymentV2Repository(db *pgxpool.Pool) *PaymentV2Repository {
	return &PaymentV2Repository{db: db}
}

// ErrDuplicatePayment is returned when an idempotency key already has a payment.
var ErrDuplicatePayment = errors.New("payment already exists for idempotency key")

// GetByIdempotencyKey returns an existing payment for a client/server key, or nil.
func (r *PaymentV2Repository) GetByIdempotencyKey(ctx context.Context, key string) (*models.PaymentV2, error) {
	if key == "" {
		return nil, nil
	}
	rows, _ := r.db.Query(ctx, selectPaymentSQL+` WHERE idempotency_key=$1`, key)
	p, err := collectOnePayment(rows)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// Create inserts a new payment row in status `initial`.
func (r *PaymentV2Repository) Create(ctx context.Context, p models.PaymentV2) (models.PaymentV2, error) {
	p.ID = newPayID("PAY")
	now := time.Now()
	p.CreatedOn, p.LastUpdatedOn = now, now
	_, err := r.db.Exec(ctx, `
		INSERT INTO payments
			(id, order_id, user_id, method, gateway, gateway_txn_id, currency, amount,
			 amount_captured, amount_refunded, status, idempotency_key, client_idempotency_key,
			 is_cod, three_ds_status, payment_date, created_on, last_updated_on)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$16,$16)`,
		p.ID, nullIfEmpty(p.OrderID), nullIfEmpty(p.UserID), p.Method, p.Gateway,
		nullIfEmpty(p.GatewayTxnID), p.Currency, p.Amount, p.AmountCaptured, p.AmountRefunded,
		p.Status, nullIfEmpty(p.IdempotencyKey), nullIfEmpty(p.ClientIdempotencyKey),
		p.IsCOD, nullIfEmpty(p.ThreeDSStatus), now)
	if err != nil {
		return models.PaymentV2{}, fmt.Errorf("create payment: %w", err)
	}
	return p, nil
}

// Transition moves a payment to a new status (FSM-guarded), optionally updating
// the gateway txn id and captured/refunded amounts, and appends a payment_event
// plus (when amount != 0 and the entry type is set) a ledger row — all in one tx.
type TransitionOpts struct {
	EventType      string  // models.Entry* / event type
	GatewayTxnID   string  // set on payment if non-empty
	CaptureAmount  float64 // added to amount_captured + ledger credit
	RefundAmount   float64 // added to amount_refunded + ledger debit
	LedgerEntry    string  // models.Entry* to write; "" = no ledger row
	LedgerDir      string  // models.DirCredit/DirDebit
	GatewayEventID string  // webhook dedupe (unique); "" allowed
	Signature      bool
}

func (r *PaymentV2Repository) Transition(ctx context.Context, paymentID, toStatus string, o TransitionOpts) (models.PaymentV2, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PaymentV2{}, err
	}
	defer tx.Rollback(ctx)

	// Lock + read current.
	rows, _ := tx.Query(ctx, selectPaymentSQL+` WHERE id=$1 FOR UPDATE`, paymentID)
	cur, err := collectOnePayment(rows)
	if err != nil {
		return models.PaymentV2{}, fmt.Errorf("load payment: %w", err)
	}
	if !models.CanTransition(cur.Status, toStatus) {
		return models.PaymentV2{}, fmt.Errorf("illegal transition %s -> %s", cur.Status, toStatus)
	}

	now := time.Now()
	newCaptured := cur.AmountCaptured + o.CaptureAmount
	newRefunded := cur.AmountRefunded + o.RefundAmount
	gtxn := cur.GatewayTxnID
	if o.GatewayTxnID != "" {
		gtxn = o.GatewayTxnID
	}

	if _, err := tx.Exec(ctx, `
		UPDATE payments SET status=$1, gateway_txn_id=$2, amount_captured=$3,
			amount_refunded=$4, last_updated_on=$5 WHERE id=$6`,
		toStatus, nullIfEmpty(gtxn), newCaptured, newRefunded, now, paymentID); err != nil {
		return models.PaymentV2{}, fmt.Errorf("update payment: %w", err)
	}

	// Audit event (gateway_event_id is UNIQUE when present → webhook idempotency).
	if _, err := tx.Exec(ctx, `
		INSERT INTO payment_events
			(id, payment_id, type, gateway_event_id, from_status, to_status, amount, signature_verified, occurred_on)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		newPayID("PEVT"), paymentID, o.EventType, nullIfEmpty(o.GatewayEventID),
		cur.Status, toStatus, o.CaptureAmount+o.RefundAmount, o.Signature, now); err != nil {
		return models.PaymentV2{}, fmt.Errorf("insert event: %w", err)
	}

	// Ledger row (append-only money movement).
	if o.LedgerEntry != "" {
		amt := o.CaptureAmount
		if o.RefundAmount > 0 {
			amt = o.RefundAmount
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO payment_ledger
				(id, payment_id, order_id, entry_type, direction, amount, currency, gateway_txn_id, occurred_on)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			newPayID("PLDG"), paymentID, nullIfEmpty(cur.OrderID), o.LedgerEntry, o.LedgerDir,
			amt, cur.Currency, nullIfEmpty(gtxn), now); err != nil {
			return models.PaymentV2{}, fmt.Errorf("insert ledger: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PaymentV2{}, err
	}
	cur.Status = toStatus
	cur.GatewayTxnID = gtxn
	cur.AmountCaptured = newCaptured
	cur.AmountRefunded = newRefunded
	cur.LastUpdatedOn = now
	return *cur, nil
}

// SetCODCollected records driver cash collection (delivery app).
func (r *PaymentV2Repository) SetCODCollected(ctx context.Context, paymentID, driverID string) (models.PaymentV2, error) {
	p, err := r.Transition(ctx, paymentID, models.StatusCODCollected, TransitionOpts{
		EventType: models.EntryCODCollect, LedgerEntry: models.EntryCODCollect, LedgerDir: models.DirCredit,
	})
	if err != nil {
		return p, err
	}
	// CaptureAmount wasn't set (COD amount is the full order amount); backfill
	// ledger + captured from the payment amount, and stamp collector.
	now := time.Now()
	_, _ = r.db.Exec(ctx, `UPDATE payments SET cod_collected_by=$1, cod_collected_at=$2,
		amount_captured=amount WHERE id=$3`, driverID, now, paymentID)
	_, _ = r.db.Exec(ctx, `UPDATE payment_ledger SET amount=(SELECT amount FROM payments WHERE id=$1)
		WHERE payment_id=$1 AND entry_type=$2 AND amount=0`, paymentID, models.EntryCODCollect)
	if fresh, ferr := r.GetByID(ctx, paymentID); ferr == nil && fresh != nil {
		return *fresh, nil
	}
	p.CODCollectedBy = driverID
	p.CODCollectedAt = &now
	return p, nil
}

// SetCODReconciled marks a collected COD payment reconciled.
func (r *PaymentV2Repository) SetCODReconciled(ctx context.Context, paymentID string) (models.PaymentV2, error) {
	p, err := r.Transition(ctx, paymentID, models.StatusCODReconciled, TransitionOpts{EventType: models.EntryAdjustment})
	if err != nil {
		return p, err
	}
	now := time.Now()
	_, _ = r.db.Exec(ctx, `UPDATE payments SET cod_reconciled_at=$1 WHERE id=$2`, now, paymentID)
	p.CODReconciledAt = &now
	return p, nil
}

// GetByID returns a payment.
func (r *PaymentV2Repository) GetByID(ctx context.Context, id string) (*models.PaymentV2, error) {
	rows, _ := r.db.Query(ctx, selectPaymentSQL+` WHERE id=$1`, id)
	return collectOnePayment(rows)
}

// GetByGatewayTxn resolves a payment from a provider transaction id (webhook).
func (r *PaymentV2Repository) GetByGatewayTxn(ctx context.Context, txnID string) (*models.PaymentV2, error) {
	rows, _ := r.db.Query(ctx, selectPaymentSQL+` WHERE gateway_txn_id=$1`, txnID)
	return collectOnePayment(rows)
}

// ListByOrder returns payments for an order (newest first) — used by the
// delivery app to resolve the COD payment to collect.
func (r *PaymentV2Repository) ListByOrder(ctx context.Context, orderID string) ([]models.PaymentV2, error) {
	rows, err := r.db.Query(ctx, selectPaymentSQL+` WHERE order_id=$1 ORDER BY created_on DESC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PaymentV2
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, nil
}

// WebhookEventSeen reports whether a gateway_event_id was already processed.
func (r *PaymentV2Repository) WebhookEventSeen(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil
	}
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM payment_events WHERE gateway_event_id=$1`, eventID).Scan(&n)
	return n > 0, err
}

// ListStuck returns prepaid payments left in a non-terminal processing state
// older than `olderThan` — the input to the status-poll backstop (#8).
func (r *PaymentV2Repository) ListStuck(ctx context.Context, olderThan time.Duration) ([]models.PaymentV2, error) {
	cutoff := time.Now().Add(-olderThan)
	rows, err := r.db.Query(ctx, selectPaymentSQL+
		` WHERE status IN ($1,$2) AND is_cod=false AND last_updated_on < $3`,
		models.StatusProcessing, models.StatusPending3DS, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PaymentV2
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, nil
}

// ── Reconciliation ───────────────────────────────────────────────────────────

// GetReconConfig returns the singleton reconciliation config.
func (r *PaymentV2Repository) GetReconConfig(ctx context.Context) (models.ReconciliationConfig, error) {
	var c models.ReconciliationConfig
	err := r.db.QueryRow(ctx, `SELECT id, cadence, run_at_hour, timezone, enabled,
		last_run_on, next_run_on, updated_on FROM reconciliation_config WHERE id='RECFG_SINGLETON'`).
		Scan(&c.ID, &c.Cadence, &c.RunAtHour, &c.Timezone, &c.Enabled, &c.LastRunOn, &c.NextRunOn, &c.UpdatedOn)
	return c, err
}

// UpdateReconConfig sets cadence/hour/timezone/enabled.
func (r *PaymentV2Repository) UpdateReconConfig(ctx context.Context, cadence string, hour int, tz string, enabled bool) (models.ReconciliationConfig, error) {
	switch cadence {
	case "daily", "weekly", "monthly":
	default:
		return models.ReconciliationConfig{}, fmt.Errorf("invalid cadence %q (daily|weekly|monthly)", cadence)
	}
	_, err := r.db.Exec(ctx, `UPDATE reconciliation_config
		SET cadence=$1, run_at_hour=$2, timezone=$3, enabled=$4, updated_on=NOW()
		WHERE id='RECFG_SINGLETON'`, cadence, hour, tz, enabled)
	if err != nil {
		return models.ReconciliationConfig{}, err
	}
	return r.GetReconConfig(ctx)
}

// RunReconciliation computes a reconciliation window [start,end): aggregates the
// ledger (captures−refunds) as the "settled" figure and COD credits separately,
// compares to the expected (sum of captured payment amounts in the window), and
// records a reconciliation_runs row. On the mock gateway these match exactly, so
// discrepancy should be 0; a real gateway compares against its settlement report.
func (r *PaymentV2Repository) RunReconciliation(ctx context.Context, cadence string, start, end time.Time) (models.ReconciliationRun, error) {
	run := models.ReconciliationRun{
		ID: newPayID("RECRUN"), Cadence: cadence, PeriodStart: start, PeriodEnd: end,
		Status: "running", CreatedOn: time.Now(),
	}
	if _, err := r.db.Exec(ctx, `INSERT INTO reconciliation_runs
		(id, cadence, period_start, period_end, status, created_on)
		VALUES ($1,$2,$3,$4,'running',$5)`, run.ID, cadence, start, end, run.CreatedOn); err != nil {
		return run, err
	}

	// Ledger settled = card/wallet captures − refunds within window.
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(CASE WHEN direction='credit' THEN amount ELSE -amount END),0)
		FROM payment_ledger WHERE entry_type IN ('capture','refund') AND occurred_on >= $1 AND occurred_on < $2`,
		start, end).Scan(&run.GatewaySettledTotal)
	// COD collected within window.
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM payment_ledger
		WHERE entry_type='cod_collect' AND occurred_on >= $1 AND occurred_on < $2`, start, end).
		Scan(&run.CODCollectedTotal)
	// Expected = captured − refunded on the payments themselves (prepaid) in window.
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(amount_captured-amount_refunded),0)
		FROM payments WHERE is_cod=false AND last_updated_on >= $1 AND last_updated_on < $2
		AND status IN ('captured','partially_refunded','refunded')`, start, end).Scan(&run.ExpectedTotal)

	var matched, unmatched int
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM payments
		WHERE last_updated_on >= $1 AND last_updated_on < $2 AND status IN ('captured','cod_reconciled','partially_refunded','refunded')`,
		start, end).Scan(&matched)
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM payments
		WHERE last_updated_on >= $1 AND last_updated_on < $2 AND status IN ('processing','pending_3ds','cod_pending','cod_collected')`,
		start, end).Scan(&unmatched)

	run.MatchedCount = matched
	run.UnmatchedCount = unmatched
	run.DiscrepancyTotal = run.ExpectedTotal - run.GatewaySettledTotal
	run.Status = "completed"
	now := time.Now()
	run.CompletedOn = &now
	_, err := r.db.Exec(ctx, `UPDATE reconciliation_runs SET status='completed',
		expected_total=$1, gateway_settled_total=$2, cod_collected_total=$3, discrepancy_total=$4,
		matched_count=$5, unmatched_count=$6, completed_on=$7 WHERE id=$8`,
		run.ExpectedTotal, run.GatewaySettledTotal, run.CODCollectedTotal, run.DiscrepancyTotal,
		matched, unmatched, now, run.ID)
	return run, err
}

// TouchReconConfigRun stamps last/next run times after a scheduled run.
func (r *PaymentV2Repository) TouchReconConfigRun(ctx context.Context, last, next time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE reconciliation_config SET last_run_on=$1, next_run_on=$2, updated_on=NOW()
		WHERE id='RECFG_SINGLETON'`, last, next)
	return err
}

// ListReconRuns returns recent reconciliation runs.
func (r *PaymentV2Repository) ListReconRuns(ctx context.Context, limit int) ([]models.ReconciliationRun, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `SELECT id, cadence, period_start, period_end, status,
		expected_total, gateway_settled_total, cod_collected_total, discrepancy_total,
		matched_count, unmatched_count, created_on, completed_on
		FROM reconciliation_runs ORDER BY created_on DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ReconciliationRun
	for rows.Next() {
		var x models.ReconciliationRun
		if err := rows.Scan(&x.ID, &x.Cadence, &x.PeriodStart, &x.PeriodEnd, &x.Status,
			&x.ExpectedTotal, &x.GatewaySettledTotal, &x.CODCollectedTotal, &x.DiscrepancyTotal,
			&x.MatchedCount, &x.UnmatchedCount, &x.CreatedOn, &x.CompletedOn); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

// ── Invoices (ZATCA scaffold) ────────────────────────────────────────────────

// LastInvoiceHash returns the most recent invoice's hash for the PIH chain
// (empty if none yet). The scaffold stores each invoice's own hash in zatca_pih
// alongside the chain by convention documented in docs §7.
func (r *PaymentV2Repository) LastInvoiceHash(ctx context.Context) (string, error) {
	var h string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(zatca_pih,'') FROM invoices ORDER BY created_on DESC LIMIT 1`).Scan(&h)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return h, err
}

// CreateInvoice inserts a draft invoice (idempotent per payment: at most one).
func (r *PaymentV2Repository) CreateInvoice(ctx context.Context, inv models.Invoice) (models.Invoice, error) {
	// Skip if an invoice already exists for this payment.
	var n int
	_ = r.db.QueryRow(ctx, `SELECT count(*) FROM invoices WHERE payment_id=$1`, inv.PaymentID).Scan(&n)
	if n > 0 {
		existing, err := r.GetInvoiceByPayment(ctx, inv.PaymentID)
		if err == nil && existing != nil {
			return *existing, nil
		}
	}
	inv.ID = newPayID("INV")
	inv.CreatedOn = time.Now()
	_, err := r.db.Exec(ctx, `
		INSERT INTO invoices (id, order_id, payment_id, invoice_number, zatca_uuid, zatca_pih,
			qr_base64, status, subtotal, vat_amount, total, currency, created_on)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		inv.ID, nullIfEmpty(inv.OrderID), nullIfEmpty(inv.PaymentID), inv.InvoiceNumber,
		inv.ZatcaUUID, inv.ZatcaPIH, inv.QRBase64, inv.Status, inv.Subtotal, inv.VATAmount,
		inv.Total, inv.Currency, inv.CreatedOn)
	if err != nil {
		return models.Invoice{}, fmt.Errorf("create invoice: %w", err)
	}
	return inv, nil
}

func scanInvoice(row pgx.Row) (*models.Invoice, error) {
	var i models.Invoice
	err := row.Scan(&i.ID, &i.OrderID, &i.PaymentID, &i.InvoiceNumber, &i.ZatcaUUID,
		&i.ZatcaPIH, &i.QRBase64, &i.Status, &i.Subtotal, &i.VATAmount, &i.Total,
		&i.Currency, &i.CreatedOn)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

const selectInvoiceSQL = `SELECT id, COALESCE(order_id,''), COALESCE(payment_id,''),
	COALESCE(invoice_number,''), COALESCE(zatca_uuid,''), COALESCE(zatca_pih,''),
	COALESCE(qr_base64,''), status, COALESCE(subtotal,0), COALESCE(vat_amount,0),
	COALESCE(total,0), COALESCE(currency,'SAR'), created_on FROM invoices`

// GetInvoiceByPayment returns the invoice for a payment, or nil.
func (r *PaymentV2Repository) GetInvoiceByPayment(ctx context.Context, paymentID string) (*models.Invoice, error) {
	inv, err := scanInvoice(r.db.QueryRow(ctx, selectInvoiceSQL+` WHERE payment_id=$1 LIMIT 1`, paymentID))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return inv, err
}

// GetInvoicesByOrder returns invoices for an order.
func (r *PaymentV2Repository) GetInvoicesByOrder(ctx context.Context, orderID string) ([]models.Invoice, error) {
	rows, err := r.db.Query(ctx, selectInvoiceSQL+` WHERE order_id=$1 ORDER BY created_on DESC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, nil
}

const selectPaymentSQL = `
	SELECT id, COALESCE(order_id,''), COALESCE(user_id,''), COALESCE(method,''),
		COALESCE(gateway,''), COALESCE(gateway_txn_id,''), COALESCE(currency,'SAR'),
		amount, COALESCE(amount_captured,0), COALESCE(amount_refunded,0), status,
		COALESCE(idempotency_key,''), COALESCE(client_idempotency_key,''), COALESCE(is_cod,false),
		COALESCE(cod_collected_by,''), cod_collected_at, cod_reconciled_at,
		COALESCE(three_ds_status,''), created_on, last_updated_on
	FROM payments`

func scanPayment(rows pgx.Rows) (*models.PaymentV2, error) {
	var p models.PaymentV2
	err := rows.Scan(&p.ID, &p.OrderID, &p.UserID, &p.Method, &p.Gateway, &p.GatewayTxnID,
		&p.Currency, &p.Amount, &p.AmountCaptured, &p.AmountRefunded, &p.Status,
		&p.IdempotencyKey, &p.ClientIdempotencyKey, &p.IsCOD, &p.CODCollectedBy,
		&p.CODCollectedAt, &p.CODReconciledAt, &p.ThreeDSStatus, &p.CreatedOn, &p.LastUpdatedOn)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func collectOnePayment(rows pgx.Rows) (*models.PaymentV2, error) {
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, pgx.ErrNoRows
	}
	return scanPayment(rows)
}
