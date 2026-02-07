package server

import (
	"dailzo/internal/api"
	"dailzo/models"
	"dailzo/pkg/response"
	"dailzo/utils"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// PostAuthLogin handles POST /auth/login
func (s *Server) PostAuthLogin(c *fiber.Ctx) error {
	var req api.PostAuthLoginJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Email == nil || req.Password == nil {
		return response.BadRequest(c, "Email and password are required")
	}

	// Get user by email (optimized for login - context-based)
	user, err := s.userRepo.GetUserByEmailForLogin(c.Context(), *req.Email)
	if err != nil {
		return response.Unauthorized(c, "Invalid email or password")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(*req.Password))
	if err != nil {
		return response.Unauthorized(c, "Invalid email or password")
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Success(c, fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"mobile_no":  user.MobileNo,
		},
	})
}

// PostAuthSignup handles POST /auth/signup
func (s *Server) PostAuthSignup(c *fiber.Ctx) error {
	var req api.PostAuthSignupJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Email == nil || req.Password == nil {
		return response.BadRequest(c, "Email and password are required")
	}

	userType := "customer"
	if req.UserType != nil {
		userType = *req.UserType
	}

	// Create user
	user := models.User{
		Email:    *req.Email,
		Password: *req.Password,
		UserType: userType,
	}

	id, err := s.userRepo.CreateUser(c.Context(), user)
	if err != nil {
		return response.InternalError(c, "Failed to create user")
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(id)
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Created(c, fiber.Map{
		"id":    id,
		"token": token,
	})
}

// PostAuthSendOtp handles POST /auth/send-otp
func (s *Server) PostAuthSendOtp(c *fiber.Ctx) error {
	var req api.PostAuthSendOtpJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Determine target (email or mobile)
	target := ""
	otpType := "email"
	if req.Email != nil && *req.Email != "" {
		target = *req.Email
		otpType = "email"
	} else if req.Mobile != nil && *req.Mobile != "" {
		target = *req.Mobile
		otpType = "mobile"
	} else {
		return response.BadRequest(c, "Email or mobile is required")
	}

	// Create OTP (generates and saves)
	_, err := s.otpRepo.CreateOTP(c.Context(), target, otpType)
	if err != nil {
		return response.InternalError(c, "Failed to create OTP")
	}

	// In production, send OTP via email/SMS
	return response.Success(c, fiber.Map{
		"message": "OTP sent successfully",
		"target":  target,
		"type":    otpType,
	})
}

// PostAuthVerifyOtp handles POST /auth/verify-otp
func (s *Server) PostAuthVerifyOtp(c *fiber.Ctx) error {
	var req api.PostAuthVerifyOtpJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.UserId == nil || req.Otp == nil {
		return response.BadRequest(c, "User ID and OTP are required")
	}

	// Verify OTP
	valid, err := s.otpRepo.VerifyOTP(c.Context(), *req.UserId, *req.Otp)
	if err != nil || !valid {
		return response.BadRequest(c, "Invalid or expired OTP")
	}

	// Generate token
	token, err := utils.GenerateJWT(*req.UserId)
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Success(c, fiber.Map{
		"message": "OTP verified successfully",
		"token":   token,
	})
}

// GetAuthMe handles GET /auth/me
func (s *Server) GetAuthMe(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	user, err := s.userRepo.GetUserByID(c.Context(), userID)
	if err != nil {
		return response.NotFound(c, "User not found")
	}

	return response.Success(c, fiber.Map{
		"id":                     user.ID,
		"email":                  user.Email,
		"first_name":             user.FirstName,
		"last_name":              user.LastName,
		"user_type":              user.UserType,
		"mobile_no":              user.MobileNo,
		"address":                user.AddressID,
		"registration_completed": user.Email != "" && user.MobileNo != "",
	})
}

// PatchAuthProfile handles PATCH /auth/profile
func (s *Server) PatchAuthProfile(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Get current user
	user, err := s.userRepo.GetUserByID(c.Context(), userID)
	if err != nil {
		return response.NotFound(c, "User not found")
	}

	// Apply updates
	if firstName, ok := updates["first_name"].(string); ok {
		user.FirstName = &firstName
	}
	if lastName, ok := updates["last_name"].(string); ok {
		user.LastName = &lastName
	}
	if mobileNo, ok := updates["mobile_no"].(string); ok {
		user.MobileNo = mobileNo
	}

	// Update user
	err = s.userRepo.UpdateUser(c.Context(), *user)
	if err != nil {
		return response.InternalError(c, "Failed to update profile")
	}

	return response.Success(c, fiber.Map{
		"message": "Profile updated successfully",
	})
}
