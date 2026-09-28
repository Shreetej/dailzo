package server

import (
	"dailzo/internal/api"
	ws "dailzo/internal/websocket"
	"dailzo/models"
	"dailzo/pkg/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

// PostDeliveryOnboarding handles POST /delivery/onboarding
func (s *Server) PostDeliveryOnboarding(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	var req map[string]interface{}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	profile := &models.DeliveryProfileCreate{
		UserID: userID,
	}

	// Extract fields from request
	if name, ok := req["name"].(string); ok {
		profile.Name = name
	}
	if phone, ok := req["phone"].(string); ok {
		profile.Phone = phone
	}
	if city, ok := req["city"].(string); ok {
		profile.City = city
	}
	if vehicleType, ok := req["vehicle_type"].(string); ok {
		profile.VehicleType = vehicleType
	}
	if vehicleNumber, ok := req["vehicle_number"].(string); ok {
		profile.VehicleNumber = vehicleNumber
	}
	if licenseNumber, ok := req["license_number"].(string); ok {
		profile.LicenseNumber = licenseNumber
	}

	// Validate required fields
	if profile.Name == "" || profile.Phone == "" {
		return response.BadRequest(c, "Name and phone are required")
	}

	id, err := s.deliveryRepo.CreateProfile(c.Context(), profile)
	if err != nil {
		return response.InternalError(c, "Failed to create delivery profile")
	}

	return response.Created(c, fiber.Map{
		"id":      id,
		"message": "Delivery profile created successfully",
	})
}

// GetDeliveryProfile handles GET /delivery/profile
func (s *Server) GetDeliveryProfile(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	profile, err := s.deliveryRepo.GetProfileByUserID(c.Context(), userID)
	if err != nil {
		return response.NotFound(c, "Delivery profile not found")
	}

	return response.Success(c, profile)
}

// GetDeliveryKpis handles GET /delivery/kpis
func (s *Server) GetDeliveryKpis(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	kpis, err := s.deliveryRepo.GetDailyKpis(c.Context(), userID, time.Now())
	if err != nil {
		return response.InternalError(c, "Failed to fetch KPIs")
	}

	return response.Success(c, kpis)
}

// GetDeliveryAvailable handles GET /delivery/available — the real job queue:
// ready orders with no partner assigned yet. Replaces the vendor app's
// earlier side door of reusing the unscoped /vendor/orders list.
func (s *Server) GetDeliveryAvailable(c *fiber.Ctx) error {
	tasks, err := s.deliveryRepo.GetAvailableOrders(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to fetch available orders")
	}
	if tasks == nil {
		tasks = []models.DeliveryTask{}
	}
	return response.Success(c, tasks)
}

// PostDeliveryAvailableClaim handles POST /delivery/available/:orderId/claim.
// The caller must have completed delivery onboarding (have a DeliveryProfile);
// the claim itself is an atomic conditional update, so two partners racing
// for the same order can't both win it. Registered manually in main.go (not
// through the OpenAPI codegen), so this takes the plain fiber.Handler shape
// and reads the path param itself rather than via a generated wrapper.
func (s *Server) PostDeliveryAvailableClaim(c *fiber.Ctx) error {
	orderId := c.Params("orderId")
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	// Confirms onboarding is complete; orders.delivery_person_id itself is a
	// users(id) FK (see AssignDeliveryPerson / GetActiveTask), not the
	// delivery_profiles row id, so ClaimOrder takes userID, not profile.ID.
	profile, err := s.deliveryRepo.GetProfileByUserID(c.Context(), userID)
	if err != nil {
		return response.BadRequest(c, "Complete delivery onboarding before accepting jobs")
	}
	// is_active is the real approval gate (set by the rider's fleet, or by
	// platform admin for a direct-to-platform rider) — distinct from
	// is_online, which only means "currently on shift".
	if !profile.IsActive {
		return response.Forbidden(c, "Your delivery registration is still pending approval")
	}

	claimed, err := s.deliveryRepo.ClaimOrder(c.Context(), orderId, userID)
	if err != nil {
		return response.InternalError(c, "Failed to claim order")
	}
	if !claimed {
		return response.Conflict(c, "This order was already claimed by another partner")
	}

	return response.Success(c, fiber.Map{
		"message":  "Order claimed",
		"order_id": orderId,
	})
}

// PostDeliveryLocation handles POST /delivery/location — a periodic ping from
// the delivery partner's app while an order is out for delivery. Persists the
// position (deliveryRepo.UpdateLocation existed but had no caller anywhere)
// and pushes it live to whoever is connected to
// GET /ws/delivery/track/:orderId for that order (hub.BroadcastLocation also
// existed but was never called — a customer connecting got one handshake
// message and then silence forever).
func (s *Server) PostDeliveryLocation(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	var req struct {
		OrderID string  `json:"order_id"`
		Lat     float64 `json:"lat"`
		Lng     float64 `json:"lng"`
		Status  string  `json:"status"`
		ETAMins int     `json:"eta_mins"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	if req.OrderID == "" {
		return response.BadRequest(c, "order_id is required")
	}

	if err := s.deliveryRepo.UpdateLocation(c.Context(), userID, req.Lat, req.Lng); err != nil {
		return response.InternalError(c, "Failed to update location")
	}

	if s.hub != nil {
		s.hub.BroadcastLocation(&ws.LocationUpdate{
			OrderID:   req.OrderID,
			Lat:       req.Lat,
			Lng:       req.Lng,
			Status:    req.Status,
			ETAMins:   req.ETAMins,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}

	return response.Success(c, fiber.Map{"message": "Location updated"})
}

// GetDeliveryActiveTask handles GET /delivery/active-task
func (s *Server) GetDeliveryActiveTask(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	task, err := s.deliveryRepo.GetActiveTask(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch active task")
	}

	if task == nil {
		return response.Success(c, fiber.Map{
			"message": "No active task",
			"task":    nil,
		})
	}

	return response.Success(c, task)
}

// GetDeliveryTraceOrderId handles GET /delivery/trace/{orderId}
func (s *Server) GetDeliveryTraceOrderId(c *fiber.Ctx, orderId string) error {
	events, err := s.deliveryRepo.GetTraceEvents(c.Context(), orderId)
	if err != nil {
		return response.InternalError(c, "Failed to fetch trace events")
	}

	return response.Success(c, events)
}

// GetDeliverySlaKpis handles GET /delivery/sla-kpis
func (s *Server) GetDeliverySlaKpis(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	kpis, err := s.deliveryRepo.GetSlaKpis(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch SLA KPIs")
	}

	return response.Success(c, kpis)
}

// GetDeliveryEarningsInsights handles GET /delivery/earnings-insights
func (s *Server) GetDeliveryEarningsInsights(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	insights, err := s.deliveryRepo.GetEarningsInsights(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch earnings insights")
	}

	return response.Success(c, insights)
}

// GetDeliveryRecommendations handles GET /delivery/recommendations
func (s *Server) GetDeliveryRecommendations(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	recommendations, err := s.deliveryRepo.GetRecommendations(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch recommendations")
	}

	if recommendations == nil {
		recommendations = []models.DeliveryRecommendation{}
	}

	return response.Success(c, recommendations)
}

// PostDeliveryRecommendationsAck handles POST /delivery/recommendations/ack
func (s *Server) PostDeliveryRecommendationsAck(c *fiber.Ctx) error {
	var req api.PostDeliveryRecommendationsAckJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if req.Id == nil || *req.Id == "" {
		return response.BadRequest(c, "Recommendation ID is required")
	}

	err := s.deliveryRepo.AckRecommendation(c.Context(), *req.Id)
	if err != nil {
		return response.InternalError(c, "Failed to acknowledge recommendation")
	}

	return response.Success(c, fiber.Map{
		"message": "Recommendation acknowledged",
	})
}

// GetDeliveryEarningsWeekly handles GET /delivery/earnings/weekly
func (s *Server) GetDeliveryEarningsWeekly(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	earnings, err := s.deliveryRepo.GetWeeklyEarnings(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch weekly earnings")
	}

	return response.Success(c, earnings)
}

// GetDeliveryShifts handles GET /delivery/shifts
func (s *Server) GetDeliveryShifts(c *fiber.Ctx) error {
	shifts, err := s.deliveryRepo.GetShifts(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to fetch shifts")
	}

	if shifts == nil {
		shifts = []models.DeliveryShift{}
	}

	return response.Success(c, shifts)
}
