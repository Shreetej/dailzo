package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConsentRepository struct {
	db *pgxpool.Pool
}

func NewConsentRepository(db *pgxpool.Pool) *ConsentRepository {
	return &ConsentRepository{db: db}
}

// 🔥 **Create Consent**
func (r *ConsentRepository) CreateConsent(ctx context.Context, consent models.Consent) (string, error) {
	// Generate unique ID for the consent record
	id := GetIdToRecord("CNSNT")

	// Prepare the query
	query := `INSERT INTO consent 
		(id, entity_to_verify, otp, otp_sent_on, otp_expired_on, created_date, created_by, last_modified_by, last_modified_on) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING id`

	// Execute the query
	err := r.db.QueryRow(ctx, query,
		id,
		consent.EntityToVerify,
		consent.OTP,
		time.Now(),
		consent.OTPExpiredOn,
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
		time.Now(),
	).Scan(&consent.ID)

	if err != nil {
		fmt.Println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}

// 🔥 **Get All Consents**
func (r *ConsentRepository) GetConsents(ctx context.Context) ([]models.Consent, error) {
	query := `SELECT id, entity_to_verify, otp, otp_sent_on, otp_expired_on, created_date, created_by, last_modified_by, last_modified_on, verified_on 
	          FROM consent`

	rows, err := r.db.Query(ctx, query)
	if err == pgx.ErrNoRows {
		return nil, errors.New("no consents found")
	}
	defer rows.Close()

	consents, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Consent])
	if err != nil {
		fmt.Println("Error collecting rows:", err)
		return nil, err
	}

	return consents, nil
}

// 🔥 **Get Consent By ID**
func (r *ConsentRepository) GetConsentByID(ctx context.Context, id string) (models.Consent, error) {
	var consent models.Consent

	query := `SELECT id, entity_to_verify, otp, otp_sent_on, otp_expired_on, created_date, created_by, last_modified_by, last_modified_on, verified_on 
	          FROM consent WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&consent.ID,
		&consent.EntityToVerify,
		&consent.OTP,
		&consent.OTPSentOn,
		&consent.OTPExpiredOn,
		&consent.CreatedDate,
		&consent.CreatedBy,
		&consent.LastModifiedBy,
		&consent.LastModifiedOn,
		&consent.VerifiedOn,
	)

	if err != nil {
		fmt.Println("Error fetching consent:", err)
		return consent, err
	}

	return consent, nil
}

// 🔥 **Get Consent By ID**
func (r *ConsentRepository) GetConsentByEmail(ctx context.Context, email string) (models.Consent, error) {
	var consent models.Consent

	query := `SELECT otp,  otp_expired_on
	          FROM consent WHERE entity_to_verify = $1`

	err := r.db.QueryRow(ctx, query, email).Scan(
		&consent.OTP,
		&consent.OTPExpiredOn,
	)

	if err != nil {
		fmt.Println("Error fetching consent:", err)
		return consent, err
	}

	return consent, nil
}

// 🔥 **Update Consent**
func (r *ConsentRepository) UpdateConsent(ctx context.Context, consent models.Consent) error {
	query := `UPDATE consent 
		SET entity_to_verify = $1, otp = $2, otp_sent_on = $3, otp_expired_on = $4, 
		last_modified_on = $5, last_modified_by = $6, verified_on = $7 
		WHERE id = $8`

	_, err := r.db.Exec(ctx, query,
		consent.EntityToVerify,
		consent.OTP,
		consent.OTPSentOn,
		consent.OTPExpiredOn,
		time.Now(),
		globals.GetLoogedInUserId(),
		consent.VerifiedOn,
		consent.ID,
	)

	return err
}

// 🔥 **Delete Consent**
func (r *ConsentRepository) DeleteConsent(ctx context.Context, id string) error {
	query := `DELETE FROM consent WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *ConsentRepository) VerifyOTP(ctx context.Context, entityToVerify string, otpEntered string) (bool, error) {
	var consent models.Consent
	// Fetch the consent record based on the entity to verify
	query := `SELECT id, otp, otp_sent_on, otp_expired_on, verified_on 
              FROM consent WHERE entity_to_verify = $1`
	err := r.db.QueryRow(ctx, query, entityToVerify).Scan(&consent.ID, &consent.OTP, &consent.OTPSentOn, &consent.OTPExpiredOn, &consent.VerifiedOn)

	if err != nil {
		if err == pgx.ErrNoRows {
			return false, errors.New("no consent record found for this entity")
		}
		return false, err
	}

	// Check if OTP is expired
	// if time.Now().After(consent.OTPExpiredOn) {
	// 	return false, errors.New("OTP has expired")
	// }

	// Check if the OTP entered is correct
	if consent.OTP != otpEntered {
		return false, errors.New("incorrect OTP")
	}

	fmt.Println("consent.OTP: ", consent.OTP)
	// If OTP is correct and not expired, mark the consent as verified
	updateQuery := `UPDATE consent SET verified_on = $1, last_modified_on = $2,  verified = true WHERE id = $3`
	_, err = r.db.Exec(ctx, updateQuery, time.Now(), time.Now(), consent.ID)
	if err != nil {
		fmt.Println("Error fetching consent:", err)
		return false, err
	}

	return true, nil
}
