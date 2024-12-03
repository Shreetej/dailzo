package controllers

import (
	"dailzo/config"
	"dailzo/globals"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type ProductVariantController struct {
	repo *repository.ProductVariantRepository
}

func NewProductVariantController(repo *repository.ProductVariantRepository) *ProductVariantController {
	return &ProductVariantController{repo: repo}
}

func (c *ProductVariantController) CreateProductVariant(ctx *fiber.Ctx) error {
	var productVariant models.ProductVariant
	if err := ctx.BodyParser(&productVariant); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	globals.UpdateUserID(ctx.Locals("user_id").(string))

	id, err := c.repo.CreateProductVariant(ctx.Context(), productVariant)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create variant"})
	}

	// Log user creation
	log := config.SetupLogger()
	log.Info().Msgf("variant created with ID: %d", id)

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}
