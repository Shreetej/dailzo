package models

import (
	"time"
)

// Restaurant represents a restaurant record in the database
type Restaurant struct {
	ID             string    `json:"id"`           // Unique ID for the restaurant
	DisplayName    string    `json:"display_name"` // Name of the restaurant
	Name           string    `json:"name"`         // Name of the restaurant
	OwnerFullName  string    `json:"owner_full_name"`
	UserType       string    `json:"user_type"`
	Address        string    `json:"address"` // Address of the restaurant
	Mobile         string    `json:"mobile"`
	PhoneNumber    string    `json:"phone_number"`     // Phone number of the restaurant
	Email          string    `json:"email"`            // Email address of the restaurant
	OpeningTime    time.Time `json:"opening_time"`     // Opening time of the restaurant
	ClosingTime    time.Time `json:"closing_time"`     // Closing time of the restaurant
	CreatedOn      time.Time `json:"created_on"`       // Timestamp when the restaurant record was created
	LastUpdatedOn  time.Time `json:"last_updated_on"`  // Timestamp when the restaurant record was last updated
	CreatedBy      string    `json:"created_by"`       // User who created the restaurant record
	LastModifiedBy string    `json:"last_modified_by"` // User who last modified the restaurant record
}

// Restaurant represents a restaurant record in the database
type DisplayRestaurant struct {
	ID          string    `json:"id"`           // Unique ID for the restaurant
	Name        string    `json:"name"`         // Name of the restaurant
	Address     string    `json:"address"`      // Address of the restaurant
	PhoneNumber string    `json:"phone_number"` // Phone number of the restaurant
	Email       string    `json:"email"`        // Email address of the restaurant
	OpeningTime time.Time `json:"opening_time"` // Opening time of the restaurant
	ClosingTime time.Time `json:"closing_time"` // Closing time of the restaurant
	// below values to check dynamically
	Distance        float64 `json:"distance"`         // Distance from the user's location
	Rating          float64 `json:"rating"`           // Rating of the restaurant
	DeliveryTimings string  `json:"delivery_timings"` // Delivery timings of the restaurant
	IsFavorite      bool    `json:"is_favorite"`      // Whether the restaurant is a favorite of the user
	//Offer           DisplayOffer `json:"offers"`           // Offer available at the restaurant
}

type DisplayRestaurantWithOffers struct {
	ID          string    `json:"id"`           // Unique ID for the restaurant
	Name        string    `json:"name"`         // Name of the restaurant
	Address     string    `json:"address"`      // Address of the restaurant
	PhoneNumber string    `json:"phone_number"` // Phone number of the restaurant
	Email       string    `json:"email"`        // Email address of the restaurant
	OpeningTime time.Time `json:"opening_time"` // Opening time of the restaurant
	ClosingTime time.Time `json:"closing_time"` // Closing time of the restaurant
	// below values to check dynamically
	Distance        float64        `json:"distance"`         // Distance from the user's location
	Rating          float64        `json:"rating"`           // Rating of the restaurant
	DeliveryTimings string         `json:"delivery_timings"` // Delivery timings of the restaurant
	IsFavorite      bool           `json:"is_favorite"`      // Whether the restaurant is a favorite of the user
	Offers          []DisplayOffer `json:"offers"`           // Offer available at the restaurant
}
