package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type PaymentController struct {
	repo *repository.PaymentRepository
}

func NewPaymentController(repo *repository.PaymentRepository) *PaymentController {
	return &PaymentController{repo: repo}
}

func (c *PaymentController) CreatePayment(ctx *fiber.Ctx) error {
	var payment models.Payment
	if err := ctx.BodyParser(&payment); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreatePayment(ctx.Context(), payment)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create payment"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Payment created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *PaymentController) GetPayment(ctx *fiber.Ctx) error {
	paymentID := ctx.Params("id")
	payment, err := c.repo.GetPaymentByID(ctx.Context(), paymentID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Payment not found"})
	}
	return ctx.JSON(payment)
}

func (c *PaymentController) GetPayments(ctx *fiber.Ctx) error {
	payments, err := c.repo.GetPayments(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No payments found"})
	}
	return ctx.JSON(payments)
}

func (c *PaymentController) UpdatePayment(ctx *fiber.Ctx) error {
	var payment models.Payment
	if err := ctx.BodyParser(&payment); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	if err := c.repo.UpdatePayment(ctx.Context(), payment); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update payment"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Payment updated successfully"})
}

func (c *PaymentController) DeletePayment(ctx *fiber.Ctx) error {
	paymentID := ctx.Params("id")
	if err := c.repo.DeletePayment(ctx.Context(), paymentID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete payment"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Payment deleted successfully"})
}
