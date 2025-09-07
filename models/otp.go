package models

import "time"

type OTP struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	OTPCode   string    `json:"otp_code"`
	Type      string    `json:"type"` // "email" or "mobile"
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}
