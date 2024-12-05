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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		globals.GetLoogedInUserId(),
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
	println("Error in query:", query)
	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}

func (r *PaymentMethodRepository) GetPaymentMethodByID(ctx context.Context, id string) (models.PaymentMethod, error) {
	var paymentMethod models.PaymentMethod
	query := `SELECT id, user_id, type, provider, account_number, created_on, last_updated_on, created_by, last_modified_by 
	          FROM payment_methods WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&paymentMethod.ID,
		&paymentMethod.UserID,
		&paymentMethod.Type,
		&paymentMethod.Provider,
		&paymentMethod.AccountNumber,
		&paymentMethod.CreatedOn,
		&paymentMethod.LastUpdatedOn,
		&paymentMethod.CreatedBy,
		&paymentMethod.LastModifiedBy,
	)

	if err != nil {
		return paymentMethod, err
	}

	return paymentMethod, nil
}

func (r *PaymentMethodRepository) GetPaymentMethods(ctx context.Context) ([]models.PaymentMethod, error) {
	var paymentMethods []models.PaymentMethod
	query := `SELECT id, user_id, type, provider, account_number, created_on, last_updated_on, created_by, last_modified_by
	          FROM payment_methods`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var paymentMethod models.PaymentMethod
		if err := rows.Scan(
			&paymentMethod.ID,
			&paymentMethod.UserID,
			&paymentMethod.Type,
			&paymentMethod.Provider,
			&paymentMethod.AccountNumber,
			&paymentMethod.CreatedOn,
			&paymentMethod.LastUpdatedOn,
			&paymentMethod.CreatedBy,
			&paymentMethod.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		paymentMethods = append(paymentMethods, paymentMethod)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return paymentMethods, nil
}

func (r *PaymentMethodRepository) UpdatePaymentMethod(ctx context.Context, paymentMethod models.PaymentMethod) error {
	query := `UPDATE payment_methods
		SET provider = $1, account_number = $2, last_updated_on = $3, last_modified_by = $4
		WHERE id = $5`

	_, err := r.db.Exec(ctx, query,
		paymentMethod.Provider,
		paymentMethod.AccountNumber,
		paymentMethod.LastUpdatedOn,
		paymentMethod.LastModifiedBy,
		paymentMethod.ID,
	)

	return err
}

func (r *PaymentMethodRepository) DeletePaymentMethod(ctx context.Context, id string) error {
	query := `DELETE FROM payment_methods WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
