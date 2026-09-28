package payments

import (
	"context"
	"dailzo/models"
	"errors"
)

// ErrNotConfigured is returned by real provider adapters until their
// credentials/SDK are provisioned. Selecting a provider without config should
// fail loudly rather than silently no-op.
var ErrNotConfigured = errors.New("payment gateway not configured")

// HyperPayGateway — stub. Fill in Sale/Refund/Poll/VerifyWebhook against the
// HyperPay COPYandPAY / Server-to-Server API once ENTITY_ID + access token are
// available. See docs/PAYMENTS_REDESIGN.md.
type HyperPayGateway struct {
	EntityID    string
	AccessToken string
	BaseURL     string
}

func (g *HyperPayGateway) Name() string { return models.GatewayHyperPay }
func (g *HyperPayGateway) Sale(context.Context, SaleRequest) (SaleResult, error) {
	return SaleResult{}, ErrNotConfigured
}
func (g *HyperPayGateway) Refund(context.Context, string, float64) (string, error) {
	return "", ErrNotConfigured
}
func (g *HyperPayGateway) Poll(context.Context, string) (string, error) {
	return "", ErrNotConfigured
}
func (g *HyperPayGateway) VerifyWebhook(string, []byte) (WebhookEvent, error) {
	return WebhookEvent{}, ErrNotConfigured
}

// PayTabsGateway — stub. Fill in against PayTabs hosted/managed API once
// profile_id + server key are available.
type PayTabsGateway struct {
	ProfileID string
	ServerKey string
	BaseURL   string
}

func (g *PayTabsGateway) Name() string { return models.GatewayPayTabs }
func (g *PayTabsGateway) Sale(context.Context, SaleRequest) (SaleResult, error) {
	return SaleResult{}, ErrNotConfigured
}
func (g *PayTabsGateway) Refund(context.Context, string, float64) (string, error) {
	return "", ErrNotConfigured
}
func (g *PayTabsGateway) Poll(context.Context, string) (string, error) {
	return "", ErrNotConfigured
}
func (g *PayTabsGateway) VerifyWebhook(string, []byte) (WebhookEvent, error) {
	return WebhookEvent{}, ErrNotConfigured
}
