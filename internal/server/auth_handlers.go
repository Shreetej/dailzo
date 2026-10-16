package server

import (
	"errors"

	"dailzo/internal/api"
	"dailzo/models"
	"dailzo/pkg/response"
	"dailzo/repository"
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
	token, err := utils.GenerateJWT(user.ID, user.UserType)
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

// publicSignupRoles are the only user_type values a client may self-assign at
// signup. "admin" and "owner" are privileged roles and must be granted by an
// existing admin, never chosen by the caller of a public endpoint.
var publicSignupRoles = map[string]bool{
	"customer":         true,
	"restaurant":       true,
	"grocery":          true,
	"delivery_partner": true,
	"delivery_vendor":  true,
}

func isPublicSignupRole(userType string) bool {
	return publicSignupRoles[userType]
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
		if !isPublicSignupRole(*req.UserType) {
			return response.BadRequest(c, "Invalid user_type")
		}
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
		if errors.Is(err, repository.ErrEmailTaken) {
			return response.Conflict(c, "An account with this email already exists")
		}
		return response.InternalError(c, "Failed to create user")
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(id, userType)
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Created(c, fiber.Map{
		"id":    id,
		"token": token,
	})
}

// PostAuthSendOtp handles POST /auth/send-otp. otps.user_id is a real users.id
// FK, not the raw email/mobile string, so the OTP is always tied to one
// resolvable account: mobile is find-or-create (phone-first login/signup is a
// single flow), email requires an existing account.
func (s *Server) PostAuthSendOtp(c *fiber.Ctx) error {
	var req api.PostAuthSendOtpJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	var userID, target, otpType string
	switch {
	case req.Mobile != nil && *req.Mobile != "":
		target = *req.Mobile
		otpType = "mobile"
		user, err := s.userRepo.GetUserByMobile(c.Context(), target)
		if err != nil {
			id, cErr := s.userRepo.CreateUser(c.Context(), models.User{
				MobileNo: target,
				UserType: "customer",
			})
			if cErr != nil {
				return response.InternalError(c, "Failed to create account")
			}
			userID = id
		} else {
			userID = user.ID
		}
	case req.Email != nil && *req.Email != "":
		target = *req.Email
		otpType = "email"
		user, err := s.userRepo.GetUserByEmailForLogin(c.Context(), target)
		if err != nil {
			return response.NotFound(c, "No account found for this email")
		}
		userID = user.ID
	default:
		return response.BadRequest(c, "Email or mobile is required")
	}

	otp, err := s.otpRepo.CreateOTP(c.Context(), userID, otpType)
	if err != nil {
		if errors.Is(err, repository.ErrOTPRateLimited) {
			return response.TooManyRequests(c, err.Error())
		}
		return response.InternalError(c, "Failed to create OTP")
	}

	// Best-effort delivery: email sends for real over SMTP when configured;
	// SMS has no carrier wired in yet (utils.SendOTPSMS logs to the server
	// console instead) — plug in a real provider there once one is chosen.
	// Either way, the OTP is already saved above, so a delivery failure here
	// shouldn't fail the request; it's still verifiable by whoever can see it.
	if otpType == "email" {
		_ = utils.SendOTPEmail(target, otp.OTPCode)
	} else {
		_ = utils.SendOTPSMS(target, otp.OTPCode)
	}

	return response.Success(c, fiber.Map{
		"message": "OTP sent successfully",
		"target":  target,
		"type":    otpType,
		"user_id": userID,
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
	if err != nil {
		if errors.Is(err, repository.ErrOTPLocked) {
			return response.TooManyRequests(c, err.Error())
		}
		return response.BadRequest(c, "Invalid or expired OTP")
	}
	if !valid {
		return response.BadRequest(c, "Invalid or expired OTP")
	}

	user, err := s.userRepo.GetUserByID(c.Context(), *req.UserId)
	if err != nil {
		return response.NotFound(c, "User not found")
	}

	// Generate token
	token, err := utils.GenerateJWT(user.ID, user.UserType)
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Success(c, fiber.Map{
		"message": "OTP verified successfully",
		"token":   token,
		"user_id": *req.UserId,
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
		"date_of_birth":          user.DateOfBirth,
		"gender":                 user.Gender,
		"registration_completed": user.Email != "" && user.MobileNo != "",
		"favourite_restaurants":  utils.SplitCSV(user.FavouriteRestaurants),
		"favourite_foods":        utils.SplitCSV(user.FavouriteFoods),
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
	if dob, ok := updates["date_of_birth"].(string); ok {
		user.DateOfBirth = &dob
	}
	// Gender is stored as a plain bool on this model (no non-binary option
	// exists in the schema yet) — "male" -> true, anything else -> false.
	if genderStr, ok := updates["gender"].(string); ok {
		g := genderStr == "male"
		user.Gender = &g
	}

	// Update the editable profile fields only — never routes through
	// UpdateUser, which would re-hash (and corrupt) the existing password.
	err = s.userRepo.UpdateProfileFields(c.Context(), *user)
	if err != nil {
		return response.InternalError(c, "Failed to update profile")
	}

	return response.Success(c, fiber.Map{
		"message": "Profile updated successfully",
	})
}

// PatchAuthFCMToken handles PATCH /auth/fcm-token — the app calls this after
// obtaining a Firebase device token so server-initiated push (order status
// changes, etc) has somewhere to send to. Previously nothing on the backend
// ever recorded this; the client only logged it locally.
func (s *Server) PatchAuthFCMToken(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return response.BadRequest(c, "token is required")
	}

	if err := s.userRepo.SetFCMToken(c.Context(), userID, body.Token); err != nil {
		return response.InternalError(c, "could not save device token")
	}

	return response.Success(c, fiber.Map{"message": "Device token saved"})
}

// PostAuthLogout handles POST /auth/logout.
//
// Authentication is stateless JWT with no server-side session or revocation
// list, so the actual logout is the client discarding its token — which
// AuthRepositoryImpl.logout already does. This endpoint exists because the app
// calls it first and treats a transport error as a failed logout, which left
// users unable to sign out against a server that answered 404. It confirms the
// call and stays a single place to hook token revocation onto later.
func (s *Server) PostAuthLogout(c *fiber.Ctx) error {
	return response.SuccessWithMessage(c, "logged out", nil)
}
