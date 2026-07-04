package models

import (
	"time"
)

// FoodProduct represents the food_product table in the database
type FoodProduct struct {
	ID             string    `json:"id" db:"id"`                             // Unique identifier for the food product
	RestaurantId   string    `json:"restaurant_id" db:"restaurant"`          // Restaurant the product belongs to
	Name           string    `json:"name" db:"name"`                         // Name of the food product
	Description    string    `json:"description" db:"description"`           // Description of the product
	Category       string    `json:"category" db:"category"`                 // Category like Pizza, Grocery, etc.
	Type           string    `json:"type" db:"type"`                         // Food type (Veg, Non-Veg, Egg)
	Price          float64   `json:"price" db:"price"`                       // Price of the product
	ImageURL       string    `json:"image_url" db:"image_url"`               // URL for the product image
	IsActive       bool      `json:"is_active" db:"is_active"`               // Whether the product is active
	CreatedOn      time.Time `json:"created_on" db:"created_on"`             // Timestamp when the product was created
	LastUpdatedOn  time.Time `json:"last_updated_on" db:"last_updated_on"`   // Timestamp when the product was last updated
	CreatedBy      string    `json:"created_by" db:"created_by"`             // User ID who created the product
	LastModifiedBy string    `json:"last_modified_by" db:"last_modified_by"` // User ID who last modified the product
	// Inventory fields
	StockQuantity     *int       `json:"stock_quantity,omitempty" db:"stock_quantity"`
	LowStockThreshold *int       `json:"low_stock_threshold,omitempty" db:"low_stock_threshold"`
	ExpiryDate        *time.Time `json:"expiry_date,omitempty" db:"expiry_date"`
	BatchNumber       *string    `json:"batch_number,omitempty" db:"batch_number"`
	// Promotion fields
	IsPromo                      *bool    `json:"is_promo,omitempty" db:"is_promo"`
	PromoPrice                   *float64 `json:"promo_price,omitempty" db:"promo_price"`
	AutoDiscountEnabled          *bool    `json:"auto_discount_enabled,omitempty" db:"auto_discount_enabled"`
	AutoDiscountDaysBeforeExpiry *int     `json:"auto_discount_days_before_expiry,omitempty" db:"auto_discount_days_before_expiry"`
	AutoDiscountPct              *float64 `json:"auto_discount_pct,omitempty" db:"auto_discount_pct"`
	// Outlet/Store fields
	OutletID      *string  `json:"outlet_id,omitempty" db:"outlet_id"`
	StorageType   *string  `json:"storage_type,omitempty" db:"storage_type"`
	Brand         *string  `json:"brand,omitempty" db:"brand"`
	Unit          *string  `json:"unit,omitempty" db:"unit"`
	Weight        *float64 `json:"weight,omitempty" db:"weight"`
	ShelfLifeDays *int     `json:"shelf_life_days,omitempty" db:"shelf_life_days"`
}

type DisplayFoodProductsWithVariants struct {
	ID           string           `json:"id" db:"id"`                    // Unique identifier for the food product
	Name         string           `json:"name" db:"name"`                // Name of the food product
	Description  string           `json:"description" db:"description"`  // Description of the product
	Category     string           `json:"category" db:"category"`        // Category like Pizza, Grocery, etc.
	Type         string           `json:"type" db:"type"`                // Food type (Veg, Non-Veg, Egg)
	Price        float64          `json:"price" db:"price"`              // Price of the product
	ImageURL     string           `json:"image_url" db:"image_url"`      // URL for the product image
	IsActive     bool             `json:"is_active" db:"is_active"`      // Whether the product is active
	RestaurantId string           `json:"restaurant_id" db:"restaurant"` // User ID who created the product
	Variants     []ProductVariant `json:"variants" db:"variants"`        // User ID who created the product
}

type DisplayFoodCatagoryProducts struct {
	Name         string `json:"name" db:"name"`                // Name of the food product
	Description  string `json:"description" db:"description"`  // Description of the product
	Category     string `json:"category" db:"category"`        // Category like Pizza, Grocery, etc.
	Type         string `json:"type" db:"type"`                // Food type (Veg, Non-Veg, Egg)
	ImageURL     string `json:"image_url" db:"image_url"`      // URL for the product image
	IsActive     bool   `json:"is_active" db:"is_active"`      // Whether the product is active
	RestaurantId string `json:"restaurant_id" db:"restaurant"` // User ID who created the product
	Restaurants  []DisplayRestaurantWithOffers
}
