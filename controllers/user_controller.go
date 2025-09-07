package controllers

import (
	"dailzo/globals"
	"dailzo/models"
	"dailzo/repository"
	"dailzo/utils"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	repo    *repository.UserRepository
	otpRepo *repository.OTPRepository
}

func NewUserController(repo *repository.UserRepository, otpRepo *repository.OTPRepository) *UserController {
	return &UserController{repo: repo, otpRepo: otpRepo}
}

func (c *UserController) CreateUser(ctx *fiber.Ctx) error {
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	// User creation request received
	id, err := c.repo.CreateUser(ctx.Context(), user)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create user"})
	}

	// User created successfully

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *UserController) Login(ctx *fiber.Ctx) error {
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	// Check user in database (optimized for login)
	dbUser, err := c.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid email or password"})
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid password"})
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(dbUser.ID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate token"})
	}

	// Update global user ID
	globals.UpdateUserID(dbUser.ID)

	// Set minimal session data for performance
	sess, err := globals.Store.Get(ctx)
	if err == nil {
		sess.Set("userID", dbUser.ID)
		sess.Set("username", dbUser.Username)
		sess.Save()
	}

	// Return all user fields with token included
	favouriteFoods := []string{}
	if dbUser.FavouriteFoods != nil {
		favouriteFoods = *dbUser.FavouriteFoods
	}
	favouriteRestaurants := []string{}
	if dbUser.FavouriteRestaurants != nil {
		favouriteRestaurants = *dbUser.FavouriteRestaurants
	}

	return ctx.JSON(fiber.Map{
		"id":                    dbUser.ID,
		"username":              dbUser.Username,
		"email":                 dbUser.Email,
		"mobile_no":             dbUser.MobileNo,
		"first_name":            dbUser.FirstName,
		"middle_name":           dbUser.MiddleName,
		"last_name":             dbUser.LastName,
		"user_type":             dbUser.UserType,
		"address_id":            dbUser.AddressID,
		"profile_image_url":     dbUser.ProfileImageURL,
		"bio":                   dbUser.Bio,
		"date_of_birth":         dbUser.DateOfBirth,
		"gender":                dbUser.Gender,
		"created_by":            dbUser.CreatedBy,
		"last_modified_by":      dbUser.LastModifiedBy,
		"favourite_foods":       favouriteFoods,
		"favourite_restaurants": favouriteRestaurants,
		"token":                 token,
	})
}

func (c *UserController) GetUser(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(string)
	user, err := c.repo.GetUserByID(ctx.Context(), string(userID))
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return ctx.JSON(user)
}

func (c *UserController) GetUserById(ctx *fiber.Ctx) error {
	userID := ctx.Params("id")
	if userID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Error in Id"})
	}
	user, err := c.repo.GetUserByID(ctx.Context(), string(userID))
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return ctx.JSON(user)
}

func (c *UserController) GetUsers(ctx *fiber.Ctx) error {
	users, err := c.repo.GetUsers(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return ctx.JSON(users)
}

// UpdateUser handles updating a user's information
func (c *UserController) UpdateUser(ctx *fiber.Ctx) error {
	var user models.User

	// Parse request body
	if err := ctx.BodyParser(&user); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Update user in the database
	if err := c.repo.UpdateUser(ctx.Context(), user); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User updated successfully",
	})
}

// DeleteUser handles deleting a user by ID
func (c *UserController) DeleteUser(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	userID := idParam
	if userID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Delete user from the database
	if err := c.repo.DeleteUser(ctx.Context(), userID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// UpdateFavoriteRestaurant updates the user's favorite restaurants
func (c *UserController) UpdateFavoriteRestaurant(ctx *fiber.Ctx) error {
	newFavoriteRestaurant := ctx.FormValue("restaurant")
	if newFavoriteRestaurant == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "restaurant is required"})
	}

	err := c.repo.UpdateFavoriteRestaurant(ctx, newFavoriteRestaurant)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Favorite restaurant updated successfully"})
}

// UpdateFavoriteFoods updates the user's favorite foods
func (c *UserController) UpdateFavoriteFoods(ctx *fiber.Ctx) error {
	newFavoriteFood := ctx.FormValue("food")
	if newFavoriteFood == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "food is required"})
	}

	err := c.repo.UpdateFavoriteFoods(ctx, newFavoriteFood)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Favorite food updated successfully"})
}

// RemoveFavoriteRestaurant removes a restaurant from the user's favorite restaurants
func (c *UserController) RemoveFavoriteRestaurant(ctx *fiber.Ctx) error {
	restaurantToRemove := ctx.FormValue("restaurant")
	if restaurantToRemove == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "restaurant is required"})
	}

	err := c.repo.RemoveFavoriteRestaurant(ctx, restaurantToRemove)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Favorite restaurant removed successfully"})
}

