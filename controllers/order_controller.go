package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type OrderController struct {
	repo *repository.OrderRepository
}

func NewOrderController(repo *repository.OrderRepository) *OrderController {
	return &OrderController{repo: repo}
}

func (c *OrderController) CreateOrder(ctx *fiber.Ctx) error {
	var order models.Order
	if err := ctx.BodyParser(&order); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreateOrder(ctx.Context(), order)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create order"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Order created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *OrderController) GetOrder(ctx *fiber.Ctx) error {
	orderID := ctx.Params("id")
	id := orderID
	if orderID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid order ID",
		})
	}
	order, err := c.repo.GetOrderByID(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Order not found"})
	}

	return ctx.JSON(order)
}

func (c *OrderController) GetOrders(ctx *fiber.Ctx) error {
	orders, err := c.repo.GetOrders(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No orders found"})
	}

	return ctx.JSON(orders)
}

func (c *OrderController) UpdateOrder(ctx *fiber.Ctx) error {
	var order models.Order

	// Parse request body
	if err := ctx.BodyParser(&order); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Update order in the database
	if err := c.repo.UpdateOrder(ctx.Context(), order); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update order",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Order updated successfully",
	})
}

func (c *OrderController) DeleteOrder(ctx *fiber.Ctx) error {
	orderID := ctx.Params("id")
	if orderID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid order ID",
		})
	}

	// Delete order from the database
	if err := c.repo.DeleteOrder(ctx.Context(), orderID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete order",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Order deleted successfully",
	})
}
