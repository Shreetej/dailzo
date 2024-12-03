package models

import "time"

// PaymentMethod represents a payment method record in the database
type PaymentMethod struct {
	ID                string    `json:"id"`                  // Unique ID for the payment method
	UserID            string    `json:"user_id"`             // ID of the user associated with this payment method
	Type              string    `json:"type"`                // Type of the payment method (e.g., Credit, Debit, Net Banking)
	Provider          string    `json:"provider"`            // Payment provider (e.g., Visa, MasterCard, PayPal)
	AccountNumber     string    `json:"account_number"`      // Account number (e.g., card number or bank account number)
	ExpiryDate        time.Time `json:"expiry_date"`         // Expiry date (for cards)
	IsDefault         bool      `json:"is_default"`          // Flag indicating if this is the default payment method
	CreatedOn         time.Time `json:"created_on"`          // Timestamp when the payment method was created
	LastUpdatedOn     time.Time `json:"last_updated_on"`     // Timestamp when the payment method was last updated
	CreatedBy         string    `json:"created_by"`          // User who created the payment method
	LastModifiedBy    string    `json:"last_modified_by"`    // User who last modified the payment method
	NameOnCard        string    `json:"name_on_card"`        // Name on the card (for card payments)
	CardType          string    `json:"card_type"`           // Type of card (e.g., Credit, Debit)
	CvvEncrypted      string    `json:"cvv_encrypted"`       // Encrypted CVV (for card payments)
	BankName          string    `json:"bank_name"`           // Bank name (for bank account payments)
	IfscCode          string    `json:"ifsc_code"`           // IFSC code (for bank accounts)
	AccountHolderName string    `json:"account_holder_name"` // Account holder's name (for bank accounts)
	IsActive          bool      `json:"is_active"`           // Flag indicating if the payment method is active
}
