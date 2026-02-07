package server

import (
	"dailzo/internal/api"
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
