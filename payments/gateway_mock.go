package payments

import (
	"context"
	"dailzo/models"
	"encoding/json"
	"errors"
	"fmt"
)

// MockGateway is a deterministic in-process gateway used until real HyperPay /
// PayTabs credentials are provisioned. Behaviour is driven purely by the
// request so tests are repeatable (no Math.random / clock):
//
//   - amount with fractional part .05  -> pending_3ds (simulates SCA challenge)
//   - amount with fractional part .09  -> failed
//   - anything else                    -> captured immediately
//
// Refund always succeeds. Poll reports captured. VerifyWebhook treats a
// non-empty signature as valid and expects a JSON body it can normalize.
type MockGateway struct{}

func NewMockGateway() *MockGateway { return &MockGateway{} }

func (m *MockGateway) Name() string { return models.GatewayMock }

func cents(amount float64) int { return int((amount-float64(int(amount)))*100 + 0.5) }

func (m *MockGateway) Sale(_ context.Context, req SaleRequest) (SaleResult, error) {
	txn := "MOCKTXN_" + req.OrderID
	switch cents(req.Amount) {
	case 5:
		return SaleResult{TxnID: txn, Status: models.StatusPending3DS,
			ThreeDSRedirect: "https://mock-3ds.local/challenge?txn=" + txn,
			Raw:             map[string]any{"mock": true}}, nil
	case 9:
		return SaleResult{TxnID: txn, Status: models.StatusFailed,
			Raw: map[string]any{"mock": true, "reason": "insufficient_funds"}}, nil
	default:
		return SaleResult{TxnID: txn, Status: models.StatusCaptured,
			Raw: map[string]any{"mock": true}}, nil
	}
}

func (m *MockGateway) Refund(_ context.Context, txnID string, amount float64) (string, error) {
	if txnID == "" {
		return "", errors.New("mock refund: empty txn id")
	}
	return "MOCKRFND_" + txnID + fmt.Sprintf("_%.2f", amount), nil
}

func (m *MockGateway) Poll(_ context.Context, txnID string) (string, error) {
	if txnID == "" {
		return "", errors.New("mock poll: empty txn id")
	}
	return models.StatusCaptured, nil
}

func (m *MockGateway) VerifyWebhook(signature string, body []byte) (WebhookEvent, error) {
	if signature == "" {
		return WebhookEvent{}, errors.New("mock webhook: missing signature")
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return WebhookEvent{}, fmt.Errorf("mock webhook: bad json: %w", err)
	}
	ev := WebhookEvent{Raw: raw}
	if v, ok := raw["event_id"].(string); ok {
		ev.GatewayEventID = v
	}
	if v, ok := raw["txn_id"].(string); ok {
		ev.TxnID = v
	}
	if v, ok := raw["status"].(string); ok {
		ev.Status = v
	}
	if v, ok := raw["amount"].(float64); ok {
		ev.Amount = v
	}
	return ev, nil
}
