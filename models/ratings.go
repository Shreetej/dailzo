package models

import "time"

// Rating represents a rating record in the database
type Rating struct {
	ID             string    `json:"id"`               // Unique ID for the rating
	Rating         int       `json:"rating"`           // Rating score (e.g., 1 to 5)
	Comment        string    `json:"comment"`          // Optional comment for the rating
	UserID         string    `json:"user_id"`          // ID of the user who gave the rating
	EntityType     string    `json:"entity_type"`      // Type of entity being rated (e.g., restaurant, product)
	EntityID       string    `json:"entity_id"`        // ID of the entity being rated (e.g., restaurant_id, product_id)
	CreatedOn      time.Time `json:"created_on"`       // Timestamp when the rating was created
	LastUpdatedOn  time.Time `json:"last_updated_on"`  // Timestamp when the rating was last updated
	CreatedBy      string    `json:"created_by"`       // User who created the rating
	LastModifiedBy string    `json:"last_modified_by"` // User who last modified the rating
}
