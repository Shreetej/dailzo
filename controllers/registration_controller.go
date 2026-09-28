package controllers

import (
	"dailzo/pkg/response"
	"dailzo/repository"
	"errors"

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

// POST /vendor/register — unified onboarding submit for restaurant/grocery/
// delivery partners. Persists the full profile payload keyed by the JWT user.
func (c *RegistrationController) RegisterVendor(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
	var payload map[string]interface{}
	if err := ctx.BodyParser(&payload); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	userType, _ := payload["user_type"].(string)
	if userType == "" {
		userType = "vendor"
	}
	if err := c.repo.SaveVendorRegistration(ctx.Context(), userID, userType, payload); err != nil {
		if errors.Is(err, repository.ErrInviteCodeRequired) || errors.Is(err, repository.ErrInvalidInviteCode) {
			return response.BadRequest(ctx, err.Error())
		}
		return response.InternalError(ctx, "could not save registration")
	}
	return response.Success(ctx, fiber.Map{"registration_completed": true, "user_type": userType})
}

// GET /vendor/registration — returns the caller's onboarding submission and
// its real approval status (submitted | approved | rejected — see
// AdminRepository.Approve/Reject, which now write this back). Previously
// `registration_completed` was `status == "submitted"`, which meant a
// vendor's own app would report them as NOT registered anymore the moment
// they were approved (status flips to "approved"), routing them back into
// the registration wizard instead of their dashboard. `registration_completed`
// now means "has ever submitted a registration" — approval state is a
// separate field the caller must check to decide dashboard vs pending-screen.
func (c *RegistrationController) GetVendorRegistration(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
	payload, status, err := c.repo.GetVendorRegistration(ctx.Context(), userID)
	if err != nil {
		// No submission yet → not an error, just not completed.
		return response.Success(ctx, fiber.Map{"registration_completed": false, "status": "none"})
	}
	return response.Success(ctx, fiber.Map{
		"registration_completed": true,
		"status":                 status,
		"payload":                payload,
	})
}

// POST /restaurant/register — requires a JWT (the vendor app is already
// logged in by this point in onboarding); the row is created owned by the
// caller so every later payment/complete/read call on it can be ownership-
// checked instead of trusting a bare restaurant_id path param.
func (c *RegistrationController) RegisterRestaurant(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
	var payload map[string]interface{}
	if err := ctx.BodyParser(&payload); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}

	saved, err := c.repo.CreateRegistration(ctx.Context(), userID, payload)
	if err != nil {
		return response.Forbidden(ctx, "could not register restaurant")
	}
	return ctx.Status(fiber.StatusCreated).JSON(saved)
}

// PUT /restaurant/:restaurant_id/payment
func (c *RegistrationController) UpdatePaymentInfo(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
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
	saved, err := c.repo.UpdateRegistration(ctx.Context(), restaurantID, userID, updates)
	if err != nil {
		return response.NotFound(ctx, "registration not found")
	}
	return ctx.JSON(saved)
}

// PUT /restaurant/:restaurant_id/complete
func (c *RegistrationController) CompleteRegistration(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
	restaurantID := ctx.Params("restaurant_id")

	saved, err := c.repo.UpdateRegistration(ctx.Context(), restaurantID, userID, map[string]interface{}{
		"registration_completed": true,
	})
	if err != nil {
		return response.NotFound(ctx, "registration not found")
	}
	return ctx.JSON(saved)
}

// GET /restaurant/mine — resolves the authenticated vendor's own restaurant
// registration without needing to already know its id.
func (c *RegistrationController) GetMyRestaurant(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
	payload, err := c.repo.GetRegistrationByUser(ctx.Context(), userID)
	if err != nil {
		return response.NotFound(ctx, "no restaurant registered for this account yet")
	}
	return ctx.JSON(payload)
}

// GET /restaurant/:restaurant_id
func (c *RegistrationController) GetRestaurantData(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
	restaurantID := ctx.Params("restaurant_id")

	payload, err := c.repo.GetRegistration(ctx.Context(), restaurantID, userID)
	if err != nil {
		return response.NotFound(ctx, "registration not found")
	}
	return ctx.JSON(payload)
}

// GET /restaurant/:restaurant_id/outlets
// The Partner app reads response.data['outlets'].
func (c *RegistrationController) GetVendorOutlets(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return response.Unauthorized(ctx, "")
	}
	restaurantID := ctx.Params("restaurant_id")
	owned, err := c.repo.IsRegistrationOwner(ctx.Context(), restaurantID, userID)
	if err != nil {
		return response.InternalError(ctx, "could not verify ownership")
	}
	if !owned {
		return response.Forbidden(ctx, "not your restaurant")
	}

	outlets, err := c.repo.GetVendorOutlets(ctx.Context(), restaurantID)
	if err != nil {
		return response.InternalError(ctx, "could not fetch outlets")
	}
	return ctx.JSON(fiber.Map{"outlets": outlets})
}
