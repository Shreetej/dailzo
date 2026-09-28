package middleware

import (
	"dailzo/utils"

	"github.com/gofiber/fiber/v2"
)

// AuthenticateRequest validates the Authorization header and, on success,
// attaches user_id/user_type to c.Locals — but unlike JWTMiddleware it does
// NOT call c.Next() itself. This lets a caller (e.g. main.go's v1Auth) run
// additional checks (like an admin-only path check) BETWEEN authentication
// and advancing to the real route handler, which isn't otherwise possible:
// c.Next() executes the rest of the handler chain synchronously, so any code
// written after a call to JWTMiddleware()(c) would only run once the real
// handler had already responded. A non-nil return here means the response
// was already written (401) and the caller must return it as-is without
// calling c.Next().
func AuthenticateRequest(c *fiber.Ctx) error {
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

	// Attach user ID to the context. This is the ONLY place the
	// authenticated user id is stored — per-request, via fiber's Locals
	// (backed by fasthttp's per-request UserValue store), not a
	// package-level variable. A prior version also mirrored this into a
	// process-wide global (globals.CurrentLoggedInUser) that every
	// concurrent request wrote to; under load, one request's audit-trail
	// writes (created_by/last_modified_by) could pick up a different
	// concurrent request's user id. Removed — repositories now read the
	// user id via repository.LoggedInUserID(ctx.Context()), which safely
	// resolves to this same per-request value (fasthttp.RequestCtx.Value
	// delegates to UserValue).
	if sub, ok := claims["sub"].(string); ok {
		c.Locals("user_id", sub)
	}

	// Attach user_type to the context for role-based access control
	if userType, ok := claims["user_type"].(string); ok {
		c.Locals("user_type", userType)
	}

	return nil
}

func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := AuthenticateRequest(c); err != nil {
			return err
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

		// Attach user ID to the context. This is the ONLY place the
		// authenticated user id is stored — per-request, via fiber's Locals
		// (backed by fasthttp's per-request UserValue store), not a
		// package-level variable. A prior version also mirrored this into a
		// process-wide global (globals.CurrentLoggedInUser) that every
		// concurrent request wrote to; under load, one request's audit-trail
		// writes (created_by/last_modified_by) could pick up a different
		// concurrent request's user id. Removed — repositories now read the
		// user id via repository.LoggedInUserID(ctx.Context()), which safely
		// resolves to this same per-request value (fasthttp.RequestCtx.Value
		// delegates to UserValue).
		if sub, ok := claims["sub"].(string); ok {
			c.Locals("user_id", sub)
		}

		// Attach user_type to the context
		if userType, ok := claims["user_type"].(string); ok {
			c.Locals("user_type", userType)
		}

		return c.Next()
	}
}
