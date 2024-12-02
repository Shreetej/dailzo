package models

import "time"

type Address struct {
	ID             string   `json:"id"`             // Unique identifier for the address
	Name           string   `json:"name"`           // Username for the user
	AddressLine1   string   `json:"address_line_1"` // Required first address line
	AddressLine2   *string  `json:"address_line_2"` // Optional second address line
	AddressLine3   *string  `json:"address_line_3"` // Optional third address line
	ZIPPin         string   `json:"zip_pin"`        // ZIP or PIN code
	Benchmark      *string  `json:"benchmark"`      // Optional benchmark field
	UserID         string   `json:"user_id"`        // Associated user ID
	City           *string  `json:"city"`           // Optional city field
	State          *string  `json:"state"`          // Optional state field
	Type           string   `json:"type"`           // Type of address
	MobileNo       string   `json:"mobileno"`
	Longitude      *float64 `json:"longitude"`        // Longitude field (optional)
	Latitude       *float64 `json:"latitude"`         // Latitude field (optional)
	CreatedOn      string   `json:"created_on"`       // Timestamp when the address was created
	LastUpdatedOn  string   `json:"last_updated_on"`  // Timestamp when the address was last updated
	CreatedBy      *string  `json:"created_by"`       // ID of the user who created this address
	LastModifiedBy *string  `json:"last_modified_by"` // ID of the user who last modified this address
}

type DisplayAddress struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	MobileNo      string     `json:"mobileno"`
	CreatedOn     *time.Time `json:"created_at"`
	LastUpdatedOn *time.Time `json:"updated_at"`
	Longitude     *float64   `json:"longitude"` // Longitude field (optional)
	Latitude      *float64   `json:"latitude"`  // Latitude field (optional)
}
