package models

import "time"

// Payment represents a payment record in the database
type Payment struct {
	ID              string    `json:"id"`                // Unique ID for the payment
	OrderID         string    `json:"order_id"`          // ID of the order associated with this payment
	UserID          string    `json:"user_id"`           // ID of the user making the payment
	PaymentMethodID string    `json:"payment_method_id"` // ID of the payment method used
	Amount          float64   `json:"amount"`            // Payment amount
	Status          string    `json:"status"`            // Payment status (e.g., Completed, Pending)
	TransactionID   string    `json:"transaction_id"`    // Unique transaction ID for this payment
	PaymentDate     time.Time `json:"payment_date"`      // Timestamp when the payment was made
	CreatedOn       time.Time `json:"created_on"`        // Timestamp when the payment record was created
	LastUpdatedOn   time.Time `json:"last_updated_on"`   // Timestamp when the payment record was last updated
	CreatedBy       string    `json:"created_by"`        // User who created the payment record
	LastModifiedBy  string    `json:"last_modified_by"`  // User who last modified the payment record
}
