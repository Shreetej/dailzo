package server

import (
	"dailzo/internal/api"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// Server implements the api.ServerInterface
type Server struct {
	userRepo     *repository.UserRepository
	orderRepo    *repository.OrderRepository
	productRepo  *repository.FoodProductRepository
	deliveryRepo *repository.DeliveryRepository
	groceryRepo  *repository.GroceryRepository
	adminRepo    *repository.AdminRepository
	otpRepo      *repository.OTPRepository
}

// NewServer creates a new server instance
func NewServer(
	userRepo *repository.UserRepository,
	orderRepo *repository.OrderRepository,
	productRepo *repository.FoodProductRepository,
	deliveryRepo *repository.DeliveryRepository,
	groceryRepo *repository.GroceryRepository,
	adminRepo *repository.AdminRepository,
	otpRepo *repository.OTPRepository,
) api.ServerInterface {
	return &Server{
		userRepo:     userRepo,
		orderRepo:    orderRepo,
		productRepo:  productRepo,
		deliveryRepo: deliveryRepo,
		groceryRepo:  groceryRepo,
		adminRepo:    adminRepo,
		otpRepo:      otpRepo,
	}
}

// getUserIDFromContext extracts user ID from fiber context
func getUserIDFromContext(c *fiber.Ctx) string {
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return ""
	}
	return userID
}
