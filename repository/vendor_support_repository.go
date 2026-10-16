package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VendorSupportRepository backs the vendor-facing complaint and outlet-rating
// screens in the Partner app. Both surfaces were built in the app against
// endpoints that never existed on the server (/complaints/{id},
// /complaints/summary, /ratings/outlets*), so every one of those screens
// failed with a 404 and rendered empty.
//
// Everything here is ownership-scoped by the caller's user_id. A vendor may
// only ever see complaints filed against their own orders and ratings left on
// their own outlets — scoping is never taken from a client-supplied path or
// query parameter.
type VendorSupportRepository struct {
	db *pgxpool.Pool
}

func NewVendorSupportRepository(db *pgxpool.Pool) *VendorSupportRepository {
	return &VendorSupportRepository{db: db}
}

// VendorScope is the set of entity ids a single vendor account owns. Restaurant
// and grocery accounts both resolve through here; orders carry restaurant_id
// (the owning business) and outlet_id (the specific branch), so a complaint or
// rating is "theirs" if it matches either set.
type VendorScope struct {
	BusinessIDs []string `json:"business_ids"`
	OutletIDs   []string `json:"outlet_ids"`
}

// IsEmpty reports whether this account owns no business at all — a user who
// signed up but never completed registration. Callers return an empty list
// rather than an error, so a half-onboarded vendor sees an empty screen
// instead of a failure.
func (s VendorScope) IsEmpty() bool {
	return len(s.BusinessIDs) == 0 && len(s.OutletIDs) == 0
}

// AllIDs is every id this account owns, business and outlet alike.
//
// Queries match it against BOTH orders.restaurant_id and orders.outlet_id
// rather than pairing each set with one column: which column holds a vendor's
// id depends on the vendor type, not on the id's origin. A restaurant's id
// lands in restaurant_id, but a grocery store's id — resolved from
// grocery_profiles, so a "business" id here — is written to outlet_id at
// checkout. Matching business ids only against restaurant_id made every
// grocery order invisible to the store that owned it.
func (s VendorScope) AllIDs() []string {
	ids := make([]string, 0, len(s.BusinessIDs)+len(s.OutletIDs))
	ids = append(ids, s.BusinessIDs...)
	ids = append(ids, s.OutletIDs...)
	return ids
}

// Owns reports whether an id names one of this account's own businesses or
// outlets. Callers use it to reject a client-supplied id that would otherwise
// widen the query past the caller's own data.
func (s VendorScope) Owns(id string) bool {
	for _, candidate := range s.BusinessIDs {
		if candidate == id {
			return true
		}
	}
	for _, candidate := range s.OutletIDs {
		if candidate == id {
			return true
		}
	}
	return false
}

