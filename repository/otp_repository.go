package repository

import (
	"context"
	"dailzo/models"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OTPRepository struct {
	db *pgxpool.Pool
}

func NewOTPRepository(db *pgxpool.Pool) *OTPRepository {
	return &OTPRepository{db: db}
}

func (r *OTPRepository) GenerateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (r *OTPRepository) CreateOTP(ctx context.Context, userID, otpType string) (*models.OTP, error) {
	id := GetIdToRecord("OTP")
	otpCode := r.GenerateOTP()
	expiresAt := time.Now().Add(5 * time.Minute) // 5 minutes expiry

	query := `INSERT INTO otps (id, user_id, otp_code, type, created_at, expires_at, used) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err := r.db.QueryRow(ctx, query, id, userID, otpCode, otpType, time.Now(), expiresAt, false).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &models.OTP{
		ID:        id,
		UserID:    userID,
		OTPCode:   otpCode,
		Type:      otpType,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		Used:      false,
	}, nil
}

func (r *OTPRepository) VerifyOTP(ctx context.Context, userID, otpCode string) (bool, error) {
	var otp models.OTP
	query := `SELECT id, user_id, otp_code, type, created_at, expires_at, used 
	          FROM otps WHERE user_id = $1 AND otp_code = $2 AND used = false AND expires_at > $3`

	err := r.db.QueryRow(ctx, query, userID, otpCode, time.Now()).Scan(
		&otp.ID, &otp.UserID, &otp.OTPCode, &otp.Type, &otp.CreatedAt, &otp.ExpiresAt, &otp.Used,
	)
	if err != nil {
		return false, err
	}

	// Mark OTP as used
	updateQuery := `UPDATE otps SET used = true WHERE id = $1`
	_, err = r.db.Exec(ctx, updateQuery, otp.ID)
	if err != nil {
		return false, err
	}

	return true, nil
}
