package controllers

import (
	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// RegistrationController serves the Partner app's restaurant registration
// flow. The app sends and expects the full registration JSON document
// (RestaurantRegistrationData) without a response envelope, so handlers echo
// raw JSON rather than the ApiResponse wrapper.
type RegistrationController struct {
	repo *repository.RegistrationRepository
}

func NewRegistrationController(repo *repository.RegistrationRepository) *RegistrationController {
	return &RegistrationController{repo: repo}
}

// POST /restaurant/register
func (c *RegistrationController) RegisterRestaurant(ctx *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := ctx.BodyParser(&payload); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}

	saved, err := c.repo.CreateRegistration(ctx.Context(), payload)
	if err != nil {
		return response.InternalError(ctx, "could not register restaurant")
	}
	return ctx.Status(fiber.StatusCreated).JSON(saved)
}

// PUT /restaurant/:restaurant_id/payment
func (c *RegistrationController) UpdatePaymentInfo(ctx *fiber.Ctx) error {
	restaurantID := ctx.Params("restaurant_id")

	var body struct {
		TransactionID *string `json:"transaction_id"`
		PaymentDate   *string `json:"payment_date"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}

	updates := map[string]interface{}{
		"transaction_id": body.TransactionID,
		"payment_date":   body.PaymentDate,
	}
	saved, err := c.repo.UpdateRegistration(ctx.Context(), restaurantID, updates)
	if err != nil {
		return response.NotFound(ctx, "registration not found")
	}
	return ctx.JSON(saved)
}

// PUT /restaurant/:restaurant_id/complete
func (c *RegistrationController) CompleteRegistration(ctx *fiber.Ctx) error {
	restaurantID := ctx.Params("restaurant_id")

	saved, err := c.repo.UpdateRegistration(ctx.Context(), restaurantID, map[string]interface{}{
		"registration_completed": true,
	})
	if err != nil {
		return response.NotFound(ctx, "registration not found")
	}
	return ctx.JSON(saved)
}

// GET /restaurant/:restaurant_id
func (c *RegistrationController) GetRestaurantData(ctx *fiber.Ctx) error {
	restaurantID := ctx.Params("restaurant_id")

	payload, err := c.repo.GetRegistration(ctx.Context(), restaurantID)
	if err != nil {
		return response.NotFound(ctx, "registration not found")
	}
	return ctx.JSON(payload)
}

// GET /restaurant/:restaurant_id/outlets
// The Partner app reads response.data['outlets'].
func (c *RegistrationController) GetVendorOutlets(ctx *fiber.Ctx) error {
	restaurantID := ctx.Params("restaurant_id")

	outlets, err := c.repo.GetVendorOutlets(ctx.Context(), restaurantID)
	if err != nil {
		return response.InternalError(ctx, "could not fetch outlets")
	}
	return ctx.JSON(fiber.Map{"outlets": outlets})
}