// GetVendorScope resolves the caller to the business ids they own.
// restaurants are owned via created_by; grocery_profiles via user_id. Outlets
// hang off vendor_outlets.restaurant_id.
func (r *VendorSupportRepository) GetVendorScope(ctx context.Context, userID string) (VendorScope, error) {
	var scope VendorScope

	const businessQuery = `
		SELECT id FROM restaurants WHERE created_by = $1
		UNION
		SELECT id FROM grocery_profiles WHERE user_id = $1`

	rows, err := r.db.Query(ctx, businessQuery, userID)
	if err != nil {
		return scope, fmt.Errorf("failed to resolve vendor businesses: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return scope, fmt.Errorf("failed to scan vendor business id: %w", err)
		}
		scope.BusinessIDs = append(scope.BusinessIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return scope, fmt.Errorf("failed to read vendor businesses: %w", err)
	}

	if len(scope.BusinessIDs) == 0 {
		return scope, nil
	}

	outletRows, err := r.db.Query(ctx,
		`SELECT id FROM vendor_outlets WHERE restaurant_id = ANY($1)`, scope.BusinessIDs)
	if err != nil {
		return scope, fmt.Errorf("failed to resolve vendor outlets: %w", err)
	}
	defer outletRows.Close()
	for outletRows.Next() {
		var id string
		if err := outletRows.Scan(&id); err != nil {
			return scope, fmt.Errorf("failed to scan outlet id: %w", err)
		}
		scope.OutletIDs = append(scope.OutletIDs, id)
	}

	return scope, outletRows.Err()
}

// ─── Complaints ──────────────────────────────────────────────────────────────

// VendorComplaint is the Partner app's complaint shape (core/models/
// complaint_model.dart). Field names are camelCase because that model is a
// plain freezed class with no @JsonKey overrides, unlike the snake_case
// models elsewhere in the app.
type VendorComplaint struct {
	ID               string     `json:"id"`
	OrderID          string     `json:"orderId"`
	CustomerID       string     `json:"customerId"`
	CustomerName     string     `json:"customerName"`
	RestaurantID     string     `json:"restaurantId"`
	OutletID         string     `json:"outletId"`
	Subject          string     `json:"subject"`
	Description      string     `json:"description"`
	ComplaintType    string     `json:"complaintType"`
	CreatedAt        time.Time  `json:"createdAt"`
	Status           string     `json:"status"`
	RefundAmount     *string    `json:"refundAmount,omitempty"`
	ResolvedAt       *time.Time `json:"resolvedAt,omitempty"`
	ResolutionNotes  *string    `json:"resolutionNotes,omitempty"`
	InvolvedEntities []string   `json:"involvedEntities,omitempty"`
}

// VendorComplaintSummary backs the counters on the complaints screen header.
type VendorComplaintSummary struct {
	TotalComplaints    int `json:"totalComplaints"`
	PendingComplaints  int `json:"pendingComplaints"`
	ResolvedComplaints int `json:"resolvedComplaints"`
	RejectedComplaints int `json:"rejectedComplaints"`
}

// The complaints table constrains status to open/investigating/resolved/closed
// (migrations/001_new_tables.sql), but the Partner app's Complaint model and
// its whole detail-screen UI are keyed to pending/accepted/rejected/resolved.
// Rather than migrate a live table or rewrite 683 lines of screen logic, the
// two vocabularies are mapped here, in one place, in both directions.
var dbToAppStatus = map[string]string{
	"open":          "pending",
	"investigating": "accepted",
	"resolved":      "resolved",
	"closed":        "rejected",
}

var appToDBStatus = map[string]string{
	"pending":  "open",
	"accepted": "investigating",
	"resolved": "resolved",
	"rejected": "closed",
}

// complaintTypeSubjects turns the stored complaint_type enum into the short
// human-readable line the app shows as `subject`. The complaints table has no
// subject column of its own.
var complaintTypeSubjects = map[string]string{
	"delivery_delay": "Delivery delay",
	"food_quality":   "Food quality issue",
	"missing_item":   "Missing item",
	"wrong_order":    "Wrong order delivered",
	"rude_behavior":  "Staff behaviour",
	"payment_issue":  "Payment issue",
	"other":          "Other issue",
}

func subjectFor(complaintType string) string {
	if s, ok := complaintTypeSubjects[complaintType]; ok {
		return s
	}
	return "Customer complaint"
}

// complaintSelect is shared by the list and detail reads so both return
// identical shapes. COALESCE mirrors AdminRepository.GetComplaints: culprit,
// reason_code and resolution_notes are nullable with no default, and the
// customer columns go NULL whenever the LEFT JOIN misses.
const complaintSelect = `
	SELECT c.id,
	       COALESCE(c.order_id, ''),
	       COALESCE(c.user_id, ''),
	       COALESCE(NULLIF(TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')), ''), 'Customer'),
	       COALESCE(o.restaurant_id, ''),
	       COALESCE(o.outlet_id, ''),
	       COALESCE(c.complaint_type, 'other'),
	       COALESCE(c.description, ''),
	       COALESCE(c.status, 'open'),
	       c.created_on,
	       c.resolved_on,
	       c.resolution_notes,
	       c.refund_amount,
	       COALESCE(c.culprit, '')
	FROM complaints c
	LEFT JOIN users u ON c.user_id = u.id
	JOIN orders o ON c.order_id = o.id`

func (r *VendorSupportRepository) scanComplaints(rows pgx.Rows) ([]VendorComplaint, error) {
	defer rows.Close()

	complaints := []VendorComplaint{}
	for rows.Next() {
		var (
			c            VendorComplaint
			dbStatus     string
			resolvedOn   *time.Time
			notes        *string
			refundAmount *float64
			culprit      string
		)
		if err := rows.Scan(
			&c.ID, &c.OrderID, &c.CustomerID, &c.CustomerName,
			&c.RestaurantID, &c.OutletID, &c.ComplaintType, &c.Description,
			&dbStatus, &c.CreatedAt, &resolvedOn, &notes, &refundAmount, &culprit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan complaint: %w", err)
		}

		c.Subject = subjectFor(c.ComplaintType)
		if mapped, ok := dbToAppStatus[dbStatus]; ok {
			c.Status = mapped
		} else {
			c.Status = dbStatus
		}
		c.ResolvedAt = resolvedOn
		c.ResolutionNotes = notes
		if refundAmount != nil && *refundAmount > 0 {
			formatted := fmt.Sprintf("%.2f", *refundAmount)
			c.RefundAmount = &formatted
		}
		if culprit != "" {
			c.InvolvedEntities = []string{culprit}
		}

		complaints = append(complaints, c)
	}

	return complaints, rows.Err()
}

// GetComplaints lists complaints filed against this vendor's own orders.
// status is the app-side vocabulary ('pending'/'accepted'/...); outletID
// narrows to one branch and is validated against the caller's own outlets by
// the handler before it reaches here.
func (r *VendorSupportRepository) GetComplaints(ctx context.Context, scope VendorScope, status, outletID string) ([]VendorComplaint, error) {
	if scope.IsEmpty() {
		return []VendorComplaint{}, nil
	}

	args := []any{scope.AllIDs()}
	conditions := []string{"(o.restaurant_id = ANY($1) OR o.outlet_id = ANY($1))"}

	if status != "" {
		dbStatus, ok := appToDBStatus[status]
		if !ok {
			dbStatus = status
		}
		args = append(args, dbStatus)
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", len(args)))
	}
	if outletID != "" {
		args = append(args, outletID)
		conditions = append(conditions, fmt.Sprintf("o.outlet_id = $%d", len(args)))
	}

	query := complaintSelect + " WHERE " + strings.Join(conditions, " AND ") + `
		ORDER BY
			CASE c.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
			c.created_on DESC
		LIMIT 200`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query vendor complaints: %w", err)
	}
	return r.scanComplaints(rows)
}

