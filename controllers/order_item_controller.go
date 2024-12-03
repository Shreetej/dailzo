package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type OrderItemController struct {
	repo *repository.OrderItemRepository
}

func NewOrderItemController(repo *repository.OrderItemRepository) *OrderItemController {
	return &OrderItemController{repo: repo}
}

func (c *OrderItemController) CreateOrderItem(ctx *fiber.Ctx) error {
	var orderItem models.OrderItem
	if err := ctx.BodyParser(&orderItem); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreateOrderItem(ctx.Context(), orderItem)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create order item"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Order item created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *OrderItemController) GetOrderItem(ctx *fiber.Ctx) error {
	orderItemID := ctx.Params("id")
	orderItem, err := c.repo.GetOrderItemByID(ctx.Context(), orderItemID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Order item not found"})
	}
	return ctx.JSON(orderItem)
}

func (c *OrderItemController) GetOrderItems(ctx *fiber.Ctx) error {
	orderItems, err := c.repo.GetOrderItems(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No order items found"})
	}
	return ctx.JSON(orderItems)
}

func (c *OrderItemController) UpdateOrderItem(ctx *fiber.Ctx) error {
	var orderItem models.OrderItem
	if err := ctx.BodyParser(&orderItem); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	if err := c.repo.UpdateOrderItem(ctx.Context(), orderItem); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update order item"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Order item updated successfully"})
}

func (c *OrderItemController) DeleteOrderItem(ctx *fiber.Ctx) error {
	orderItemID := ctx.Params("id")
	if err := c.repo.DeleteOrderItem(ctx.Context(), orderItemID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete order item"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Order item deleted successfully"})
}
