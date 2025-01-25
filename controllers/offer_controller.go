package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type OfferController struct {
	repo *repository.OfferRepository
}

func NewOfferController(repo *repository.OfferRepository) *OfferController {
	return &OfferController{repo: repo}
}

// Create a new offer
func (c *OfferController) CreateOffer(ctx *fiber.Ctx) error {
	var offer models.Offer
	if err := ctx.BodyParser(&offer); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	id, err := c.repo.CreateOffer(ctx.Context(), offer)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create offer"})
	}

	log := config.SetupLogger()
	log.Info().Msgf("Offer created with ID: %s", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// Retrieve all offers
func (c *OfferController) GetOffers(ctx *fiber.Ctx) error {
	offers, err := c.repo.GetOffers(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no offers found"})
	}
	return ctx.JSON(offers)
}

// Update an existing offer
func (c *OfferController) UpdateOffer(ctx *fiber.Ctx) error {
	var offer models.Offer
	if err := ctx.BodyParser(&offer); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	if err := c.repo.UpdateOffer(ctx.Context(), offer); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update offer"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Offer updated successfully"})
}

// Delete an offer by ID
func (c *OfferController) DeleteOffer(ctx *fiber.Ctx) error {
	offerID := ctx.Params("id")
	if err := c.repo.DeleteOffer(ctx.Context(), offerID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete offer"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Offer deleted successfully"})
}

// Create a new condition
func (c *OfferController) CreateCondition(ctx *fiber.Ctx) error {
	var condition models.Condition
	if err := ctx.BodyParser(&condition); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	id, err := c.repo.CreateCondition(ctx.Context(), condition)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create condition"})
	}

	log := config.SetupLogger()
	log.Info().Msgf("Condition created with ID: %s", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// Retrieve conditions for a specific offer
func (c *OfferController) GetConditionsByOfferID(ctx *fiber.Ctx) error {
	offerID := ctx.Params("offer_id")
	conditions, err := c.repo.GetConditionsByOfferID(ctx.Context(), offerID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no conditions found for the offer"})
	}

	return ctx.JSON(conditions)
}

// Create a new applicable entity
func (c *OfferController) CreateApplicableEntity(ctx *fiber.Ctx) error {
	var entity models.ApplicableEntity
	if err := ctx.BodyParser(&entity); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	id, err := c.repo.CreateApplicableEntity(ctx.Context(), entity)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create applicable entity"})
	}

	log := config.SetupLogger()
	log.Info().Msgf("Applicable entity created with ID: %s", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// Retrieve applicable entities for a specific offer
func (c *OfferController) GetEntitiesByOfferID(ctx *fiber.Ctx) error {
	offerID := ctx.Params("offer_id")
	entities, err := c.repo.GetEntitiesByOfferID(ctx.Context(), offerID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no entities found for the offer"})
	}

	return ctx.JSON(entities)
}
