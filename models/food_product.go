package models

import (
	"time"
)

// FoodProduct represents the food_product table in the database
type FoodProduct struct {
	ID             string    `json:"id" db:"id"`                             // Unique identifier for the food product
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
}

type DisplayFoodProducts struct {
	ID           string  `json:"id" db:"id"`                    // Unique identifier for the food product
	Name         string  `json:"name" db:"name"`                // Name of the food product
	Description  string  `json:"description" db:"description"`  // Description of the product
	Category     string  `json:"category" db:"category"`        // Category like Pizza, Grocery, etc.
	Type         string  `json:"type" db:"type"`                // Food type (Veg, Non-Veg, Egg)
	Price        float64 `json:"price" db:"price"`              // Price of the product
	ImageURL     string  `json:"image_url" db:"image_url"`      // URL for the product image
	IsActive     bool    `json:"is_active" db:"is_active"`      // Whether the product is active
	RestaurantId string  `json:"restaurant_id" db:"restaurant"` // User ID who created the product
	Restaurants  []DisplayRestaurant
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
