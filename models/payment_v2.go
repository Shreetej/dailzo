package models

import "time"

// Payment v2 domain — see docs/PAYMENTS_REDESIGN.md.
//
// Capture model is immediate full "Sale" at order placement (no auth-hold, no
// buffer). The exact order total is charged; post-placement changes are
// refund-only. COD is a first-class method with its own lifecycle.

// Payment methods.
const (
	MethodCOD        = "cod"
	MethodMada       = "mada"
	MethodVisa       = "visa"
	MethodMastercard = "mastercard"
	MethodApplePay   = "applepay"
	MethodTabby      = "tabby"
	MethodTamara     = "tamara"
	MethodSTCPay     = "stcpay"
)

// Gateways.
const (
	GatewayHyperPay = "hyperpay"
	GatewayPayTabs  = "paytabs"
	GatewayCOD      = "cod"
	GatewayMock     = "mock"
)

// Payment statuses (single FSM across prepaid + COD).
const (
	StatusCreated           = "created"
	StatusProcessing        = "processing"
	StatusPending3DS        = "pending_3ds"
	StatusCaptured          = "captured"
	StatusFailed            = "failed"
	StatusExpired           = "expired"
	StatusVoided            = "voided"
	StatusPartiallyRefunded = "partially_refunded"
	StatusRefunded          = "refunded"
	// COD lifecycle
	StatusCODPending    = "cod_pending"
	StatusCODCollected  = "cod_collected"
	StatusCODReconciled = "cod_reconciled"
	StatusCancelled     = "cancelled"
)

// Ledger entry types / directions.
const (
	EntryCapture    = "capture"
	EntryRefund     = "refund"
	EntryCODCollect = "cod_collect"
	EntryFee        = "fee"
	EntryPayout     = "payout"
	EntryAdjustment = "adjustment"

	DirCredit = "credit"
	DirDebit  = "debit"
)

// PaymentV2 is the augmented payment record (maps to the extended payments table).
type PaymentV2 struct {
	ID                   string     `json:"id"`
	OrderID              string     `json:"order_id"`
	UserID               string     `json:"user_id"`
	Method               string     `json:"method"`
	Gateway              string     `json:"gateway"`
	GatewayTxnID         string     `json:"gateway_txn_id"`
	Currency             string     `json:"currency"`
	Amount               float64    `json:"amount"`
	AmountCaptured       float64    `json:"amount_captured"`
	AmountRefunded       float64    `json:"amount_refunded"`
	Status               string     `json:"status"`
	IdempotencyKey       string     `json:"idempotency_key"`
	ClientIdempotencyKey string     `json:"client_idempotency_key"`
	IsCOD                bool       `json:"is_cod"`
	CODCollectedBy       string     `json:"cod_collected_by,omitempty"`
	CODCollectedAt       *time.Time `json:"cod_collected_at,omitempty"`
	CODReconciledAt      *time.Time `json:"cod_reconciled_at,omitempty"`
	ThreeDSStatus        string     `json:"three_ds_status,omitempty"`
	CreatedOn            time.Time  `json:"created_on"`
	LastUpdatedOn        time.Time  `json:"last_updated_on"`
}

// PaymentEvent is an immutable audit/idempotency record for a payment transition.
type PaymentEvent struct {
	ID                string    `json:"id"`
	PaymentID         string    `json:"payment_id"`
	Type              string    `json:"type"`
	GatewayEventID    string    `json:"gateway_event_id,omitempty"`
	FromStatus        string    `json:"from_status,omitempty"`
	ToStatus          string    `json:"to_status,omitempty"`
	Amount            float64   `json:"amount,omitempty"`
	SignatureVerified bool      `json:"signature_verified"`
	OccurredOn        time.Time `json:"occurred_on"`
}

// LedgerEntry is an append-only money-movement record.
type LedgerEntry struct {
	ID               string    `json:"id"`
	PaymentID        string    `json:"payment_id"`
	OrderID          string    `json:"order_id"`
	EntryType        string    `json:"entry_type"`
	Direction        string    `json:"direction"`
	Amount           float64   `json:"amount"`
	Currency         string    `json:"currency"`
	GatewayTxnID     string    `json:"gateway_txn_id,omitempty"`
	ReconciliationID string    `json:"reconciliation_id,omitempty"`
	OccurredOn       time.Time `json:"occurred_on"`
}

