package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentMethodRepository struct {
	db *pgxpool.Pool
}

func NewPaymentMethodRepository(db *pgxpool.Pool) *PaymentMethodRepository {
	return &PaymentMethodRepository{db: db}
}

// CreatePaymentMethod inserts a new payment method into the database
func (r *PaymentMethodRepository) CreatePaymentMethod(ctx context.Context, paymentMethod models.PaymentMethod) (string, error) {

	id := GetIdToRecord("PAYMTD")
	query := `INSERT INTO payment_methods 
		(id, user_id, type, provider, account_number, expiry_date, is_default, created_on, last_updated_on, created_by, last_modified_by, name_on_card, card_type, cvv_encrypted, bank_name, ifsc_code, account_holder_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		paymentMethod.UserID,
		paymentMethod.Type,
		paymentMethod.Provider,
		paymentMethod.AccountNumber,
		paymentMethod.ExpiryDate,
		paymentMethod.IsDefault,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
		paymentMethod.NameOnCard,
		paymentMethod.CardType,
		paymentMethod.CvvEncrypted,
		paymentMethod.BankName,
		paymentMethod.IfscCode,
		paymentMethod.AccountHolderName,
		paymentMethod.IsActive,
	).Scan(&paymentMethod.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}
