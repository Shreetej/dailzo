package models

import "time"

// Refund represents a refund record in the database
type Refund struct {
	ID             string    `json:"id"`               // Unique ID for the refund
	PaymentID      string    `json:"payment_id"`       // ID of the payment being refunded
	UserID         string    `json:"user_id"`          // ID of the user requesting the refund
	Amount         float64   `json:"amount"`           // Amount refunded
	Reason         string    `json:"reason"`           // Reason for the refund
	Status         string    `json:"status"`           // Status of the refund (e.g., Pending, Completed)
	RefundDate     time.Time `json:"refund_date"`      // Timestamp when the refund was processed
	CreatedOn      time.Time `json:"created_on"`       // Timestamp when the refund record was created
	LastUpdatedOn  time.Time `json:"last_updated_on"`  // Timestamp when the refund record was last updated
	CreatedBy      string    `json:"created_by"`       // User who created the refund record
	LastModifiedBy string    `json:"last_modified_by"` // User who last modified the refund record
}
