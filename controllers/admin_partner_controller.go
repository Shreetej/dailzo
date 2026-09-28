package controllers

import (
	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// AdminPartnerController covers admin partner-management actions added
// alongside the OpenAPI-generated admin routes (internal/server/admin_handlers.go):
// resume/reinstate, commission rate, and per-partner performance. Registered
// as hand-written Fiber routes in main.go, same pattern as the reconciliation
// routes, rather than extending the generated OpenAPI surface.
type AdminPartnerController struct {
	repo *repository.AdminRepository
}

func NewAdminPartnerController(repo *repository.AdminRepository) *AdminPartnerController {
	return &AdminPartnerController{repo: repo}
}

// PostResume → POST /admin/partners/:id/resume?type=restaurant|grocery|delivery|delivery_vendor
func (ctrl *AdminPartnerController) PostResume(c *fiber.Ctx) error {
	id := c.Params("id")
	partnerType := c.Query("type", "delivery")
	adminID, _ := c.Locals("user_id").(string)

	if err := ctrl.repo.ResumePartner(c.Context(), id, partnerType, adminID); err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.SuccessWithMessage(c, "Partner resumed successfully", nil)
}

// GetPerformance → GET /admin/partners/:id/performance?type=...&days=7
func (ctrl *AdminPartnerController) GetPerformance(c *fiber.Ctx) error {
	id := c.Params("id")
	partnerType := c.Query("type", "restaurant")
	days := c.QueryInt("days", 7)

	perf, err := ctrl.repo.GetPartnerPerformance(c.Context(), id, partnerType, days)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PERFORMANCE_ERROR", err.Error())
	}
	return response.Success(c, perf)
}

// GetCommission → GET /admin/partners/:id/commission?type=...
func (ctrl *AdminPartnerController) GetCommission(c *fiber.Ctx) error {
	id := c.Params("id")
	partnerType := c.Query("type", "restaurant")

	commission, err := ctrl.repo.GetPartnerCommission(c.Context(), id, partnerType)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "COMMISSION_ERROR", err.Error())
	}
	return response.Success(c, commission)
}

type setCommissionReq struct {
	CommissionRate float64 `json:"commission_rate"`
}

// PutCommission → PUT /admin/partners/:id/commission?type=...
func (ctrl *AdminPartnerController) PutCommission(c *fiber.Ctx) error {
	id := c.Params("id")
	partnerType := c.Query("type", "restaurant")

	var req setCommissionReq
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := ctrl.repo.SetPartnerCommission(c.Context(), id, partnerType, req.CommissionRate); err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.SuccessWithMessage(c, "Commission rate updated", nil)
}

// GetRevenueTimeseries → GET /admin/reports/timeseries?days=7
func (ctrl *AdminPartnerController) GetRevenueTimeseries(c *fiber.Ctx) error {
	days := c.QueryInt("days", 7)
	series, err := ctrl.repo.GetRevenueTimeseries(c.Context(), days)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TIMESERIES_ERROR", err.Error())
	}
	return response.Success(c, series)
}
