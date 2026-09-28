package server

import (
	"dailzo/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// GetDeliveryVendorMine handles GET /delivery-vendor/mine — resolves the
// caller's own fleet/agency profile (mirrors GetMyRestaurant/GetGroceryProfile).
func (s *Server) GetDeliveryVendorMine(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	profile, err := s.deliveryVendorRepo.GetProfileByUserID(c.Context(), userID)
	if err != nil {
		return response.NotFound(c, "Delivery vendor profile not found")
	}

	return response.Success(c, profile)
}

// GetDeliveryVendorRiders handles GET /delivery-vendor/riders — riders
// awaiting THIS vendor's own approval (ownership-scoped by the caller's
// user_id, not a path/query parameter, so one fleet can't list another's).
func (s *Server) GetDeliveryVendorRiders(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	riders, err := s.deliveryVendorRepo.GetPendingRiders(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch riders")
	}

	return response.Success(c, riders)
}

// PostDeliveryVendorRidersApprove handles POST /delivery-vendor/riders/:id/approve
func (s *Server) PostDeliveryVendorRidersApprove(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}
	riderID := c.Params("id")

	if err := s.deliveryVendorRepo.ApproveRider(c.Context(), userID, riderID); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Rider approved"})
}

// PostDeliveryVendorRidersReject handles POST /delivery-vendor/riders/:id/reject
func (s *Server) PostDeliveryVendorRidersReject(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}
	riderID := c.Params("id")

	if err := s.deliveryVendorRepo.RejectRider(c.Context(), userID, riderID); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Rider rejected"})
}

// GetDeliveryVendorRidersActive handles GET /delivery-vendor/riders/active
func (s *Server) GetDeliveryVendorRidersActive(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	riders, err := s.deliveryVendorRepo.GetActiveRiders(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch riders")
	}

	return response.Success(c, riders)
}

// GetDeliveryVendorRidersSuspended handles GET /delivery-vendor/riders/suspended
func (s *Server) GetDeliveryVendorRidersSuspended(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	riders, err := s.deliveryVendorRepo.GetSuspendedRiders(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch riders")
	}

	return response.Success(c, riders)
}

// PostDeliveryVendorRidersSuspend handles POST /delivery-vendor/riders/:id/suspend
func (s *Server) PostDeliveryVendorRidersSuspend(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}
	riderID := c.Params("id")

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&body)

	if err := s.deliveryVendorRepo.SuspendRider(c.Context(), userID, riderID, body.Reason); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Rider suspended"})
}

// PostDeliveryVendorRidersReinstate handles POST /delivery-vendor/riders/:id/reinstate
func (s *Server) PostDeliveryVendorRidersReinstate(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}
	riderID := c.Params("id")

	if err := s.deliveryVendorRepo.ReinstateRider(c.Context(), userID, riderID); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Rider reinstated"})
}

// GetDeliveryVendorComplaints handles GET /delivery-vendor/complaints — every
// complaint on an order THIS vendor's own riders delivered.
func (s *Server) GetDeliveryVendorComplaints(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}

	complaints, err := s.deliveryVendorRepo.GetComplaints(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to fetch complaints")
	}

	return response.Success(c, complaints)
}

// PostDeliveryVendorComplaintsResolve handles
// POST /delivery-vendor/complaints/:id/resolve
func (s *Server) PostDeliveryVendorComplaintsResolve(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}
	complaintID := c.Params("id")

	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.BodyParser(&body)

	if err := s.deliveryVendorRepo.ResolveComplaint(c.Context(), userID, complaintID, body.Notes); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Complaint resolved"})
}

// PostDeliveryVendorComplaintsEscalate handles
// POST /delivery-vendor/complaints/:id/escalate
func (s *Server) PostDeliveryVendorComplaintsEscalate(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}
	complaintID := c.Params("id")

	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.BodyParser(&body)

	if err := s.deliveryVendorRepo.EscalateComplaint(c.Context(), userID, complaintID, body.Notes); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Complaint escalated to admin"})
}

// PostDeliveryVendorComplaintsUpdate handles
// POST /delivery-vendor/complaints/:id/update — logs an investigation
// update without closing the complaint.
func (s *Server) PostDeliveryVendorComplaintsUpdate(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return response.Unauthorized(c, "")
	}
	complaintID := c.Params("id")

	var body struct {
		Note string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil || body.Note == "" {
		return response.BadRequest(c, "note is required")
	}

	if err := s.deliveryVendorRepo.UpdateComplaintNotes(c.Context(), userID, complaintID, body.Note); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{"message": "Update logged"})
}
