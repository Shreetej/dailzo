package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type PaymentMethodController struct {
	repo *repository.PaymentMethodRepository
}

func NewPaymentMethodController(repo *repository.PaymentMethodRepository) *PaymentMethodController {
	return &PaymentMethodController{repo: repo}
}

func (c *PaymentMethodController) CreatePaymentMethod(ctx *fiber.Ctx) error {
	var paymentMethod models.PaymentMethod
	if err := ctx.BodyParser(&paymentMethod); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreatePaymentMethod(ctx.Context(), paymentMethod)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create payment method"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Payment method created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *PaymentMethodController) GetPaymentMethod(ctx *fiber.Ctx) error {
	paymentMethodID := ctx.Params("id")
	paymentMethod, err := c.repo.GetPaymentMethodByID(ctx.Context(), paymentMethodID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Payment method not found"})
	}
	return ctx.JSON(paymentMethod)
}

func (c *PaymentMethodController) GetPaymentMethods(ctx *fiber.Ctx) error {
	paymentMethods, err := c.repo.GetPaymentMethods(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No payment methods found"})
	}
	return ctx.JSON(paymentMethods)
}

func (c *PaymentMethodController) UpdatePaymentMethod(ctx *fiber.Ctx) error {
	var paymentMethod models.PaymentMethod
	if err := ctx.BodyParser(&paymentMethod); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	if err := c.repo.UpdatePaymentMethod(ctx.Context(), paymentMethod); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update payment method"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Payment method updated successfully"})
}

func (c *PaymentMethodController) DeletePaymentMethod(ctx *fiber.Ctx) error {
	paymentMethodID := ctx.Params("id")
	if err := c.repo.DeletePaymentMethod(ctx.Context(), paymentMethodID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete payment method"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Payment method deleted successfully"})
}
