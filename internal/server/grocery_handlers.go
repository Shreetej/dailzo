package server

import (
	"dailzo/models"
	"dailzo/pkg/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

// PostGroceryOnboarding handles POST /grocery/onboarding
func (s *Server) PostGroceryOnboarding(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	var req map[string]interface{}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	profile := &models.GroceryProfileCreate{
		UserID: userID,
	}

	// Extract fields from request
	if storeName, ok := req["store_name"].(string); ok {
		profile.StoreName = storeName
	}
	if ownerName, ok := req["owner_name"].(string); ok {
		profile.OwnerName = ownerName
	}
	if email, ok := req["email"].(string); ok {
		profile.Email = email
	}
	if phone, ok := req["phone"].(string); ok {
		profile.Phone = phone
	}
	if address, ok := req["address"].(string); ok {
		profile.Address = address
	}
	if city, ok := req["city"].(string); ok {
		profile.City = city
	}
	if pincode, ok := req["pincode"].(string); ok {
		profile.Pincode = pincode
	}
	if fssaiLicense, ok := req["fssai_license"].(string); ok {
		profile.FSSAILicense = fssaiLicense
	}
	if gstNumber, ok := req["gst_number"].(string); ok {
		profile.GSTNumber = gstNumber
	}
	if panNumber, ok := req["pan_number"].(string); ok {
		profile.PANNumber = panNumber
	}
	if workingHours, ok := req["working_hours"].(string); ok {
		profile.WorkingHours = workingHours
	}

	// Validate required fields
	if profile.StoreName == "" || profile.Phone == "" {
		return response.BadRequest(c, "Store name and phone are required")
	}

	id, err := s.groceryRepo.CreateProfile(c.Context(), profile)
	if err != nil {
		return response.InternalError(c, "Failed to create grocery profile")
	}

	return response.Created(c, fiber.Map{
		"id":      id,
		"message": "Grocery profile created successfully",
	})
}

// GetGroceryProfile handles GET /grocery/profile
func (s *Server) GetGroceryProfile(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	profile, err := s.groceryRepo.GetProfileByUserID(c.Context(), userID)
	if err != nil {
		return response.NotFound(c, "Grocery profile not found")
	}

	return response.Success(c, profile)
}

// GetGroceryKpis handles GET /grocery/kpis
func (s *Server) GetGroceryKpis(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	kpis, err := s.groceryRepo.GetDailyKpis(c.Context(), userID, time.Now())
	if err != nil {
		return response.InternalError(c, "Failed to fetch KPIs")
	}

	return response.Success(c, kpis)
}

// GetGroceryExpiryAlerts handles GET /grocery/expiry-alerts
func (s *Server) GetGroceryExpiryAlerts(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	alerts, err := s.groceryRepo.GetExpiryAlerts(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch expiry alerts")
	}

	if alerts == nil {
		alerts = []models.GroceryExpiryAlert{}
	}

	return response.Success(c, alerts)
}

// GetGroceryStockAlerts handles GET /grocery/stock-alerts
func (s *Server) GetGroceryStockAlerts(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	alerts, err := s.groceryRepo.GetStockAlerts(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch stock alerts")
	}

	if alerts == nil {
		alerts = []models.GroceryStockAlert{}
	}

	return response.Success(c, alerts)
}

// GetGroceryPayoutSummary handles GET /grocery/payout-summary
func (s *Server) GetGroceryPayoutSummary(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	summary, err := s.groceryRepo.GetPayoutSummary(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch payout summary")
	}

	return response.Success(c, summary)
}
