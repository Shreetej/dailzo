package controllers

import (
	"dailzo/models"
	"dailzo/utils"
	"fmt"
	"net/smtp"
	"os"

	"github.com/gofiber/fiber/v2"
)

// EmailController struct to group all email-related functions
//type EmailController struct{}

type EmailController struct {
	consentController *ConsentController // Injecting ConsentController
}

// EmailRequest struct for parsing incoming POST data
type EmailRequest struct {
	To      string `json:"to" validate:"required,email"`
	Subject string `json:"subject" validate:"required"`
	Message string `json:"message" validate:"required"`
}

// EmailRequest struct for parsing incoming POST data
type EmailOTPVerify struct {
	Otp   string `json:"to" validate:"required,email"`
	Email string `json:"subject" validate:"required"`
}

// NewEmailController returns a new instance of EmailController
func NewEmailController() *EmailController {
	return &EmailController{}
}

func NewEmailControllerWithConsent(consentController *ConsentController) *EmailController {
	return &EmailController{consentController: consentController}
}

// SendEmail handles the actual email-sending logic
func (e *EmailController) SendEmail(to, subject, message string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	email := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	println("Email :", email)
	println("password :", password)

	// Check if SMTP configuration is missing
	if smtpHost == "" || smtpPort == "" || email == "" || password == "" {
		return fmt.Errorf("SMTP configuration is missing in environment variables")
	}

	// Create the email message
	fullMessage := fmt.Sprintf("Subject: %s\n\n%s", subject, message)

	// Authentication for SMTP
	auth := smtp.PlainAuth("", email, password, smtpHost)

	// Send the email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, email, []string{to}, []byte(fullMessage))
	if err != nil {
		return err
	}

	return nil
}

// SendVerifyEmailOtp is the HTTP handler for sending an email (now works with Fiber)
func (e *EmailController) SendVerifyEmailOtp(c *fiber.Ctx) error {
	var emailRequest EmailRequest
	fmt.Printf("Received order: %+v\n", &emailRequest)

	// Parse the incoming JSON request into the EmailRequest struct
	if err := c.BodyParser(&emailRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request data. Please provide 'to', 'subject', and 'message'.",
		})
	}

	otpToSend := utils.GenerateOTP()
	// Call SendEmail to send the email
	// Call CreateConsent method from ConsentController before sending email
	consent := models.Consent{
		EntityToVerify: emailRequest.To,
		OTP:            otpToSend, // Example OTP, you may generate dynamically
	}
	e.consentController.CreateConsent(c, consent)

	emailRequest.Message = "Hello otp is " + otpToSend
	//CreateConsent(c)
	err := e.SendEmail(emailRequest.To, emailRequest.Subject, emailRequest.Message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to send email",
			"details": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Email sent successfully!",
	})
}

// VerifyOTPHandler handles the OTP verification process
func (e *EmailController) VerifyOTPHandler(c *fiber.Ctx) error {
	fmt.Printf("Received order: %+v\n", c.Params)
	// Get the entity (email or phone) and the OTP from the request
	entityToVerify := c.Params("entityToVerify")
	otpEntered := c.FormValue("otp")
	fmt.Printf("Received order: %+v\n", entityToVerify)
	fmt.Printf("Received order: %+v\n", otpEntered)
	// Call the VerifyOTP method from the repository to verify the OTP
	isVerified := e.consentController.VerifyOTP(c, entityToVerify, otpEntered)

	fmt.Printf("isVerified order: %+v\n", isVerified)

	// if err != nil {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": err.Error(),
	// 	})
	// }

	if isVerified == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "OTP verified successfully",
		})
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "Failed to verify OTP",
	})
}
