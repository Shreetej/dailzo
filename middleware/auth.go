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

		// Remove "Bearer " prefix if present
		if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
			tokenStr = tokenStr[7:]
		}

		// Parse and validate the JWT token
		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Attach user ID to the context
		if sub, ok := claims["sub"].(string); ok {
			c.Locals("user_id", sub)
			globals.UpdateUserID(sub)
		}

		// Attach user_type to the context for role-based access control
		if userType, ok := claims["user_type"].(string); ok {
			c.Locals("user_type", userType)
		}

		return c.Next()
	}
}

// OptionalJWTMiddleware extracts user info if token is present but doesn't require it
func OptionalJWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Get("Authorization")
		if tokenStr == "" {
			return c.Next()
		}

		// Remove "Bearer " prefix if present
		if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
			tokenStr = tokenStr[7:]
		}

		// Parse and validate the JWT token
		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			// Token is invalid but it's optional, so continue
			return c.Next()
		}

		// Attach user ID to the context
		if sub, ok := claims["sub"].(string); ok {
			c.Locals("user_id", sub)
			globals.UpdateUserID(sub)
		}

		// Attach user_type to the context
		if userType, ok := claims["user_type"].(string); ok {
			c.Locals("user_type", userType)
		}

		return c.Next()
	}
}
