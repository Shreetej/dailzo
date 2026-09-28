package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type FoodProductController struct {
	repo  *repository.FoodProductRepository
	trans *repository.TranslationRepository
}

func NewFoodProductController(repo *repository.FoodProductRepository, trans *repository.TranslationRepository) *FoodProductController {
	return &FoodProductController{repo: repo, trans: trans}
}

func (c *FoodProductController) CreateFoodProduct(ctx *fiber.Ctx) error {
	var foodProduct models.FoodProduct
	if err := ctx.BodyParser(&foodProduct); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	// Admins can create/manage products on any vendor's behalf; everyone else
	// must own the target restaurant/outlet AND have cleared admin approval.
	// Previously there was no check at all here, so a vendor whose
	// registration was never approved (or was rejected) could still list
	// products and receive orders.
	userID, _ := ctx.Locals("user_id").(string)
	userType, _ := ctx.Locals("user_type").(string)
	if userType != "admin" {
		ok, reason, err := c.repo.VerifyVendorActive(ctx.Context(), userID, foodProduct.RestaurantId, foodProduct.OutletID)
		if err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not verify vendor status"})
		}
		if !ok {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": reason})
		}
	}

	id, err := c.repo.CreateFoodProduct(ctx.Context(), foodProduct)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create food product"})
	}
	// Auto-translate the vendor-entered name/description on write so the product
	// renders localized without a re-release; an admin/vendor can override later.
	c.trans.AutoTranslate(ctx.Context(), "product", id, "name", foodProduct.Name)
	c.trans.AutoTranslate(ctx.Context(), "product", id, "description", foodProduct.Description)
	log := config.SetupLogger()
	log.Info().Msgf("Food product created with ID: %s", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *FoodProductController) GetFoodProductWithEntity(ctx *fiber.Ctx) error {
	foodProductEntity := ctx.Params("entity")
	println("entity  :", foodProductEntity)

	foodProducts, err := c.repo.GetFoodProductWithEntity(ctx, foodProductEntity)
	if err != nil {
		fmt.Println("err : 39 ", err.Error())

		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	fmt.Println("foodProducts :", foodProducts[0].RestaurantId)
	return ctx.JSON(foodProducts)
}

func (c *FoodProductController) GetFoodProductById(ctx *fiber.Ctx) error {
	foodProductID := ctx.Params("id")
	foodProductEntity := ctx.Params("entity")
	println("entity  :", foodProductEntity)
	println("foodProductID  :", foodProductID)

	foodProduct, err := c.repo.GetFoodProductByID(ctx.Context(), foodProductID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Food product not found"})
	}
	return ctx.JSON(foodProduct)
}

func (c *FoodProductController) GetFoodProducts(ctx *fiber.Ctx) error {
	foodProducts, err := c.repo.GetFoodProducts(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return ctx.JSON(foodProducts)
}

func (c *FoodProductController) UpdateFoodProduct(ctx *fiber.Ctx) error {
	var foodProduct models.FoodProduct
	if err := ctx.BodyParser(&foodProduct); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	if err := c.repo.UpdateFoodProduct(ctx.Context(), foodProduct); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update food product"})
	}
	// Refresh auto-translations (manual overrides are preserved by AutoTranslate).
	c.trans.AutoTranslate(ctx.Context(), "product", foodProduct.ID, "name", foodProduct.Name)
	c.trans.AutoTranslate(ctx.Context(), "product", foodProduct.ID, "description", foodProduct.Description)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Food product updated successfully"})
}

func (c *FoodProductController) DeleteFoodProduct(ctx *fiber.Ctx) error {
	foodProductID := ctx.Params("id")
	if err := c.repo.DeleteFoodProduct(ctx.Context(), foodProductID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete food product"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Food product deleted successfully"})
}
