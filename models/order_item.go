package models

import "time"

// OrderItem represents an order item in the database
type OrderItem struct {
	ID               string    `json:"id"`                 // Unique ID for the order item
	OrderID          string    `json:"order_id"`           // ID of the order this item belongs to
	ProductVariantID string    `json:"product_variant_id"` // ID of the product variant in the order
	Quantity         int       `json:"quantity"`           // Quantity of the product variant ordered
	Price            float64   `json:"price"`              // Price of the product variant
	CreatedOn        time.Time `json:"created_on"`         // Timestamp when the order item was created
	LastUpdatedOn    time.Time `json:"last_updated_on"`    // Timestamp when the order item was last updated
	CreatedBy        string    `json:"created_by"`         // User who created the order item
	LastModifiedBy   string    `json:"last_modified_by"`   // User who last modified the order item
}
