package utils

import (
	"crypto/rand"
	"math/big"
)

// GenerateOTP generates a random 6-digit OTP
func GenerateOTP() string {
	numbers := "0123456789"
	otp := make([]byte, 6)
	for i := range otp {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(numbers))))
		otp[i] = numbers[randomIndex.Int64()]
	}
	return string(otp)
}
