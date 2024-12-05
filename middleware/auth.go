package middleware

import (
	"dailzo/globals"
	"dailzo/utils"

	"github.com/gofiber/fiber/v2"
)

func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Get("Authorization")
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authorization header is missing"})
		}

		// Parse and validate the JWT token
		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Attach user ID to the context
		c.Locals("user_id", claims["sub"])
		globals.UpdateUserID(claims["sub"].(string))
		return c.Next()
	}
}
