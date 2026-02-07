package server

import (
	"dailzo/internal/api"
	"dailzo/models"
	"dailzo/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// GetProducts handles GET /products
func (s *Server) GetProducts(c *fiber.Ctx, params api.GetProductsParams) error {
	// Convert params to filter values
	var category, stock *string
	var promo *bool
	var expiry, role *string

	if params.Category != nil {
		category = params.Category
	}
	if params.Stock != nil {
		stock = params.Stock
	}
	if params.Promo != nil {
		promo = params.Promo
	}
	if params.Expiry != nil {
		e := string(*params.Expiry)
		expiry = &e
	}
	if params.Role != nil {
		r := string(*params.Role)
		role = &r
	}

	products, err := s.productRepo.GetProductsWithFilters(c.Context(), category, stock, promo, expiry, role)
	if err != nil {
		return response.InternalError(c, "Failed to fetch products")
	}

	if products == nil {
		products = []models.FoodProduct{}
	}

	return response.Success(c, products)
}

// GetProductsExpiryAlerts handles GET /products/expiry-alerts
func (s *Server) GetProductsExpiryAlerts(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	// Get outlet ID if user has a grocery profile
	outletID := ""
	if userID != "" {
		profile, err := s.groceryRepo.GetProfileByUserID(c.Context(), userID)
		if err == nil && profile != nil {
			outletID = profile.ID
		}
	}

	products, err := s.productRepo.GetExpiryAlerts(c.Context(), outletID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch expiry alerts")
	}

	if products == nil {
		products = []models.FoodProduct{}
	}

	return response.Success(c, products)
}

// GetProductsAutoDiscounts handles GET /products/auto-discounts
func (s *Server) GetProductsAutoDiscounts(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	// Get outlet ID if user has a grocery profile
	outletID := ""
	if userID != "" {
		profile, err := s.groceryRepo.GetProfileByUserID(c.Context(), userID)
		if err == nil && profile != nil {
			outletID = profile.ID
		}
	}

	products, err := s.productRepo.GetAutoDiscounts(c.Context(), outletID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch auto discounts")
	}

	if products == nil {
		products = []models.FoodProduct{}
	}

	return response.Success(c, products)
}

// PatchProductsId handles PATCH /products/{id}
func (s *Server) PatchProductsId(c *fiber.Ctx, id string) error {
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if len(updates) == 0 {
		return response.BadRequest(c, "No updates provided")
	}

	// Validate and sanitize updates - only allow certain fields
	allowedFields := map[string]bool{
		"name":              true,
		"description":       true,
		"price":             true,
		"category":          true,
		"image_url":         true,
		"is_active":         true,
		"stock_quantity":    true,
		"expiry_date":       true,
		"is_promo":          true,
		"promo_price":       true,
		"auto_discount_pct": true,
	}

	sanitizedUpdates := make(map[string]interface{})
	for key, value := range updates {
		if allowedFields[key] {
			sanitizedUpdates[key] = value
		}
	}

	if len(sanitizedUpdates) == 0 {
		return response.BadRequest(c, "No valid fields to update")
	}

	err := s.productRepo.PatchProduct(c.Context(), id, sanitizedUpdates)
	if err != nil {
		if err.Error() == "product not found" {
			return response.NotFound(c, "Product not found")
		}
		return response.InternalError(c, "Failed to update product")
	}

	return response.Success(c, fiber.Map{
		"message": "Product updated successfully",
	})
}
