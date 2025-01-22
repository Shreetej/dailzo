package models

import "time"

type User struct {
	ID                   string  `json:"id"`       // Unique identifier for the user
	Username             string  `json:"username"` // Username for the user
	Email                string  `json:"email"`    // Email address for the user
	MobileNo             string  `json:"mobileno"`
	FirstName            *string `json:"first_name"` // First name of the user
	MiddleName           *string `json:"middle_name"`
	LastName             *string `json:"last_name"`             // Last name of the user
	Password             string  `json:"password"`              // Password hash for authentication
	CreatedOn            string  `json:"created_on"`            // Timestamp when the user was created
	LastUpdatedOn        string  `json:"last_updated_on"`       // Timestamp when the user was last updated
	CreatedBy            *string `json:"created_by"`            // ID of the user who created this user (if applicable)
	LastModifiedBy       *string `json:"last_modified_by"`      // ID of the user who last modified this user (if applicable)
	FavouriteRestaurants *string `json:"favourite_restaurants"` // List of favorite restaurants
	FavouriteFoods       *string `json:"favourite_foods"`       // List of favorite foods
}

type DisplayUser struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	MobileNo      string     `json:"mobileno"`
	CreatedOn     *time.Time `json:"created_on"`
	LastUpdatedOn *time.Time `json:"last_updated_on"`
}
