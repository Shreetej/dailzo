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
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&refund.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}
