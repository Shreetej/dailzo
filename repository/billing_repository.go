package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultRegionCode is the market used when a caller names none. It matches
// the customer app's kDefaultRegion.
const DefaultRegionCode = "SA"

// BillingRepository owns every number that turns a cart into an amount due.
//
// These lived in the customer app as constants (handling 10, delivery 30, VAT
// 5%) and were passed to the server at checkout, which trusted them. Fees are
// now a server-side policy an admin edits, and the app renders a quote rather
// than doing arithmetic.
type BillingRepository struct {
	db *pgxpool.Pool
}

func NewBillingRepository(db *pgxpool.Pool) *BillingRepository {
	return &BillingRepository{db: db}
}

// BillingConfig is one market's fee schedule.
type BillingConfig struct {
	RegionCode            string   `json:"region_code"`
	CountryName           string   `json:"country_name"`
	CurrencyCode          string   `json:"currency_code"`
	VATRate               float64  `json:"vat_rate"`
	HandlingFee           float64  `json:"handling_fee"`
	DeliveryFee           float64  `json:"delivery_fee"`
	FreeDeliveryThreshold *float64 `json:"free_delivery_threshold"`
	SmallOrderFee         float64  `json:"small_order_fee"`
	MinOrderAmount        float64  `json:"min_order_amount"`
	IsActive              bool     `json:"is_active"`
}

// CartQuote is the fully priced bill for a cart: exactly what the customer
// app renders, line for line, so no total is ever computed on the device.
type CartQuote struct {
	CurrencyCode string  `json:"currency_code"`
	Subtotal     float64 `json:"subtotal"`
	ItemSavings  float64 `json:"item_savings"`
	HandlingFee  float64 `json:"handling_fee"`
	DeliveryFee  float64 `json:"delivery_fee"`
	// Set when the subtotal cleared free_delivery_threshold, so the app can
	// say why delivery is free instead of just showing zero.
	FreeDelivery  bool    `json:"free_delivery"`
	SmallOrderFee float64 `json:"small_order_fee"`
	Discount      float64 `json:"discount"`
	Tip           float64 `json:"tip"`
	VATRate       float64 `json:"vat_rate"`
	Tax           float64 `json:"tax"`
	Total         float64 `json:"total"`
	ItemCount     int     `json:"item_count"`
	// Below the market's minimum: the app disables checkout and explains.
	BelowMinimum   bool    `json:"below_minimum"`
	MinOrderAmount float64 `json:"min_order_amount"`
}

const billingSelect = `
	SELECT region_code, country_name, currency_code, vat_rate::float8,
	       handling_fee::float8, delivery_fee::float8,
	       free_delivery_threshold::float8, small_order_fee::float8,
	       min_order_amount::float8, is_active
	FROM billing_config`

func scanBillingConfig(row pgx.Row) (BillingConfig, error) {
	var c BillingConfig
	err := row.Scan(&c.RegionCode, &c.CountryName, &c.CurrencyCode, &c.VATRate,
		&c.HandlingFee, &c.DeliveryFee, &c.FreeDeliveryThreshold,
		&c.SmallOrderFee, &c.MinOrderAmount, &c.IsActive)
	return c, err
}

// GetBillingConfig returns one market's schedule, falling back to the platform
// default region when the requested one isn't configured — a cart must always
// be priceable.
func (r *BillingRepository) GetBillingConfig(ctx context.Context, regionCode string) (BillingConfig, error) {
	if regionCode == "" {
		regionCode = DefaultRegionCode
	}

	cfg, err := scanBillingConfig(r.db.QueryRow(ctx, billingSelect+` WHERE region_code = $1 AND is_active`, regionCode))
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return BillingConfig{}, fmt.Errorf("failed to read billing config: %w", err)
	}
	if regionCode == DefaultRegionCode {
		return BillingConfig{}, errors.New("no billing config for the default region")
	}

	cfg, err = scanBillingConfig(r.db.QueryRow(ctx, billingSelect+` WHERE region_code = $1 AND is_active`, DefaultRegionCode))
	if err != nil {
		return BillingConfig{}, fmt.Errorf("failed to read default billing config: %w", err)
	}
	return cfg, nil
}

