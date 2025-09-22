package models

import (
	"time"
)

// Outlet represents a restaurant record in the database

type Outlet struct {
	ID             string    `json:"id"`               // Unique ID for the restaurant
	Name           string    `json:"name"`             // Name of the restaurant
	Address        string    `json:"address"`          // Address of the restaurant
	PhoneNumber    string    `json:"phone_number"`     // Phone number of the restaurant
	Email          string    `json:"email"`            // Email address of the restaurant
	OpeningTime    time.Time `json:"opening_time"`     // Opening time of the restaurant
	ClosingTime    time.Time `json:"closing_time"`     // Closing time of the restaurant
	CreatedOn      time.Time `json:"created_on"`       // Timestamp when the restaurant record was created
	LastUpdatedOn  time.Time `json:"last_updated_on"`  // Timestamp when the restaurant record was last updated
	CreatedBy      string    `json:"created_by"`       // User who created the restaurant record
	LastModifiedBy string    `json:"last_modified_by"` // User who last modified the restaurant record
}
