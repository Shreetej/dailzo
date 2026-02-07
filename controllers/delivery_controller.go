package controllers

import (
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// DeliveryController handles delivery-related API requests
type DeliveryController struct {
	repo *repository.DeliveryRepository
}

// NewDeliveryController creates a new DeliveryController instance
func NewDeliveryController(repo *repository.DeliveryRepository) *DeliveryController {
	return &DeliveryController{repo: repo}
}

// Profile retrieves delivery partner profile
func (c *DeliveryController) Profile(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing delivery id"})
	}
	profile, err := c.repo.GetProfileByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "delivery profile not found"})
	}
	return ctx.JSON(profile)
}

// Trace retrieves delivery task trace by order ID
func (c *DeliveryController) Trace(ctx *fiber.Ctx) error {
	orderId := ctx.Params("orderId")
	if orderId == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing order ID"})
	}
	trace, err := c.repo.GetTraceEvents(ctx.Context(), orderId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "delivery trace not found"})
	}
	return ctx.JSON(trace)
}