// GetComplaintByID reads one complaint, still ownership-scoped: a vendor
// guessing another vendor's complaint id gets the same not-found as a
// complaint that doesn't exist.
func (r *VendorSupportRepository) GetComplaintByID(ctx context.Context, scope VendorScope, complaintID string) (*VendorComplaint, error) {
	if scope.IsEmpty() {
		return nil, errors.New("complaint not found")
	}

	query := complaintSelect + ` WHERE c.id = $2 AND (o.restaurant_id = ANY($1) OR o.outlet_id = ANY($1)) LIMIT 1`

	rows, err := r.db.Query(ctx, query, scope.AllIDs(), complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to query complaint: %w", err)
	}
	complaints, err := r.scanComplaints(rows)
	if err != nil {
		return nil, err
	}
	if len(complaints) == 0 {
		return nil, errors.New("complaint not found")
	}
	return &complaints[0], nil
}

// GetComplaintSummary returns the header counters for the complaints screen,
// counted over this vendor's own complaints only.
func (r *VendorSupportRepository) GetComplaintSummary(ctx context.Context, scope VendorScope) (*VendorComplaintSummary, error) {
	summary := &VendorComplaintSummary{}
	if scope.IsEmpty() {
		return summary, nil
	}

	query := `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE c.status IN ('open', 'investigating')),
		       COUNT(*) FILTER (WHERE c.status = 'resolved'),
		       COUNT(*) FILTER (WHERE c.status = 'closed')
		FROM complaints c
		JOIN orders o ON c.order_id = o.id
		WHERE o.restaurant_id = ANY($1) OR o.outlet_id = ANY($1)`

	if err := r.db.QueryRow(ctx, query, scope.AllIDs()).Scan(
		&summary.TotalComplaints, &summary.PendingComplaints,
		&summary.ResolvedComplaints, &summary.RejectedComplaints,
	); err != nil {
		return nil, fmt.Errorf("failed to summarise complaints: %w", err)
	}

	return summary, nil
}

