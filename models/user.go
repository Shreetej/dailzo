package models

import "time"

type User struct {
	ID                   string    `json:"id"`       // Unique identifier for the user
	Username             string    `json:"username"` // Username for the user
	Email                string    `json:"email"`    // Email address for the user
	MobileNo             string    `json:"mobileno"`
	FirstName            *string   `json:"first_name"` // First name of the user
	MiddleName           *string   `json:"middle_name"`
	LastName             *string   `json:"last_name"`         // Last name of the user
	// Password hash. Keep the "password" json tag (needed to parse it off
	// inbound login/signup request bodies) — handlers must never pass a raw
	// User struct straight to ctx.JSON(); use safeUserMap() instead so this
	// field never reaches a response.
	Password             string    `json:"password"`
	UserType             string    `json:"user_type"`         // Type of user (e.g., admin, customer)
	AddressID            *string   `json:"address_id"`        // Foreign key to the Address table
	ProfileImageURL      *string   `json:"profile_image_url"` // URL to the user's profile image
	Bio                  *string   `json:"bio"`               // Short biography or description of the user
	DateOfBirth          *string   `json:"date_of_birth"`     // Date of birth of the user
	Gender               *bool     `json:"gender"`
	CreatedOn            time.Time `json:"created_on"`            // Timestamp when the user was created
	LastUpdatedOn        time.Time `json:"last_updated_on"`       // Timestamp when the user was last updated
	CreatedBy            *string   `json:"created_by"`            // ID of the user who created this user (if applicable)
	LastModifiedBy       *string   `json:"last_modified_by"`      // ID of the user who last modified this user (if applicable)
	// Comma-separated id list — the favourite_restaurants/favourite_foods
	// columns are plain TEXT, not a Postgres array, so this must stay *string
	// (not *[]string) or pgx fails to scan any non-null value. Callers split
	// on "," at the presentation layer (see safeUserMap/favouriteList).
	FavouriteRestaurants *string `json:"favourite_restaurants"` // Comma-separated favorite restaurant ids
	FavouriteFoods       *string `json:"favourite_foods"`       // Comma-separated favorite food/product ids
}

type DisplayUser struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	MobileNo      string     `json:"mobileno"`
	CreatedOn     *time.Time `json:"created_on"`
	LastUpdatedOn *time.Time `json:"last_updated_on"`
}
