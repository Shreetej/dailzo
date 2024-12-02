package controllers

import (
	"dailzo/config"
	"dailzo/globals"
	"dailzo/models"
	"dailzo/repository"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type FoodProductController struct {
	repo *repository.FoodProductRepository
}

func NewFoodProductController(repo *repository.FoodProductRepository) *FoodProductController {
	return &FoodProductController{repo: repo}
}

func (c *FoodProductController) CreateFoodProduct(ctx *fiber.Ctx) error {
	var foodProduct models.FoodProduct
	if err := ctx.BodyParser(&foodProduct); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	globals.UpdateUserID(ctx.Locals("user_id").(string))
	fmt.Print("User details:", ctx.Locals("user_id"))
	id, err := c.repo.CreateFoodProduct(ctx.Context(), foodProduct)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create user"})
	}

	// Log user creation
	log := config.SetupLogger()
	log.Info().Msgf("Address created with ID: %d", id)

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}