// ListBillingConfigs returns every market, for the admin editor.
func (r *BillingRepository) ListBillingConfigs(ctx context.Context) ([]BillingConfig, error) {
	rows, err := r.db.Query(ctx, billingSelect+` ORDER BY region_code`)
	if err != nil {
		return nil, fmt.Errorf("failed to list billing configs: %w", err)
	}
	defer rows.Close()

	configs := []BillingConfig{}
	for rows.Next() {
		c, err := scanBillingConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan billing config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// UpsertBillingConfig saves an admin edit. Every field is validated here
// rather than trusted: these numbers decide what customers are charged.
func (r *BillingRepository) UpsertBillingConfig(ctx context.Context, c BillingConfig, actorID string) error {
	if c.RegionCode == "" {
		return errors.New("region_code is required")
	}
	if c.CurrencyCode == "" {
		return errors.New("currency_code is required")
	}
	if c.VATRate < 0 || c.VATRate > 1 {
		return errors.New("vat_rate must be a fraction between 0 and 1")
	}
	for name, v := range map[string]float64{
		"handling_fee":     c.HandlingFee,
		"delivery_fee":     c.DeliveryFee,
		"small_order_fee":  c.SmallOrderFee,
		"min_order_amount": c.MinOrderAmount,
	} {
		if v < 0 {
			return fmt.Errorf("%s cannot be negative", name)
		}
	}
	if c.FreeDeliveryThreshold != nil && *c.FreeDeliveryThreshold < 0 {
		return errors.New("free_delivery_threshold cannot be negative")
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO billing_config
		    (region_code, country_name, currency_code, vat_rate, handling_fee,
		     delivery_fee, free_delivery_threshold, small_order_fee,
		     min_order_amount, is_active, last_updated_on, last_modified_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (region_code) DO UPDATE SET
		    country_name = EXCLUDED.country_name,
		    currency_code = EXCLUDED.currency_code,
		    vat_rate = EXCLUDED.vat_rate,
		    handling_fee = EXCLUDED.handling_fee,
		    delivery_fee = EXCLUDED.delivery_fee,
		    free_delivery_threshold = EXCLUDED.free_delivery_threshold,
		    small_order_fee = EXCLUDED.small_order_fee,
		    min_order_amount = EXCLUDED.min_order_amount,
		    is_active = EXCLUDED.is_active,
		    last_updated_on = EXCLUDED.last_updated_on,
		    last_modified_by = EXCLUDED.last_modified_by`,
		c.RegionCode, c.CountryName, c.CurrencyCode, c.VATRate, c.HandlingFee,
		c.DeliveryFee, c.FreeDeliveryThreshold, c.SmallOrderFee,
		c.MinOrderAmount, c.IsActive, time.Now(), actorID)
	if err != nil {
		return fmt.Errorf("failed to save billing config: %w", err)
	}
	return nil
}

// QuoteCart prices a cart under one market's policy.
//
// This is the single place fees and VAT are calculated. The quote endpoint and
// Checkout both call it, so the bill the customer approves and the amount the
// order records come from identical arithmetic.
func (r *BillingRepository) QuoteCart(
	ctx context.Context,
	cfg BillingConfig,
	subtotal, mrpTotal, discount, tip float64,
	itemCount int,
) CartQuote {
	q := CartQuote{
		CurrencyCode:   cfg.CurrencyCode,
		Subtotal:       round2(subtotal),
		ItemSavings:    round2(maxFloat(mrpTotal-subtotal, 0)),
		Discount:       round2(discount),
		Tip:            round2(tip),
		VATRate:        cfg.VATRate,
		ItemCount:      itemCount,
		MinOrderAmount: cfg.MinOrderAmount,
	}

	// An empty cart costs nothing — no fees, no minimum warning.
	if itemCount == 0 || subtotal <= 0 {
		return q
	}

	q.HandlingFee = round2(cfg.HandlingFee)

	q.DeliveryFee = round2(cfg.DeliveryFee)
	if cfg.FreeDeliveryThreshold != nil && subtotal >= *cfg.FreeDeliveryThreshold {
		q.DeliveryFee = 0
		q.FreeDelivery = true
	}

	if subtotal < cfg.MinOrderAmount {
		q.SmallOrderFee = round2(cfg.SmallOrderFee)
		q.BelowMinimum = true
	}

	// VAT applies to goods and service fees, after discount, and never to the
	// tip (it goes to the rider, not the platform).
	taxable := maxFloat(q.Subtotal+q.HandlingFee+q.DeliveryFee+q.SmallOrderFee-q.Discount, 0)
	q.Tax = round2(taxable * cfg.VATRate)
	q.Total = round2(taxable + q.Tax + q.Tip)
	return q
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// round2 keeps money at two decimals so the quote, the order row and the
// payment row agree to the halala.
func round2(v float64) float64 {
	return float64(int64(v*100+signOf(v)*0.5)) / 100
}

func signOf(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