// ReconciliationConfig is the singleton controlling reconciliation cadence.
type ReconciliationConfig struct {
	ID        string     `json:"id"`
	Cadence   string     `json:"cadence"` // daily|weekly|monthly
	RunAtHour int        `json:"run_at_hour"`
	Timezone  string     `json:"timezone"`
	Enabled   bool       `json:"enabled"`
	LastRunOn *time.Time `json:"last_run_on,omitempty"`
	NextRunOn *time.Time `json:"next_run_on,omitempty"`
	UpdatedOn time.Time  `json:"updated_on"`
}

// ReconciliationRun records one executed reconciliation window.
type ReconciliationRun struct {
	ID                  string     `json:"id"`
	Cadence             string     `json:"cadence"`
	PeriodStart         time.Time  `json:"period_start"`
	PeriodEnd           time.Time  `json:"period_end"`
	Status              string     `json:"status"`
	ExpectedTotal       float64    `json:"expected_total"`
	GatewaySettledTotal float64    `json:"gateway_settled_total"`
	CODCollectedTotal   float64    `json:"cod_collected_total"`
	DiscrepancyTotal    float64    `json:"discrepancy_total"`
	MatchedCount        int        `json:"matched_count"`
	UnmatchedCount      int        `json:"unmatched_count"`
	CreatedOn           time.Time  `json:"created_on"`
	CompletedOn         *time.Time `json:"completed_on,omitempty"`
}

// Invoice is a (ZATCA-scaffold) tax invoice. Phase-1 QR + VAT breakdown are
// generated now; Phase-2 cryptographic stamp + clearance are the tracked TODO
// (see docs/PAYMENTS_REDESIGN.md §7).
type Invoice struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	PaymentID     string    `json:"payment_id"`
	InvoiceNumber string    `json:"invoice_number"`
	ZatcaUUID     string    `json:"zatca_uuid"`
	ZatcaPIH      string    `json:"zatca_pih"` // previous-invoice-hash chain
	QRBase64      string    `json:"qr_base64"`
	Status        string    `json:"status"` // draft|cleared|reported|failed
	Subtotal      float64   `json:"subtotal"`
	VATAmount     float64   `json:"vat_amount"`
	Total         float64   `json:"total"`
	Currency      string    `json:"currency"`
	CreatedOn     time.Time `json:"created_on"`
}

// KSA standard VAT rate (15%).
const VATRate = 0.15

// allowedTransitions is the payment FSM. Illegal transitions are rejected so
// out-of-order webhooks (#8) and double-processing cannot corrupt state.
var allowedTransitions = map[string][]string{
	StatusCreated:    {StatusProcessing, StatusPending3DS, StatusCODPending, StatusExpired, StatusFailed, StatusCancelled},
	StatusProcessing: {StatusCaptured, StatusPending3DS, StatusFailed, StatusExpired, StatusVoided},
	StatusPending3DS: {StatusProcessing, StatusCaptured, StatusFailed, StatusExpired},
	StatusCaptured:   {StatusPartiallyRefunded, StatusRefunded, StatusVoided},
	// COD
	StatusCODPending:   {StatusCODCollected, StatusCancelled},
	StatusCODCollected: {StatusCODReconciled, StatusRefunded, StatusPartiallyRefunded},
	// terminal-ish
	StatusPartiallyRefunded: {StatusRefunded, StatusPartiallyRefunded},
}

// CanTransition reports whether status `from` may move to `to`.
func CanTransition(from, to string) bool {
	if from == to {
		return true // idempotent re-apply of the same state
	}
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// IsTerminal reports whether a payment status is final.
func IsTerminal(status string) bool {
	switch status {
	case StatusRefunded, StatusCODReconciled, StatusFailed, StatusExpired, StatusCancelled, StatusVoided:
		return true
	}
	return false
}
