package server

import (
	"dailzo/internal/api"
	"dailzo/models"
	"dailzo/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// GetOrders handles GET /orders
func (s *Server) GetOrders(c *fiber.Ctx, params api.GetOrdersParams) error {
	orders, err := s.orderRepo.GetOrdersWithFilters(c.Context(), params.Status, params.OutletId)
	if err != nil {
		return response.InternalError(c, "Failed to fetch orders")
	}

	if orders == nil {
		orders = []models.Order{}
	}

	return response.Success(c, orders)
}

// PatchOrdersOrderIdStatus handles PATCH /orders/{order_id}/status
func (s *Server) PatchOrdersOrderIdStatus(c *fiber.Ctx, orderId string) error {
	var req map[string]interface{}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	status, ok := req["status"].(string)
	if !ok || status == "" {
		return response.BadRequest(c, "Status is required")
	}

	// Validate status values
	validStatuses := map[string]bool{
		"pending":    true,
		"confirmed":  true,
		"preparing":  true,
		"ready":      true,
		"picked_up":  true,
		"in_transit": true,
		"delivered":  true,
		"cancelled":  true,
	}

	if !validStatuses[status] {
		return response.BadRequest(c, "Invalid status value")
	}

	err := s.orderRepo.UpdateOrderStatus(c.Context(), orderId, status)
	if err != nil {
		if err.Error() == "order not found" {
			return response.NotFound(c, "Order not found")
		}
		return response.InternalError(c, "Failed to update order status")
	}

	return response.Success(c, fiber.Map{
		"message": "Order status updated successfully",
		"status":  status,
	})
}

// PostOrdersOrderIdAssignDelivery handles POST /orders/{order_id}/assign-delivery
func (s *Server) PostOrdersOrderIdAssignDelivery(c *fiber.Ctx, orderId string) error {
	var req api.PostOrdersOrderIdAssignDeliveryJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.DeliveryPersonId == nil || *req.DeliveryPersonId == "" {
		return response.BadRequest(c, "Delivery person ID is required")
	}

	// Verify delivery person exists
	_, err := s.deliveryRepo.GetProfileByID(c.Context(), *req.DeliveryPersonId)
	if err != nil {
		return response.BadRequest(c, "Delivery person not found")
	}

	err = s.orderRepo.AssignDeliveryPerson(c.Context(), orderId, *req.DeliveryPersonId)
	if err != nil {
		if err.Error() == "order not found" {
			return response.NotFound(c, "Order not found")
		}
		return response.InternalError(c, "Failed to assign delivery person")
	}

	return response.Success(c, fiber.Map{
		"message":            "Delivery person assigned successfully",
		"delivery_person_id": *req.DeliveryPersonId,
	})
}
