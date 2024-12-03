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
func (r *PaymentRepository) GetPaymentByID(ctx context.Context, id string) (models.Payment, error) {
	var payment models.Payment
	query := `SELECT id, order_id, user_id, payment_method_id, amount, status, transaction_id, payment_date, created_on, last_updated_on, created_by, last_modified_by 
	          FROM payments WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.UserID,
		&payment.PaymentMethodID,
		&payment.Amount,
		&payment.Status,
		&payment.TransactionID,
		&payment.PaymentDate,
		&payment.CreatedOn,
		&payment.LastUpdatedOn,
		&payment.CreatedBy,
		&payment.LastModifiedBy,
	)

	if err != nil {
		return payment, err
	}

	return payment, nil
}

func (r *PaymentRepository) GetPayments(ctx context.Context) ([]models.Payment, error) {
	var payments []models.Payment
	query := `SELECT id, order_id, user_id, payment_method_id, amount, status, transaction_id, payment_date, created_on, last_updated_on, created_by, last_modified_by 
	          FROM payments`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var payment models.Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.UserID,
			&payment.PaymentMethodID,
			&payment.Amount,
			&payment.Status,
			&payment.TransactionID,
			&payment.PaymentDate,
			&payment.CreatedOn,
			&payment.LastUpdatedOn,
			&payment.CreatedBy,
			&payment.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PaymentRepository) UpdatePayment(ctx context.Context, payment models.Payment) error {
	query := `UPDATE payments
		SET order_id = $1, user_id = $2, payment_method_id = $3, amount = $4, status = $5, transaction_id = $6, payment_date = $7, 
		last_updated_on = $8, last_modified_by = $9
		WHERE id = $10`

	_, err := r.db.Exec(ctx, query,
		payment.OrderID,
		payment.UserID,
		payment.PaymentMethodID,
		payment.Amount,
		payment.Status,
		payment.TransactionID,
		payment.PaymentDate,
		payment.LastUpdatedOn,
		payment.LastModifiedBy,
		payment.ID,
	)

	return err
}

func (r *PaymentRepository) DeletePayment(ctx context.Context, id string) error {
	query := `DELETE FROM payments WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
