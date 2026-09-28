package repository

import (
	"context"
	"crypto/rand"
	"dailzo/models"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrOTPRateLimited is returned when a caller has requested too many OTPs
// for one user in the throttle window.
var ErrOTPRateLimited = errors.New("too many OTP requests, please try again later")

// ErrOTPLocked is returned once an OTP has accumulated too many failed
// verification attempts — the caller must request a new OTP rather than
// keep guessing the same one.
var ErrOTPLocked = errors.New("too many incorrect attempts, request a new OTP")

const (
	otpSendWindow    = 10 * time.Minute
	otpSendMax       = 5 // at most 5 OTPs issued per user per window
	otpVerifyAttempt = 5 // at most 5 guesses against one OTP before it locks
)

type OTPRepository struct {
	db *pgxpool.Pool
}

func NewOTPRepository(db *pgxpool.Pool) *OTPRepository {
	return &OTPRepository{db: db}
}

// GenerateOTP returns a 6-digit code from crypto/rand — math/rand (the
// previous implementation) is not a CSPRNG and, seeded from the wall clock,
// is predictable enough to matter for something guarding account access.
func (r *OTPRepository) GenerateOTP() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// crypto/rand failing means the system RNG is unavailable — extremely
		// unusual, but fail loud rather than silently falling back to a
		// weaker source.
		panic(fmt.Errorf("otp: crypto/rand unavailable: %w", err))
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// CreateOTP issues a new OTP, refusing once the caller has requested more
// than otpSendMax within otpSendWindow (per user) — a 6-digit OTP is only as
// safe as the number of guesses/sends an attacker gets to make.
func (r *OTPRepository) CreateOTP(ctx context.Context, userID, otpType string) (*models.OTP, error) {
	var recentCount int
	countQuery := `SELECT COUNT(*) FROM otps WHERE user_id = $1 AND created_at > $2`
	if err := r.db.QueryRow(ctx, countQuery, userID, time.Now().Add(-otpSendWindow)).Scan(&recentCount); err != nil {
		return nil, err
	}
	if recentCount >= otpSendMax {
		return nil, ErrOTPRateLimited
	}

	id := GetIdToRecord("OTP")
	otpCode := r.GenerateOTP()
	expiresAt := time.Now().Add(5 * time.Minute) // 5 minutes expiry

	query := `INSERT INTO otps (id, user_id, otp_code, type, created_at, expires_at, used, attempts)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, 0) RETURNING id`

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

// VerifyOTP checks the code against the most recent unused, unexpired OTP for
// this user. Every call against that row increments its attempt counter
// (right or wrong) so a wrong guess costs the attacker one of a small,
// fixed budget instead of being free to retry forever.
func (r *OTPRepository) VerifyOTP(ctx context.Context, userID, otpCode string) (bool, error) {
	var otp models.OTP
	var attempts int
	query := `SELECT id, user_id, otp_code, type, created_at, expires_at, used, attempts
	          FROM otps WHERE user_id = $1 AND used = false AND expires_at > $2
	          ORDER BY created_at DESC LIMIT 1`

	err := r.db.QueryRow(ctx, query, userID, time.Now()).Scan(
		&otp.ID, &otp.UserID, &otp.OTPCode, &otp.Type, &otp.CreatedAt, &otp.ExpiresAt, &otp.Used, &attempts,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, errors.New("invalid or expired OTP")
		}
		return false, err
	}

	if attempts >= otpVerifyAttempt {
		return false, ErrOTPLocked
	}

	if _, err := r.db.Exec(ctx, `UPDATE otps SET attempts = attempts + 1 WHERE id = $1`, otp.ID); err != nil {
		return false, err
	}

	if otp.OTPCode != otpCode {
		return false, nil
	}

	if _, err := r.db.Exec(ctx, `UPDATE otps SET used = true WHERE id = $1`, otp.ID); err != nil {
		return false, err
	}

	return true, nil
}
