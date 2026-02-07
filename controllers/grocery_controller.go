package controllers

import (
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// GroceryController handles grocery-related API requests
type GroceryController struct {
	repo *repository.GroceryRepository
}

// NewGroceryController creates a new GroceryController instance
func NewGroceryController(repo *repository.GroceryRepository) *GroceryController {
	return &GroceryController{repo: repo}
}

// Onboarding handles grocery vendor onboarding
func (c *GroceryController) Onboarding(ctx *fiber.Ctx) error {
	var profile models.GroceryProfileCreate
	if err := ctx.BodyParser(&profile); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreateProfile(ctx.Context(), &profile)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create grocery profile"})
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// Profile retrieves grocery vendor profile
func (c *GroceryController) Profile(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing grocery id"})
	}
	profile, err := c.repo.GetProfileByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "grocery profile not found"})
	}
	return ctx.JSON(profile)
}
