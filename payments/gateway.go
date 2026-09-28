// Package payments contains the payment-gateway abstraction and provider
// adapters. The rest of the app talks to the Gateway interface only; concrete
// providers (HyperPay/PayTabs) are swapped in behind it once credentials exist.
// See docs/PAYMENTS_REDESIGN.md.
package payments

import "context"

// SaleRequest is an immediate full-capture ("Sale") request. There is no
// separate authorize step — the exact Amount is charged at order placement.
type SaleRequest struct {
	Amount    float64
	Currency  string
	Method    string // mada|visa|mastercard|applepay|tabby|tamara|stcpay
	Token     string // opaque payment token from the client SDK (never a PAN)
	OrderID   string
	UserID    string
	ReturnURL string // for 3DS challenge redirect
	Metadata  map[string]string
}

// SaleResult is the outcome of a Sale. Status is one of the model payment
// statuses: "captured", "pending_3ds", or "failed".
type SaleResult struct {
	TxnID           string
	Status          string
	ThreeDSRedirect string // populated when Status == pending_3ds
	Raw             map[string]any
}

// WebhookEvent is a normalized provider webhook.
type WebhookEvent struct {
	GatewayEventID string
	TxnID          string
	Status         string // captured|failed|refunded
	Amount         float64
	Raw            map[string]any
}

// Gateway is implemented by each payment provider (and the mock).
type Gateway interface {
	Name() string
	// Sale performs an immediate full capture of req.Amount.
	Sale(ctx context.Context, req SaleRequest) (SaleResult, error)
	// Refund returns funds (full or partial) for a captured transaction.
	Refund(ctx context.Context, txnID string, amount float64) (refundID string, err error)
	// Poll returns the current status of a transaction — used by the
	// status-poll backstop for payments stuck in a non-terminal state.
	Poll(ctx context.Context, txnID string) (status string, err error)
	// VerifyWebhook validates the provider signature and normalizes the event.
	VerifyWebhook(signature string, body []byte) (WebhookEvent, error)
}
