package models

import (
	"time"
)

type ProductVariant struct {
	ID                    string    `json:"id" db:"id"`                                         // Unique identifier for the product variant
	ProductID             string    `json:"product_id" db:"product_id"`                         // Foreign key referencing the food_product table
	VariantName           string    `json:"variant_name" db:"variant_name"`                     // E.g., Small, Medium, Large
	AdditionalDescription string    `json:"additional_description" db:"additional_description"` // Description like "Spicy", "Cheese-filled"
	Price                 float64   `json:"price" db:"price"`                                   // Price for the variant
	QuantityAvailable     int       `json:"quantity_available" db:"quantity_available"`         // Available quantity for this variant
	IsActive              bool      `json:"is_active" db:"is_active"`                           // Whether this variant is active
	CreatedOn             time.Time `json:"created_on" db:"created_on"`                         // Timestamp when variant was created
	LastUpdatedOn         time.Time `json:"last_updated_on" db:"last_updated_on"`               // Timestamp when variant was last updated
	CreatedBy             string    `json:"created_by" db:"created_by"`                         // User ID who created the variant
	LastModifiedBy        string    `json:"last_modified_by" db:"last_modified_by"`             // User ID who last modified the variant
}
