package utils

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
)

func SendOTPEmail(email, otp string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || smtpHost == "" || smtpPort == "" {
		return fmt.Errorf("email configuration not set")
	}

	// Email content
	subject := "Your OTP Code"
	body := fmt.Sprintf("Your OTP code is: %s\nThis code will expire in 5 minutes.", otp)
	message := fmt.Sprintf("Subject: %s\n\n%s", subject, body)

	// Send email with optimized SMTP client
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Create SMTP client
	client, err := smtp.Dial(smtpHost + ":" + smtpPort)
	if err != nil {
		return err
	}
	defer client.Close()

	// Start TLS if available
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: smtpHost}); err != nil {
			return err
		}
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Set sender and recipient
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(email); err != nil {
		return err
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	// Send QUIT
	err = client.Quit()
	if err != nil {
		return err
	}

	return nil
}

func SendOTPSMS(phone, otp string) error {
	// For SMS, you would integrate with an SMS service like Twilio
	// For now, just print to console
	fmt.Printf("SMS OTP to %s: %s\n", phone, otp)
	return nil
}
