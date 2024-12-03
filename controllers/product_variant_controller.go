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
func (c *ProductVariantController) GetProductVariantById(ctx *fiber.Ctx) error {
	variantID := ctx.Params("id")
	productVariant, err := c.repo.GetProductVariantByID(ctx.Context(), variantID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product variant not found"})
	}
	return ctx.JSON(productVariant)
}

func (c *ProductVariantController) GetProductVariants(ctx *fiber.Ctx) error {
	productVariants, err := c.repo.GetProductVariants(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve product variants"})
	}
	return ctx.JSON(productVariants)
}

func (c *ProductVariantController) UpdateProductVariant(ctx *fiber.Ctx) error {
	var productVariant models.ProductVariant
	if err := ctx.BodyParser(&productVariant); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	if err := c.repo.UpdateProductVariant(ctx.Context(), productVariant); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update product variant"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Product variant updated successfully"})
}

func (c *ProductVariantController) DeleteProductVariant(ctx *fiber.Ctx) error {
	variantID := ctx.Params("id")
	if err := c.repo.DeleteProductVariant(ctx.Context(), variantID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete product variant"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Product variant deleted successfully"})
}