// RemoveFavoriteFood removes a food from the user's favorite foods
func (c *UserController) RemoveFavoriteFood(ctx *fiber.Ctx) error {
	foodToRemove := ctx.FormValue("food")
	if foodToRemove == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "food is required"})
	}

	err := c.repo.RemoveFavoriteFood(ctx, foodToRemove)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return ctx.JSON(fiber.Map{"message": "Favorite food removed successfully"})
}

// SendOTP sends OTP to user's email or mobile
func (c *UserController) SendOTP(ctx *fiber.Ctx) error {
	startTime := time.Now()
	var request struct {
		Email    string `json:"email"`
		Mobile   string `json:"mobile"`
		UserType string `json:"user_type"`
		Type     string `json:"type"` // "email" or "mobile"
	}

	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	// Find user by email or mobile
	var user *models.User
	var err error
	if request.Email != "" {
		user, err = c.repo.GetUserByEmail(ctx, request.Email)
	} else if request.Mobile != "" {
		// Need to add GetUserByMobile method or modify GetUserByEmail
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "mobile login not implemented yet"})
	} else {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email or mobile required"})
	}

	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Time taken to find user: %s\n", elapsedTime)

	// Generate and store OTP
	otpType := "email"
	if request.Type == "mobile" {
		otpType = "mobile"
	}

	otp, err := c.otpRepo.CreateOTP(ctx.Context(), user.ID, otpType)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate OTP"})
	}

	elapsedTime = time.Since(startTime)
	fmt.Printf("Time taken to generate OTP: %s\n", elapsedTime)

	// Send OTP asynchronously
	go func() {
		var sendErr error
		if otpType == "email" {
			sendErr = utils.SendOTPEmail(user.Email, otp.OTPCode)
		} else {
			sendErr = utils.SendOTPSMS(user.MobileNo, otp.OTPCode)
		}

		if sendErr != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to send OTP: %v\n", sendErr)
		} else {
			fmt.Printf("OTP sent successfully to %s\n", user.Email)
		}
	}()

	elapsedTime = time.Since(startTime)
	fmt.Printf("Time taken to send OTP (async): %s\n", elapsedTime)

	return ctx.JSON(fiber.Map{"message": "OTP sent successfully", "user_id": user.ID})
}

// VerifyOTPLogin verifies OTP and logs in user
func (c *UserController) VerifyOTPLogin(ctx *fiber.Ctx) error {
	startTime := time.Now()
	var request struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		OTP    string `json:"otp"`
	}

	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	var userID string
	if request.UserID != "" {
		userID = request.UserID
	} else if request.Email != "" {
		// Get user by email if user_id not provided (optimized)
		user, err := c.repo.GetUserByEmail(ctx, request.Email)
		if err != nil {
			return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		userID = user.ID
	} else {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id or email required"})
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Time taken to get user ID: %s\n", elapsedTime)

	// Verify OTP
	valid, err := c.otpRepo.VerifyOTP(ctx.Context(), userID, request.OTP)
	if err != nil || !valid {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid OTP"})
	}

	elapsedTime = time.Since(startTime)
	fmt.Printf("Time taken to verify OTP: %s\n", elapsedTime)

	// Get user details
	user, err := c.repo.GetUserByID(ctx.Context(), userID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	elapsedTime = time.Since(startTime)
	fmt.Printf("Time taken to get user details: %s\n", elapsedTime)

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate token"})
	}

	elapsedTime = time.Since(startTime)
	fmt.Printf("Time taken to generate JWT: %s\n", elapsedTime)

	// Set session
	sess, err := globals.Store.Get(ctx)
	sess.Set("userID", user.ID)
	sess.Set("username", user.Username)
	sess.Save()

	elapsedTime = time.Since(startTime)
	fmt.Printf("Total time for OTP login: %s\n", elapsedTime)

	// Return all user fields with token included
	favouriteFoods := []string{}
	if user.FavouriteFoods != nil {
		favouriteFoods = *user.FavouriteFoods
	}
	favouriteRestaurants := []string{}
	if user.FavouriteRestaurants != nil {
		favouriteRestaurants = *user.FavouriteRestaurants
	}

	return ctx.JSON(fiber.Map{
		"id":                    user.ID,
		"username":              user.Username,
		"email":                 user.Email,
		"mobile_no":             user.MobileNo,
		"first_name":            user.FirstName,
		"middle_name":           user.MiddleName,
		"last_name":             user.LastName,
		"user_type":             user.UserType,
		"address_id":            user.AddressID,
		"profile_image_url":     user.ProfileImageURL,
		"bio":                   user.Bio,
		"date_of_birth":         user.DateOfBirth,
		"gender":                user.Gender,
		"created_by":            user.CreatedBy,
		"last_modified_by":      user.LastModifiedBy,
		"favourite_foods":       favouriteFoods,
		"favourite_restaurants": favouriteRestaurants,
		"token":                 token,
	})
}
