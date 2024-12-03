package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// CreatePayment inserts a new payment record into the database
func (r *PaymentRepository) CreatePayment(ctx context.Context, payment models.Payment) (string, error) {

	id := GetIdToRecord("PYMT")
	query := `INSERT INTO payments 
		(id, order_id, user_id, payment_method_id, amount, status, transaction_id, payment_date, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		payment.OrderID,
		payment.UserID,
		payment.PaymentMethodID,
		payment.Amount,
		payment.Status,
		payment.TransactionID,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&payment.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}
