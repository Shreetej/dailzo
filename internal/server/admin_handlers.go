package server

import (
	"dailzo/internal/api"
	"dailzo/models"
	"dailzo/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// GetAdminApprovals handles GET /admin/approvals
func (s *Server) GetAdminApprovals(c *fiber.Ctx) error {
	approvals, err := s.adminRepo.GetApprovals(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to fetch approvals")
	}

	if approvals == nil {
		approvals = []models.Approval{}
	}

	return response.Success(c, approvals)
}

// PostAdminApprovalsIdApprove handles POST /admin/approvals/{id}/approve
func (s *Server) PostAdminApprovalsIdApprove(c *fiber.Ctx, id string) error {
	adminID := getUserIDFromContext(c)
	if adminID == "" {
		return response.Unauthorized(c, "")
	}

	var req api.PostAdminApprovalsIdApproveJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	err := s.adminRepo.Approve(c.Context(), id, notes, adminID)
	if err != nil {
		if err.Error() == "approval not found or already processed" {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, "Failed to approve")
	}

	return response.Success(c, fiber.Map{
		"message": "Approval successful",
	})
}

// PostAdminApprovalsIdReject handles POST /admin/approvals/{id}/reject
func (s *Server) PostAdminApprovalsIdReject(c *fiber.Ctx, id string) error {
	adminID := getUserIDFromContext(c)
	if adminID == "" {
		return response.Unauthorized(c, "")
	}

	var req api.PostAdminApprovalsIdRejectJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	err := s.adminRepo.Reject(c.Context(), id, notes, adminID)
	if err != nil {
		if err.Error() == "approval not found or already processed" {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, "Failed to reject")
	}

	return response.Success(c, fiber.Map{
		"message": "Rejection successful",
	})
}

// GetAdminPartners handles GET /admin/partners
func (s *Server) GetAdminPartners(c *fiber.Ctx) error {
	partners, err := s.adminRepo.GetPartners(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to fetch partners")
	}

	if partners == nil {
		partners = []models.Partner{}
	}

	return response.Success(c, partners)
}

// PostAdminPartnersIdSuspend handles POST /admin/partners/{id}/suspend
func (s *Server) PostAdminPartnersIdSuspend(c *fiber.Ctx, id string) error {
	adminID := getUserIDFromContext(c)
	if adminID == "" {
		return response.Unauthorized(c, "")
	}

	var req api.PostAdminPartnersIdSuspendJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	reason := ""
	if req.Reason != nil {
		reason = *req.Reason
	}

	// Get partner type from query param or body
	partnerType := c.Query("type", "delivery")

	err := s.adminRepo.SuspendPartner(c.Context(), id, partnerType, reason, adminID)
	if err != nil {
		return response.InternalError(c, "Failed to suspend partner")
	}

	return response.Success(c, fiber.Map{
		"message": "Partner suspended successfully",
	})
}

// GetAdminComplaints handles GET /admin/complaints
func (s *Server) GetAdminComplaints(c *fiber.Ctx) error {
	complaints, err := s.adminRepo.GetComplaints(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to fetch complaints")
	}

	if complaints == nil {
		complaints = []models.Complaint{}
	}

	return response.Success(c, complaints)
}

// PostAdminComplaintsIdResolve handles POST /admin/complaints/{id}/resolve
func (s *Server) PostAdminComplaintsIdResolve(c *fiber.Ctx, id string) error {
	adminID := getUserIDFromContext(c)
	if adminID == "" {
		return response.Unauthorized(c, "")
	}

	var req api.PostAdminComplaintsIdResolveJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	notes := ""
	if req.ResolutionNotes != nil {
		notes = *req.ResolutionNotes
	}

	refundAmount := float64(0)
	if req.RefundAmount != nil {
		refundAmount = float64(*req.RefundAmount)
	}

	err := s.adminRepo.ResolveComplaint(c.Context(), id, notes, refundAmount, adminID)
	if err != nil {
		if err.Error() == "complaint not found or already resolved" {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, "Failed to resolve complaint")
	}

	return response.Success(c, fiber.Map{
		"message": "Complaint resolved successfully",
	})
}

// GetAdminComplaintsIdInvestigation handles GET /admin/complaints/{id}/investigation
func (s *Server) GetAdminComplaintsIdInvestigation(c *fiber.Ctx, id string) error {
	investigation, err := s.adminRepo.GetInvestigation(c.Context(), id)
	if err != nil {
		if err.Error() == "complaint not found" {
			return response.NotFound(c, "Complaint not found")
		}
		return response.InternalError(c, "Failed to fetch investigation")
	}

	return response.Success(c, investigation)
}

// GetAdminReportsKpis handles GET /admin/reports/kpis
func (s *Server) GetAdminReportsKpis(c *fiber.Ctx) error {
	kpis, err := s.adminRepo.GetPlatformKpis(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to fetch platform KPIs")
	}

	return response.Success(c, kpis)
}

// GetAdminOnboardingLeads handles GET /admin/onboarding-leads
func (s *Server) GetAdminOnboardingLeads(c *fiber.Ctx) error {
	leads, err := s.adminRepo.GetOnboardingLeads(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to fetch onboarding leads")
	}

	if leads == nil {
		leads = []models.OnboardingLead{}
	}

	return response.Success(c, leads)
}

// PostAdminOnboardingLeadsIdNotify handles POST /admin/onboarding-leads/{id}/notify
func (s *Server) PostAdminOnboardingLeadsIdNotify(c *fiber.Ctx, id string) error {
	var req api.PostAdminOnboardingLeadsIdNotifyJSONRequestBody
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	channel := "email"
	if req.Channel != nil {
		channel = *req.Channel
	}

	message := ""
	if req.Message != nil {
		message = *req.Message
	}

	err := s.adminRepo.NotifyLead(c.Context(), id, channel, message)
	if err != nil {
		return response.InternalError(c, "Failed to send notification")
	}

	return response.Success(c, fiber.Map{
		"message": "Notification sent successfully",
		"channel": channel,
	})
}
