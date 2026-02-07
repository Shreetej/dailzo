package middleware

import (
	"dailzo/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// RoleMiddleware checks if the user has one of the required roles
func RoleMiddleware(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userType, ok := c.Locals("user_type").(string)
		if !ok || userType == "" {
			return response.Unauthorized(c, "User type not found in token")
		}

		for _, role := range allowedRoles {
			if userType == role {
				return c.Next()
			}
		}

		return response.Forbidden(c, "Insufficient permissions")
	}
}

// AdminOnly restricts access to admin users only
func AdminOnly() fiber.Handler {
	return RoleMiddleware("admin")
}

// DeliveryOnly restricts access to delivery partners only
func DeliveryOnly() fiber.Handler {
	return RoleMiddleware("delivery")
}

// GroceryOnly restricts access to grocery partners only
func GroceryOnly() fiber.Handler {
	return RoleMiddleware("grocery")
}

// PartnerOnly restricts access to delivery or grocery partners
func PartnerOnly() fiber.Handler {
	return RoleMiddleware("delivery", "grocery")
}
