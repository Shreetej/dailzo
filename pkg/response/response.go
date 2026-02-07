package response

import "github.com/gofiber/fiber/v2"

// ApiResponse represents the standard API response format
type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ApiError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ApiError represents an error in the API response
type ApiError struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorDetail provides additional error information
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// Meta contains pagination and other metadata
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// Success returns a successful API response
func Success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(ApiResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage returns a successful API response with a message
func SuccessWithMessage(c *fiber.Ctx, message string, data interface{}) error {
	return c.JSON(ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta returns a successful API response with metadata
func SuccessWithMeta(c *fiber.Ctx, data interface{}, meta *Meta) error {
	return c.JSON(ApiResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created returns a 201 Created response
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(ApiResponse{
		Success: true,
		Data:    data,
	})
}

// NoContent returns a 204 No Content response
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Error returns an error API response
func Error(c *fiber.Ctx, status int, code string, message string) error {
	return c.Status(status).JSON(ApiResponse{
		Success: false,
		Error: &ApiError{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails returns an error API response with details
func ErrorWithDetails(c *fiber.Ctx, status int, code string, message string, details []ErrorDetail) error {
	return c.Status(status).JSON(ApiResponse{
		Success: false,
		Error: &ApiError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest returns a 400 Bad Request response
func BadRequest(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized returns a 401 Unauthorized response
func Unauthorized(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "Authentication required"
	}
	return Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden returns a 403 Forbidden response
func Forbidden(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "Access denied"
	}
	return Error(c, fiber.StatusForbidden, "FORBIDDEN", message)
}

// NotFound returns a 404 Not Found response
func NotFound(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "Resource not found"
	}
	return Error(c, fiber.StatusNotFound, "NOT_FOUND", message)
}

// Conflict returns a 409 Conflict response
func Conflict(c *fiber.Ctx, message string) error {
	return Error(c, fiber.StatusConflict, "CONFLICT", message)
}

// ValidationError returns a 422 Unprocessable Entity response
func ValidationError(c *fiber.Ctx, details []ErrorDetail) error {
	return ErrorWithDetails(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", details)
}

// InternalError returns a 500 Internal Server Error response
func InternalError(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "An unexpected error occurred"
	}
	return Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// TooManyRequests returns a 429 Too Many Requests response
func TooManyRequests(c *fiber.Ctx, message string) error {
	if message == "" {
		message = "Rate limit exceeded"
	}
	return Error(c, fiber.StatusTooManyRequests, "RATE_LIMITED", message)
}
