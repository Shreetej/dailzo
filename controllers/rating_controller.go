package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type RatingController struct {
	repo *repository.RatingRepository
}

func NewRatingController(repo *repository.RatingRepository) *RatingController {
	return &RatingController{repo: repo}
}

func (c *RatingController) CreateRating(ctx *fiber.Ctx) error {
	var rating models.Rating
	if err := ctx.BodyParser(&rating); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreateRating(ctx.Context(), rating)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create rating"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Rating created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// func (c *RatingController) GetRating(ctx *fiber.Ctx) error {
// 	ratingID := ctx.Params("id")
// 	rating, err := c.repo.GetRatingByID(ctx.Context(), ratingID)
// 	if err != nil {
// 		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Rating not found"})
// 	}
// 	return ctx.JSON(rating)
// }

// func (c *RatingController) GetRatings(ctx *fiber.Ctx) error {
// 	ratings, err := c.repo.GetRatings(ctx.Context())
// 	if err != nil {
// 		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No ratings found"})
// 	}
// 	return ctx.JSON(ratings)
// }

// func (c *RatingController) UpdateRating(ctx *fiber.Ctx) error {
// 	var rating models.Rating
// 	if err := ctx.BodyParser(&rating); err != nil {
// 		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
// 	}

// 	if err := c.repo.UpdateRating(ctx.Context(), rating); err != nil {
// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update rating"})
// 	}

// 	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Rating updated successfully"})
// }

// func (c *RatingController) DeleteRating(ctx *fiber.Ctx) error {
// 	ratingID := ctx.Params("id")
// 	if err := c.repo.DeleteRating(ctx.Context(), ratingID); err != nil {
// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete rating"})
// 	}

// 	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Rating deleted successfully"})
// }
