package models

import "time"

// GroceryProfile represents a grocery vendor profile
type GroceryProfile struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	StoreName            string    `json:"store_name"`
	OwnerName            string    `json:"owner_name"`
	Email                string    `json:"email"`
	Phone                string    `json:"phone"`
	Address              string    `json:"address"`
	City                 string    `json:"city"`
	Pincode              string    `json:"pincode,omitempty"`
	KYCStatus            string    `json:"kyc_status"`
	KYCDocuments         *string   `json:"kyc_documents,omitempty"` // JSON string
	FSSAILicense         string    `json:"fssai_license,omitempty"`
	GSTNumber            string    `json:"gst_number,omitempty"`
	PANNumber            string    `json:"pan_number,omitempty"`
	PayoutStatus         string    `json:"payout_status"`
	BankDetails          *string   `json:"bank_details,omitempty"` // JSON string
	WorkingHours         string    `json:"working_hours"`
	HomeDelivery         bool      `json:"home_delivery"`
	AcceptsOnlinePayment bool      `json:"accepts_online_payment"`
	StoreType            string    `json:"store_type"`
	StoreSize            string    `json:"store_size"`
	ServiceArea          string    `json:"service_area"`
	IsActive             bool      `json:"is_active"`
	Rating               float64   `json:"rating"`
	TotalOrders          int       `json:"total_orders"`
	CommissionRate       float64   `json:"commission_rate"`
	CreatedOn            time.Time `json:"created_on"`
	LastUpdatedOn        time.Time `json:"last_updated_on"`
	CreatedBy            *string   `json:"created_by,omitempty"`
	LastModifiedBy       *string   `json:"last_modified_by,omitempty"`
}

// GroceryProfileCreate is used for onboarding
type GroceryProfileCreate struct {
	UserID       string `json:"user_id"`
	StoreName    string `json:"store_name"`
	OwnerName    string `json:"owner_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	City         string `json:"city"`
	Pincode      string `json:"pincode"`
	FSSAILicense string `json:"fssai_license"`
	GSTNumber    string `json:"gst_number"`
	PANNumber    string `json:"pan_number"`
	WorkingHours string `json:"working_hours"`
}

// GroceryKPI represents grocery vendor KPIs
type GroceryKPI struct {
	ID              string  `json:"id,omitempty"`
	GroceryID       string  `json:"grocery_id,omitempty"`
	Date            string  `json:"date,omitempty"`
	TodayRevenue    float64 `json:"today_revenue"`
	PendingOrders   int     `json:"pending_orders"`
	CompletedOrders int     `json:"completed_orders,omitempty"`
	CancelledOrders int     `json:"cancelled_orders,omitempty"`
	CancelRate      float64 `json:"cancel_rate"`
	AvgPrepTime     string  `json:"avg_prep_time"`
	AvgPrepTimeMins int     `json:"avg_prep_time_mins,omitempty"`
	LowStockItems   int     `json:"low_stock_items"`
	ExpiryRiskItems int     `json:"expiry_risk_items"`
}

// GroceryExpiryAlert represents expiry alerts
type GroceryExpiryAlert struct {
	ProductID     string  `json:"product_id"`
	Name          string  `json:"name"`
	DaysLeft      int     `json:"days_left"`
	ExpiryDate    string  `json:"expiry_date"`
	StockQuantity int     `json:"stock_quantity"`
	Category      string  `json:"category,omitempty"`
	Price         float64 `json:"price,omitempty"`
}

// GroceryStockAlert represents stock alerts
type GroceryStockAlert struct {
	ProductID     string  `json:"product_id"`
	Name          string  `json:"name"`
	StockQuantity int     `json:"stock_quantity"`
	Threshold     int     `json:"threshold"`
	Category      string  `json:"category,omitempty"`
	Price         float64 `json:"price,omitempty"`
}

// GroceryPayoutSummary represents payout summary
type GroceryPayoutSummary struct {
	NextPayoutDate     string  `json:"next_payout_date"`
	Amount             float64 `json:"amount"`
	Status             string  `json:"status"`
	PendingAmount      float64 `json:"pending_amount,omitempty"`
	LastPayoutDate     string  `json:"last_payout_date,omitempty"`
	LastPayoutAmount   float64 `json:"last_payout_amount,omitempty"`
	TotalEarnings      float64 `json:"total_earnings,omitempty"`
	CommissionDeducted float64 `json:"commission_deducted,omitempty"`
}

// GroceryStoreSettings mirrors the vendor app's GroceryStoreSettings model
// (lib/features/grocery/domain/models/grocery_models.dart) for GET /grocery/settings.
type GroceryStoreSettings struct {
	HomeDelivery         bool   `json:"home_delivery"`
	AcceptsOnlinePayment bool   `json:"accepts_online_payment"`
	StoreType            string `json:"store_type"`
	StoreSize            string `json:"store_size"`
	ServiceArea          string `json:"service_area"`
}

// GroceryDocument mirrors the vendor app's GroceryDocument model for
// GET /grocery/documents. Backed by the profile's kyc_documents JSON column.
type GroceryDocument struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GroceryPayoutAccount mirrors the vendor app's GroceryPayoutAccount model
// for GET /grocery/payout-account. Backed by the profile's bank_details JSON column.
type GroceryPayoutAccount struct {
	BankName        string `json:"bank_name"`
	AccountNumber   string `json:"account_number"`
	IFSC            string `json:"ifsc"`
	BeneficiaryName string `json:"beneficiary_name"`
}

// GroceryPayout represents a single payout record
type GroceryPayout struct {
	ID            string    `json:"id"`
	GroceryID     string    `json:"grocery_id"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	PayoutDate    *string   `json:"payout_date,omitempty"`
	TransactionID string    `json:"transaction_id,omitempty"`
	BankReference string    `json:"bank_reference,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	CreatedOn     time.Time `json:"created_on"`
}
