package controllers

import (
	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// AnalyticsController serves the vendor "Product Intelligence" dashboard
// sections. The vendor app reads these responses as a bare {"items": [...]}
// object (repository reads response.data['items']), so these handlers return
// that shape directly rather than the standard {success,data} envelope.
type AnalyticsController struct {
	repo *repository.AnalyticsRepository
}

func NewAnalyticsController(repo *repository.AnalyticsRepository) *AnalyticsController {
	return &AnalyticsController{repo: repo}
}

func (c *AnalyticsController) queryParams(ctx *fiber.Ctx) (role, timeframe string) {
	role = ctx.Query("role", "grocery")
	timeframe = ctx.Query("timeframe", "7d")
	return
}

// ProductTrends → GET /analytics/vendor/products/trending
func (c *AnalyticsController) ProductTrends(ctx *fiber.Ctx) error {
	role, tf := c.queryParams(ctx)
	items, err := c.repo.ProductTrends(ctx.Context(), role, tf)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": fiber.Map{"code": "ANALYTICS_ERROR", "message": err.Error()}})
	}
	return ctx.JSON(fiber.Map{"items": items})
}

// CategoryTrends → GET /analytics/vendor/categories/trending
func (c *AnalyticsController) CategoryTrends(ctx *fiber.Ctx) error {
	role, tf := c.queryParams(ctx)
	items, err := c.repo.CategoryTrends(ctx.Context(), role, tf)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": fiber.Map{"code": "ANALYTICS_ERROR", "message": err.Error()}})
	}
	return ctx.JSON(fiber.Map{"items": items})
}

// LowSellers → GET /analytics/vendor/products/low-sellers
func (c *AnalyticsController) LowSellers(ctx *fiber.Ctx) error {
	role, tf := c.queryParams(ctx)
	items, err := c.repo.LowSellers(ctx.Context(), role, tf)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": fiber.Map{"code": "ANALYTICS_ERROR", "message": err.Error()}})
	}
	return ctx.JSON(fiber.Map{"items": items})
}

// PayoutSummary → GET /grocery/payout-summary
// Unlike the analytics endpoints above, the grocery datasource unwraps the
// standard {success,data} envelope, so this returns response.Success.
func (c *AnalyticsController) PayoutSummary(ctx *fiber.Ctx) error {
	role := ctx.Query("role", "grocery")
	summary, err := c.repo.PayoutSummary(ctx.Context(), role)
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "PAYOUT_ERROR", err.Error())
	}
	return response.Success(ctx, summary)
}

// UnmetDemand → GET /vendor/search-demand/unmet
// Aggregates real customer search_logs into queries that returned no results.
func (c *AnalyticsController) UnmetDemand(ctx *fiber.Ctx) error {
	role, tf := c.queryParams(ctx)
	items, err := c.repo.UnmetDemand(ctx.Context(), role, tf)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": fiber.Map{"code": "ANALYTICS_ERROR", "message": err.Error()}})
	}
	return ctx.JSON(fiber.Map{"items": items})
}

// AdminUnmetDemand → GET /admin/search-demand/unmet
// Platform-wide unmet demand across every partner type (see vendor UnmetDemand
// above for the per-role version).
func (c *AnalyticsController) AdminUnmetDemand(ctx *fiber.Ctx) error {
	tf := ctx.Query("timeframe", "7d")
	items, err := c.repo.UnmetDemandAdmin(ctx.Context(), tf)
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "ANALYTICS_ERROR", err.Error())
	}
	return response.Success(ctx, items)
}

type adminNotifyUnmetDemandReq struct {
	Query string `json:"query"`
}

// AdminNotifyUnmetDemand → POST /admin/search-demand/unmet/notify
func (c *AnalyticsController) AdminNotifyUnmetDemand(ctx *fiber.Ctx) error {
	var req adminNotifyUnmetDemandReq
	if err := ctx.BodyParser(&req); err != nil || req.Query == "" {
		return response.BadRequest(ctx, "query is required")
	}
	adminID, _ := ctx.Locals("user_id").(string)
	if err := c.repo.NotifyUnmetDemand(ctx.Context(), req.Query, adminID); err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "ANALYTICS_ERROR", err.Error())
	}
	return response.SuccessWithMessage(ctx, "Vendors notified", nil)
}

// GetStoreStatus → GET /vendor/store-status
func (c *AnalyticsController) GetStoreStatus(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	isOpen, err := c.repo.GetStoreStatus(ctx.Context(), userID)
	if err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "STORE_STATUS_ERROR", err.Error())
	}
	return response.Success(ctx, fiber.Map{"is_open": isOpen})
}

// SetStoreStatus → PUT /vendor/store-status  body {is_open: bool}
func (c *AnalyticsController) SetStoreStatus(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	var body struct {
		IsOpen bool `json:"is_open"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "invalid body")
	}
	if err := c.repo.SetStoreStatus(ctx.Context(), userID, body.IsOpen); err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "STORE_STATUS_ERROR", err.Error())
	}
	return response.Success(ctx, fiber.Map{"is_open": body.IsOpen})
}

// LogSearch → POST /search-log  body {query, city, result_count, kind}
func (c *AnalyticsController) LogSearch(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	var body struct {
		Query       string `json:"query"`
		City        string `json:"city"`
		ResultCount int    `json:"result_count"`
		Kind        string `json:"kind"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "invalid body")
	}
	if err := c.repo.LogSearch(ctx.Context(), userID, body.Query, body.City, body.Kind, body.ResultCount); err != nil {
		return response.Error(ctx, fiber.StatusInternalServerError, "SEARCH_LOG_ERROR", err.Error())
	}
	return response.Success(ctx, fiber.Map{"logged": true})
}
