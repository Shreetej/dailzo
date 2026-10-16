package controllers

import (
	"strconv"

	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// VendorSupportController serves the Partner app's complaint and outlet-rating
// screens. Those screens shipped against endpoints that were never implemented
// server-side, so each one 404'd; see VendorSupportRepository for the ownership
// model.
type VendorSupportController struct {
	repo *repository.VendorSupportRepository
}

func NewVendorSupportController(repo *repository.VendorSupportRepository) *VendorSupportController {
	return &VendorSupportController{repo: repo}
}

// scopeFor resolves the caller to the businesses they own. A missing user_id
// means the JWT middleware did not run, which is a routing mistake rather than
// a client error, but it is still answered as unauthorized.
func (c *VendorSupportController) scopeFor(ctx *fiber.Ctx) (repository.VendorScope, bool) {
	userID, _ := ctx.Locals("user_id").(string)
	if userID == "" {
		return repository.VendorScope{}, false
	}
	scope, err := c.repo.GetVendorScope(ctx.Context(), userID)
	if err != nil {
		return repository.VendorScope{}, false
	}
	return scope, true
}

// ownsOutlet guards the outlet-scoped reads. Without it, /ratings/outlets/:id
// would happily aggregate a competitor's outlet for any authenticated vendor.
func ownsOutlet(scope repository.VendorScope, outletID string) bool {
	for _, id := range scope.OutletIDs {
		if id == outletID {
			return true
		}
	}
	return false
}

// ─── Complaints ──────────────────────────────────────────────────────────────

// GetComplaints handles GET /complaints — complaints against this vendor's own
// orders. Distinct from GET /my-complaints, which is the customer app reading
// the complaints a user filed.
func (c *VendorSupportController) GetComplaints(ctx *fiber.Ctx) error {
	scope, ok := c.scopeFor(ctx)
	if !ok {
		return response.Unauthorized(ctx, "authentication required")
	}

	outletID := ctx.Query("outletId")
	if outletID != "" && !ownsOutlet(scope, outletID) {
		return response.Forbidden(ctx, "outlet does not belong to this account")
	}

	complaints, err := c.repo.GetComplaints(ctx.Context(), scope, ctx.Query("status"), outletID)
	if err != nil {
		return response.InternalError(ctx, "could not fetch complaints")
	}

	return response.Success(ctx, complaints)
}

// GetComplaintSummary handles GET /complaints/summary. Registered before
// /complaints/:id so Fiber does not capture "summary" as an id.
func (c *VendorSupportController) GetComplaintSummary(ctx *fiber.Ctx) error {
	scope, ok := c.scopeFor(ctx)
	if !ok {
		return response.Unauthorized(ctx, "authentication required")
	}

	summary, err := c.repo.GetComplaintSummary(ctx.Context(), scope)
	if err != nil {
		return response.InternalError(ctx, "could not summarise complaints")
	}

	return response.Success(ctx, summary)
}

// GetComplaintByID handles GET /complaints/:id.
func (c *VendorSupportController) GetComplaintByID(ctx *fiber.Ctx) error {
	scope, ok := c.scopeFor(ctx)
	if !ok {
		return response.Unauthorized(ctx, "authentication required")
	}

	complaint, err := c.repo.GetComplaintByID(ctx.Context(), scope, ctx.Params("id"))
	if err != nil {
		return response.NotFound(ctx, "complaint not found")
	}

	return response.Success(ctx, complaint)
}

// UpdateComplaintStatus handles PATCH /complaints/:id/status.
// The app sends refundAmount as a string (its Complaint model types it that
// way), so it is parsed here rather than typed as a number.
func (c *VendorSupportController) UpdateComplaintStatus(ctx *fiber.Ctx) error {
	userID, _ := ctx.Locals("user_id").(string)
	scope, ok := c.scopeFor(ctx)
	if !ok {
		return response.Unauthorized(ctx, "authentication required")
	}

	var body struct {
		Status          string `json:"status"`
		RefundAmount    string `json:"refundAmount"`
		ResolutionNotes string `json:"resolutionNotes"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if body.Status == "" {
		return response.BadRequest(ctx, "status is required")
	}

	var refundAmount *float64
	if body.RefundAmount != "" {
		parsed, err := strconv.ParseFloat(body.RefundAmount, 64)
		if err != nil {
			return response.BadRequest(ctx, "refundAmount must be a number")
		}
		refundAmount = &parsed
	}

	if err := c.repo.UpdateComplaintStatus(
		ctx.Context(), scope, ctx.Params("id"), body.Status, body.ResolutionNotes, refundAmount, userID,
	); err != nil {
		return response.BadRequest(ctx, err.Error())
	}

	return response.SuccessWithMessage(ctx, "updated", nil)
}

// ─── Outlet ratings ──────────────────────────────────────────────────────────

// GetOutletRatings handles GET /ratings/outlets — one aggregate per outlet the
// caller owns.
func (c *VendorSupportController) GetOutletRatings(ctx *fiber.Ctx) error {
	scope, ok := c.scopeFor(ctx)
	if !ok {
		return response.Unauthorized(ctx, "authentication required")
	}

	ratings, err := c.repo.GetOutletRatings(ctx.Context(), scope)
	if err != nil {
		return response.InternalError(ctx, "could not fetch outlet ratings")
	}

	return response.Success(ctx, ratings)
}

// GetOutletRating handles GET /ratings/outlets/:outletId.
func (c *VendorSupportController) GetOutletRating(ctx *fiber.Ctx) error {
	scope, ok := c.scopeFor(ctx)
	if !ok {
		return response.Unauthorized(ctx, "authentication required")
	}

	outletID := ctx.Params("outletId")
	if !ownsOutlet(scope, outletID) {
		return response.NotFound(ctx, "outlet not found")
	}

	rating, err := c.repo.GetOutletRating(ctx.Context(), scope, outletID)
	if err != nil {
		return response.InternalError(ctx, "could not fetch outlet rating")
	}

	return response.Success(ctx, rating)
}

// GetOutletReviews handles GET /ratings/outlets/:outletId/reviews.
func (c *VendorSupportController) GetOutletReviews(ctx *fiber.Ctx) error {
	scope, ok := c.scopeFor(ctx)
	if !ok {
		return response.Unauthorized(ctx, "authentication required")
	}

	outletID := ctx.Params("outletId")
	if !ownsOutlet(scope, outletID) {
		return response.NotFound(ctx, "outlet not found")
	}

	reviews, err := c.repo.GetOutletReviews(ctx.Context(), scope, outletID)
	if err != nil {
		return response.InternalError(ctx, "could not fetch outlet reviews")
	}

	return response.Success(ctx, reviews)
}
