package controllers

import (
	"dailzo/config"
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
	id, err := c.repo.CreateFoodProduct(ctx.Context(), foodProduct)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create food product"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Food product created with ID: %d", id)
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

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Food product updated successfully"})
}

func (c *FoodProductController) DeleteFoodProduct(ctx *fiber.Ctx) error {
	foodProductID := ctx.Params("id")
	if err := c.repo.DeleteFoodProduct(ctx.Context(), foodProductID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete food product"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Food product deleted successfully"})
}