// UpdateComplaintStatus moves a complaint through the vendor's own workflow.
// Unlike AdminRepository.ResolveComplaint (admin-only, resolve-only) this
// accepts any of the app's four states and stays ownership-scoped. refundAmount
// is only written when the vendor actually granted one.
func (r *VendorSupportRepository) UpdateComplaintStatus(
	ctx context.Context,
	scope VendorScope,
	complaintID, appStatus, notes string,
	refundAmount *float64,
	actorID string,
) error {
	if scope.IsEmpty() {
		return errors.New("complaint not found")
	}

	dbStatus, ok := appToDBStatus[appStatus]
	if !ok {
		return fmt.Errorf("invalid complaint status %q", appStatus)
	}

	// resolved_on/resolved_by only make sense for terminal states; leaving
	// them NULL for pending/accepted keeps a re-opened complaint from
	// carrying a stale resolution timestamp.
	var resolvedOn *time.Time
	var resolvedBy *string
	if dbStatus == "resolved" || dbStatus == "closed" {
		now := time.Now()
		resolvedOn = &now
		resolvedBy = &actorID
	}

	query := `
		UPDATE complaints c
		SET status = $3,
		    resolved_on = $4,
		    resolved_by = $5,
		    resolution_notes = COALESCE(NULLIF($6, ''), c.resolution_notes),
		    refund_amount = COALESCE($7, c.refund_amount),
		    last_updated_on = CURRENT_TIMESTAMP
		FROM orders o
		WHERE c.id = $2
		  AND c.order_id = o.id
		  AND (o.restaurant_id = ANY($1) OR o.outlet_id = ANY($1))`

	result, err := r.db.Exec(ctx, query,
		scope.AllIDs(), complaintID,
		dbStatus, resolvedOn, resolvedBy, notes, refundAmount,
	)
	if err != nil {
		return fmt.Errorf("failed to update complaint status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("complaint not found")
	}

	return nil
}

// ─── Outlet ratings ──────────────────────────────────────────────────────────

// OutletReview is one customer review, in the Partner app's Rating shape
// (core/models/rating_model.dart).
type OutletReview struct {
	ID           string    `json:"id"`
	OutletID     string    `json:"outletId"`
	CustomerID   string    `json:"customerId"`
	CustomerName string    `json:"customerName"`
	Rating       float64   `json:"rating"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"createdAt"`
	OrderIDs     []string  `json:"orderIds"`
}

// OutletRatingSummary is the per-outlet aggregate the ratings screen renders.
type OutletRatingSummary struct {
	OutletID      string         `json:"outletId"`
	AverageRating float64        `json:"averageRating"`
	TotalRatings  int            `json:"totalRatings"`
	Ratings       []OutletReview `json:"ratings"`
}

// ratingEntityIDs decides which ratings rows belong to a given outlet.
//
// The ratings table keys rows by (entity_type, entity_id) and the customer app
// currently writes only entity_type 'restaurant' (rate_order_screen.dart) and
// 'delivery' — nothing writes 'outlet'. So outlet-level attribution is only
// unambiguous when the business has exactly ONE outlet: every restaurant-level
// rating is necessarily that outlet's. With two or more outlets a
// restaurant-scoped rating cannot be attributed to a branch without inventing
// data, so those rows are left out and only genuine outlet-scoped ratings
// count. The ratings screen already renders a missing outlet as zero, so this
// degrades honestly rather than showing one branch's score on all of them.
func ratingEntityIDs(outletID string, scope VendorScope) []string {
	ids := []string{outletID}
	if len(scope.OutletIDs) == 1 && scope.OutletIDs[0] == outletID {
		ids = append(ids, scope.BusinessIDs...)
	}
	return ids
}

// GetOutletRatings returns one aggregate per outlet this vendor owns.
func (r *VendorSupportRepository) GetOutletRatings(ctx context.Context, scope VendorScope) ([]OutletRatingSummary, error) {
	summaries := []OutletRatingSummary{}
	for _, outletID := range scope.OutletIDs {
		summary, err := r.GetOutletRating(ctx, scope, outletID)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, *summary)
	}
	return summaries, nil
}

// GetOutletRating aggregates one outlet. An outlet with no ratings yet returns
// a zeroed summary rather than an error — that is a new outlet, not a failure.
func (r *VendorSupportRepository) GetOutletRating(ctx context.Context, scope VendorScope, outletID string) (*OutletRatingSummary, error) {
	summary := &OutletRatingSummary{OutletID: outletID, Ratings: []OutletReview{}}

	query := `
		SELECT COALESCE(AVG(rating), 0)::float8, COUNT(*)
		FROM ratings
		WHERE entity_id = ANY($1) AND entity_type IN ('outlet', 'restaurant')`

	if err := r.db.QueryRow(ctx, query, ratingEntityIDs(outletID, scope)).Scan(
		&summary.AverageRating, &summary.TotalRatings,
	); err != nil {
		return nil, fmt.Errorf("failed to aggregate outlet ratings: %w", err)
	}

	return summary, nil
}

// GetOutletReviews lists the individual reviews behind one outlet's aggregate.
func (r *VendorSupportRepository) GetOutletReviews(ctx context.Context, scope VendorScope, outletID string) ([]OutletReview, error) {
	query := `
		SELECT rt.id,
		       COALESCE(rt.user_id, ''),
		       COALESCE(NULLIF(TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')), ''), 'Customer'),
		       rt.rating::float8,
		       COALESCE(rt.comment, ''),
		       rt.created_on
		FROM ratings rt
		LEFT JOIN users u ON rt.user_id = u.id
		WHERE rt.entity_id = ANY($1) AND rt.entity_type IN ('outlet', 'restaurant')
		ORDER BY rt.created_on DESC
		LIMIT 200`

	rows, err := r.db.Query(ctx, query, ratingEntityIDs(outletID, scope))
	if err != nil {
		return nil, fmt.Errorf("failed to query outlet reviews: %w", err)
	}
	defer rows.Close()

	reviews := []OutletReview{}
	for rows.Next() {
		review := OutletReview{OutletID: outletID, OrderIDs: []string{}}
		if err := rows.Scan(
			&review.ID, &review.CustomerID, &review.CustomerName,
			&review.Rating, &review.Comment, &review.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan outlet review: %w", err)
		}
		reviews = append(reviews, review)
	}

	return reviews, rows.Err()
}
