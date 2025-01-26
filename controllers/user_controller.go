package controllers

import (
	"dailzo/config"
	"dailzo/globals"
	"dailzo/models"
	"dailzo/repository"
	"dailzo/utils"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	repo *repository.UserRepository
}

func NewUserController(repo *repository.UserRepository) *UserController {
	return &UserController{repo: repo}
}

func (c *UserController) CreateUser(ctx *fiber.Ctx) error {
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	fmt.Print("User details:", user)
	id, err := c.repo.CreateUser(ctx.Context(), user)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create user"})
	}

	// Log user creation
	log := config.SetupLogger()
	log.Info().Msgf("User created with ID: %d", id)

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *UserController) Login(ctx *fiber.Ctx) error {
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	// Check user in database
	dbUser, err := c.repo.GetUserByEmail(ctx, user.Email)
	fmt.Println(dbUser)
	if err != nil {
		fmt.Println(err)
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid email or password"})
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
	if err != nil {
		fmt.Println(err.Error())
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid password"})
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(dbUser.ID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate token"})
	}

	// Log user login
	log := config.SetupLogger()
	log.Info().Msgf("User logged in with ID: %d", dbUser.ID)
	globals.UpdateUserID(dbUser.ID)
	var favouriteFoods string
	if dbUser.FavouriteFoods != nil {
		favouriteFoods = *dbUser.FavouriteFoods // Dereference the pointer
	} else {
		favouriteFoods = "" // Or assign a default value if you prefer
	}
	var favouriteRestaurants string
	if dbUser.FavouriteRestaurants != nil {
		favouriteRestaurants = *dbUser.FavouriteRestaurants
	} else {
		favouriteRestaurants = ""
	}

	// Add user to shared state

	sess, err := globals.Store.Get(ctx)
	sess.Set("userID", dbUser.ID)
	sess.Set("username", dbUser.Username)
	sess.Set("favouriteFoods", favouriteFoods)
	sess.Set("favouriteRestaurants", favouriteRestaurants)
	sess.Save()
	return ctx.JSON(fiber.Map{"token": token})
}

func (c *UserController) GetUser(ctx *fiber.Ctx) error {
	userID := ctx.Locals("user_id").(string)
	fmt.Println("UserId: ", userID)
	user, err := c.repo.GetUserByID(ctx.Context(), string(userID))
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return ctx.JSON(user)
}

func (c *UserController) GetUserById(ctx *fiber.Ctx) error {
	userID := ctx.Params("id")
	fmt.Println("UserId: ", userID)
	if userID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Error in Id"})
	}
	fmt.Println("UserId: ", userID)
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
