package models

import (
	"time"
)

type Offer struct {
	ID                string    `json:"id"`                  // Unique identifier for the offer
	Name              string    `json:"name"`                // Name of the offer
	Description       *string   `json:"description"`         // Detailed description (optional)
	DiscountPercent   float64   `json:"discount_percent"`    // Discount percentage
	MaxDiscountAmount float64   `json:"max_discount_amount"` // Maximum discount amount
	StartDate         time.Time `json:"start_date"`          // Offer start date (ISO-8601 format: YYYY-MM-DD)
	EndDate           time.Time `json:"end_date"`            // Offer end date (ISO-8601 format: YYYY-MM-DD)
	IsActive          bool      `json:"is_active"`           // Whether the offer is active
	CreatedOn         time.Time `json:"created_on"`          // Creation timestamp
	LastUpdatedOn     time.Time `json:"last_updated_on"`     // Last update timestamp
	CreatedBy         *string   `json:"created_by"`          // User ID who created the record (optional)
	LastModifiedBy    *string   `json:"last_modified_by"`    // User ID who last updated the record (optional)
}

type DisplayOffer struct {
	ID                      string                  `json:"id"`                        // Unique identifier for the offer
	Name                    string                  `json:"name"`                      // Name of the offer
	Description             *string                 `json:"description"`               // Detailed description (optional)
	DiscountPercent         float64                 `json:"discount_percent"`          // Discount percentage
	MaxDiscountAmount       float64                 `json:"max_discount_amount"`       // Maximum discount amount
	StartDate               time.Time               `json:"start_date"`                // Offer start date (ISO-8601 format: YYYY-MM-DD)
	EndDate                 time.Time               `json:"end_date"`                  // Offer end date (ISO-8601 format: YYYY-MM-DD)
	IsActive                bool                    `json:"is_active"`                 // Whether the offer is active
	OfferConditions         []OfferCondition        `json:"offer_conditions"`          // Conditions for the offer
	OfferApplicableEntities []OfferApplicableEntity `json:"offer_applicable_entities"` // Applicable entities for the offer
}

type OfferCondition struct {
	ID             string    `json:"id"`               // Unique identifier for the condition
	OfferID        string    `json:"offer_id"`         // Associated offer ID
	ConditionType  string    `json:"condition_type"`   // Type of condition (e.g., "payment_method", "max_price")
	Value          string    `json:"value"`            // Value of the condition (e.g., "VISA", "500")
	CreatedOn      time.Time `json:"created_on"`       // Creation timestamp
	LastUpdatedOn  time.Time `json:"last_updated_on"`  // Last update timestamp
	CreatedBy      *string   `json:"created_by"`       // User ID who created the record (optional)
	LastModifiedBy *string   `json:"last_modified_by"` // User ID who last updated the record (optional)
}

type OfferApplicableEntity struct {
	ID             string    `json:"id"`               // Unique identifier for the applicable entity
	OfferID        string    `json:"offer_id"`         // Associated offer ID
	EntityType     string    `json:"entity_type"`      // Type of entity (e.g., "restaurant", "order", "dish")
	EntityID       string    `json:"entity_id"`        // Specific entity ID
	CreatedOn      time.Time `json:"created_on"`       // Creation timestamp
	LastUpdatedOn  time.Time `json:"last_updated_on"`  // Last update timestamp
	CreatedBy      *string   `json:"created_by"`       // User ID who created the record (optional)
	LastModifiedBy *string   `json:"last_modified_by"` // User ID who last updated the record (optional)
}
