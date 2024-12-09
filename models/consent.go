package models

import "time"

type Consent struct {
	ID             string     `json:"id"`
	EntityToVerify string     `json:"entity_to_verify"`
	OTP            string     `json:"otp"`
	OTPSentOn      *time.Time `json:"otp_sent_on"`
	OTPExpiredOn   *time.Time `json:"otp_expired_on"`
	CreatedDate    time.Time  `json:"created_date"`
	CreatedBy      string     `json:"created_by"`
	LastModifiedBy string     `json:"last_modified_by"`
	LastModifiedOn time.Time  `json:"last_modified_on"`
	VerifiedOn     *time.Time `json:"verified_on"`
}
