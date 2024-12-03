package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type RestaurantController struct {
	repo *repository.RestaurantRepository
}

func NewRestaurantController(repo *repository.RestaurantRepository) *RestaurantController {
	return &RestaurantController{repo: repo}
}

func (c *RestaurantController) CreateRestaurant(ctx *fiber.Ctx) error {
	var restaurant models.Restaurant
	if err := ctx.BodyParser(&restaurant); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreateRestaurant(ctx.Context(), restaurant)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create restaurant"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Restaurant created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// func (c *RestaurantController) GetRestaurant(ctx *fiber.Ctx) error {
// 	restaurantID := ctx.Params("id")
// 	restaurant, err := c.repo.GetRestaurantByID(ctx.Context(), restaurantID)
// 	if err != nil {
// 		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Restaurant not found"})
// 	}
// 	return ctx.JSON(restaurant)
// }

// func (c *RestaurantController) GetRestaurants(ctx *fiber.Ctx) error {
// 	restaurants, err := c.repo.GetRestaurants(ctx.Context())
// 	if err != nil {
// 		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No restaurants found"})
// 	}
// 	return ctx.JSON(restaurants)
// }

// func (c *RestaurantController) UpdateRestaurant(ctx *fiber.Ctx) error {
// 	var restaurant models.Restaurant
// 	if err := ctx.BodyParser(&restaurant); err != nil {
// 		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
// 	}

// 	if err := c.repo.UpdateRestaurant(ctx.Context(), restaurant); err != nil {
// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update restaurant"})
// 	}

// 	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Restaurant updated successfully"})
// }

// func (c *RestaurantController) DeleteRestaurant(ctx *fiber.Ctx) error {
// 	restaurantID := ctx.Params("id")
// 	if err := c.repo.DeleteRestaurant(ctx.Context(), restaurantID); err != nil {
// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete restaurant"})
// 	}

// 	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Restaurant deleted successfully"})
// }
