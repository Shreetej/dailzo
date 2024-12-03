package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type RefundController struct {
	repo *repository.RefundRepository
}

func NewRefundController(repo *repository.RefundRepository) *RefundController {
	return &RefundController{repo: repo}
}

func (c *RefundController) CreateRefund(ctx *fiber.Ctx) error {
	var refund models.Refund
	if err := ctx.BodyParser(&refund); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreateRefund(ctx.Context(), refund)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create refund"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Refund created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// func (c *RefundController) GetRefund(ctx *fiber.Ctx) error {
// 	refundID := ctx.Params("id")
// 	refund, err := c.repo.GetRefundByID(ctx.Context(), refundID)
// 	if err != nil {
// 		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Refund not found"})
// 	}
// 	return ctx.JSON(refund)
// }

// func (c *RefundController) GetRefunds(ctx *fiber.Ctx) error {
// 	refunds, err := c.repo.GetRefunds(ctx.Context())
// 	if err != nil {
// 		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No refunds found"})
// 	}
// 	return ctx.JSON(refunds)
// }

// func (c *RefundController) UpdateRefund(ctx *fiber.Ctx) error {
// 	var refund models.Refund
// 	if err := ctx.BodyParser(&refund); err != nil {
// 		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
// 	}

// 	if err := c.repo.UpdateRefund(ctx.Context(), refund); err != nil {
// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update refund"})
// 	}

// 	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Refund updated successfully"})
// }

// func (c *RefundController) DeleteRefund(ctx *fiber.Ctx) error {
// 	refundID := ctx.Params("id")
// 	if err := c.repo.DeleteRefund(ctx.Context(), refundID); err != nil {
// 		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete refund"})
// 	}

// 	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Refund deleted successfully"})
// }
