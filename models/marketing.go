package models

import (
	"encoding/json"
	"time"
)

// DiscountCampaign is a vendor-created discount from the Partner app.
// JSON field names match the Partner app's DiscountCampaign model.
type DiscountCampaign struct {
	ID            string  `json:"id"`
	RestaurantID  *string `json:"restaurant_id,omitempty"`
	Code          string  `json:"code"`
	Type          string  `json:"type"`    // percentage | flatOff | eliteExclusive
	Segment       string  `json:"segment"` // all | newCustomers | returningCustomers | dormantCustomers
	Percent       *int    `json:"percent"`
	FlatAmount    *int    `json:"flat_amount"`
	MinOrderValue int     `json:"min_order_value"`
	CappingAmount *int    `json:"capping_amount"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date"`
	Status        string  `json:"status"` // active | upcoming | inactive
}

// AdCampaign is a vendor ad campaign (quick pack or custom listing ad).
type AdCampaign struct {
	ID              string   `json:"id"`
	RestaurantID    *string  `json:"restaurant_id,omitempty"`
	Kind            string   `json:"kind"` // quickPack | custom
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	Cpc             float64  `json:"cpc"`
	TargetCustomers string   `json:"target_customers"`
	Timeslot        string   `json:"timeslot"`
	OutletIDs       []string `json:"outlet_ids"`
	StartDate       string   `json:"start_date"`
	DurationDays    int      `json:"duration_days"`
	BudgetLine      string   `json:"budget_line"`
	Clicks          int      `json:"clicks"`
	Spend           int      `json:"spend"`
}

// AdPack is a predefined quick-setup campaign pack.
type AdPack struct {
	ID           string  `json:"id"`
	Tier         string  `json:"tier"` // recommended | standard | turbo
	Description  string  `json:"description"`
	Clicks       int     `json:"clicks"`
	Price        int     `json:"price"`
	Cpc          float64 `json:"cpc"`
	DurationDays int     `json:"duration_days"`
}

// VendorOutlet is an outlet created from the Partner app registration flow.
// Nested structures are stored as JSONB.
type VendorOutlet struct {
	ID              string          `json:"id"`
	RestaurantID    string          `json:"restaurant_id"`
	OutletType      string          `json:"outlet_type"`
	Menu            string          `json:"menu"`
	Cuisines        []string        `json:"cuisines"`
	CostForTwo      int             `json:"cost_for_two"`
	AvgDeliveryTime string          `json:"avg_delivery_time"`
	Address         json.RawMessage `json:"address"`
	PackagingCharge json.RawMessage `json:"packaging_charge"`
	OperatingHours  json.RawMessage `json:"operating_hours"`
	IsActive        bool            `json:"is_active"`
	CreatedOn       time.Time       `json:"created_on,omitempty"`
}
