package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type ConsentController struct {
	repo *repository.ConsentRepository
}

func NewConsentController(repo *repository.ConsentRepository) *ConsentController {
	return &ConsentController{repo: repo}
}

// 🔥 **CreateConsent**
func (c *ConsentController) CreateConsent(ctx *fiber.Ctx, consent models.Consent) error {
	//var consent models.Consent
	if err := ctx.BodyParser(&consent); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	// Create the consent record
	id, err := c.repo.CreateConsent(ctx.Context(), consent)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create consent"})
	}

	// Log the creation
	log := config.SetupLogger()
	log.Info().Msgf("Consent created with ID: %s", id)

	// Respond with success
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// 🔥 **GetConsentByID**
func (c *ConsentController) GetConsentByID(ctx *fiber.Ctx) error {
	consentID := ctx.Params("id")
	consent, err := c.repo.GetConsentByID(ctx.Context(), consentID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Consent not found"})
	}
	return ctx.JSON(consent)
}

// 🔥 **GetConsentByID**
func (c *ConsentController) VerifyOTP(ctx *fiber.Ctx, entityToVerify string, otpEntered string) error {
	consent, err := c.repo.VerifyOTP(ctx.Context(), entityToVerify, otpEntered)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Consent not found"})
	}
	return ctx.JSON(consent)
}

// 🔥 **GetConsents**
func (c *ConsentController) GetConsents(ctx *fiber.Ctx) error {
	consents, err := c.repo.GetConsents(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No consents found"})
	}
	return ctx.JSON(consents)
}

// 🔥 **UpdateConsent**
func (c *ConsentController) UpdateConsent(ctx *fiber.Ctx) error {
	var consent models.Consent
	if err := ctx.BodyParser(&consent); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	if err := c.repo.UpdateConsent(ctx.Context(), consent); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update consent"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Consent updated successfully"})
}

// 🔥 **DeleteConsent**
func (c *ConsentController) DeleteConsent(ctx *fiber.Ctx) error {
	consentID := ctx.Params("id")
	if err := c.repo.DeleteConsent(ctx.Context(), consentID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete consent"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Consent deleted successfully"})
}
