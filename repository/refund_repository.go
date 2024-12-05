package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RefundRepository struct {
	db *pgxpool.Pool
}

func NewRefundRepository(db *pgxpool.Pool) *RefundRepository {
	return &RefundRepository{db: db}
}

// CreateRefund inserts a new refund record into the database
func (r *RefundRepository) CreateRefund(ctx context.Context, refund models.Refund) (string, error) {

	id := GetIdToRecord("RFND")
	query := `INSERT INTO refunds 
		(id, payment_id, user_id, amount, reason, status, refund_date, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		refund.PaymentID,
		refund.UserID,
		refund.Amount,
		refund.Reason,
		refund.Status,
		time.Now(),
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&refund.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}
func (r *RefundRepository) GetRefundByID(ctx context.Context, id string) (models.Refund, error) {
	var refund models.Refund
	query := `SELECT id, payment_id, user_id, amount, reason, status, refund_date, created_on, last_updated_on, created_by, last_modified_by 
	          FROM refunds WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&refund.ID,
		&refund.PaymentID,
		&refund.UserID,
		&refund.Amount,
		&refund.Reason,
		&refund.Status,
		&refund.RefundDate,
		&refund.CreatedOn,
		&refund.LastUpdatedOn,
		&refund.CreatedBy,
		&refund.LastModifiedBy,
	)

	if err != nil {
		return refund, err
	}

	return refund, nil
}

func (r *RefundRepository) GetRefunds(ctx context.Context) ([]models.Refund, error) {
	var refunds []models.Refund
	query := `SELECT id, payment_id, user_id, amount, reason, status, refund_date, created_on, last_updated_on, created_by, last_modified_by 
	          FROM refunds`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var refund models.Refund
		if err := rows.Scan(
			&refund.ID,
			&refund.PaymentID,
			&refund.UserID,
			&refund.Amount,
			&refund.Reason,
			&refund.Status,
			&refund.RefundDate,
			&refund.CreatedOn,
			&refund.LastUpdatedOn,
			&refund.CreatedBy,
			&refund.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		refunds = append(refunds, refund)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return refunds, nil
}

func (r *RefundRepository) UpdateRefund(ctx context.Context, refund models.Refund) error {
	query := `UPDATE refunds
		SET payment_id = $1, user_id = $2, amount = $3, reason = $4, status = $5, refund_date = $6, 
		last_updated_on = $7, last_modified_by = $8
		WHERE id = $9`

	_, err := r.db.Exec(ctx, query,
		refund.PaymentID,
		refund.UserID,
		refund.Amount,
		refund.Reason,
		refund.Status,
		refund.RefundDate,
		refund.LastUpdatedOn,
		refund.LastModifiedBy,
		refund.ID,
	)

	return err
}

func (r *RefundRepository) DeleteRefund(ctx context.Context, id string) error {
	query := `DELETE FROM refunds WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
