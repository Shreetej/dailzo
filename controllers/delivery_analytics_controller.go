package controllers

import (
	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// DeliveryAnalyticsController serves the delivery-partner intelligence dashboard
// (KPIs, weekly earnings, SLA, peak-time/zone insights, recommendations, shifts,
// payout, online toggle). The delivery datasource unwraps the {success,data}
// envelope, so every handler returns response.Success.
type DeliveryAnalyticsController struct {
	repo *repository.DeliveryAnalyticsRepository
}

func NewDeliveryAnalyticsController(repo *repository.DeliveryAnalyticsRepository) *DeliveryAnalyticsController {
	return &DeliveryAnalyticsController{repo: repo}
}

func (c *DeliveryAnalyticsController) partnerID(ctx *fiber.Ctx) string {
	id, _ := ctx.Locals("user_id").(string)
	return id
}

func (c *DeliveryAnalyticsController) fail(ctx *fiber.Ctx, err error) error {
	return response.Error(ctx, fiber.StatusInternalServerError, "DELIVERY_ANALYTICS_ERROR", err.Error())
}

// Kpis → GET /delivery/kpis
func (c *DeliveryAnalyticsController) Kpis(ctx *fiber.Ctx) error {
	v, err := c.repo.Kpis(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, v)
}

// WeeklyEarnings → GET /delivery/earnings/weekly
func (c *DeliveryAnalyticsController) WeeklyEarnings(ctx *fiber.Ctx) error {
	v, err := c.repo.WeeklyEarnings(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, v)
}

// SlaKpis → GET /delivery/sla-kpis
func (c *DeliveryAnalyticsController) SlaKpis(ctx *fiber.Ctx) error {
	v, err := c.repo.SlaKpis(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, v)
}

// EarningsInsights → GET /delivery/earnings-insights
func (c *DeliveryAnalyticsController) EarningsInsights(ctx *fiber.Ctx) error {
	v, err := c.repo.EarningsInsights(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, v)
}

// Recommendations → GET /delivery/recommendations
func (c *DeliveryAnalyticsController) Recommendations(ctx *fiber.Ctx) error {
	v, err := c.repo.Recommendations(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, v)
}

// Shifts → GET /delivery/shifts
func (c *DeliveryAnalyticsController) Shifts(ctx *fiber.Ctx) error {
	v, err := c.repo.Shifts(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, v)
}

// PayoutSummary → GET /delivery/payout-summary
func (c *DeliveryAnalyticsController) PayoutSummary(ctx *fiber.Ctx) error {
	v, err := c.repo.PayoutSummary(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, v)
}

// GetOnline → GET /delivery/online
func (c *DeliveryAnalyticsController) GetOnline(ctx *fiber.Ctx) error {
	v, err := c.repo.GetOnline(ctx.Context(), c.partnerID(ctx))
	if err != nil {
		return c.fail(ctx, err)
	}
	return response.Success(ctx, fiber.Map{"is_online": v})
}

// SetOnline → PUT /delivery/online  body {is_online: bool}
func (c *DeliveryAnalyticsController) SetOnline(ctx *fiber.Ctx) error {
	var body struct {
		IsOnline bool `json:"is_online"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.Error(ctx, fiber.StatusBadRequest, "BAD_REQUEST", "invalid body")
	}
	if err := c.repo.SetOnline(ctx.Context(), c.partnerID(ctx), body.IsOnline); err != nil {
		if body.IsOnline {
			// SetOnline only returns an error going online — pending-approval
			// is a caller-facing 403, not a server fault.
			return response.Error(ctx, fiber.StatusForbidden, "NOT_APPROVED", err.Error())
		}
		return c.fail(ctx, err)
	}
	return response.Success(ctx, fiber.Map{"is_online": body.IsOnline})
}
