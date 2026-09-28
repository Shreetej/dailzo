package repository

import (
	"context"
	"crypto/rand"
	"dailzo/models"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeliveryVendorRepository manages delivery fleet/agency profiles and the
// individual riders registered under them via invite code.
type DeliveryVendorRepository struct {
	db *pgxpool.Pool
}

func NewDeliveryVendorRepository(db *pgxpool.Pool) *DeliveryVendorRepository {
	return &DeliveryVendorRepository{db: db}
}

// queryRower is satisfied by both *pgxpool.Pool and pgx.Tx, so
// GenerateInviteCode can run inside registration_repository.go's onboarding
// transaction or standalone.
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// GenerateInviteCode returns a short, unique, human-shareable code (e.g.
// "FLT-7K3P9Q4R") a delivery vendor shares with prospective riders, retrying
// on the astronomically rare collision.
func GenerateInviteCode(ctx context.Context, q queryRower) (string, error) {
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	for attempt := 0; attempt < 5; attempt++ {
		buf := make([]byte, 5)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to generate invite code: %w", err)
		}
		code := "FLT-" + strings.ToUpper(enc.EncodeToString(buf))[:8]

		var exists bool
		if err := q.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM delivery_vendor_profiles WHERE invite_code = $1)`, code,
		).Scan(&exists); err != nil {
			return "", fmt.Errorf("failed to check invite code uniqueness: %w", err)
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("failed to generate a unique invite code after several attempts")
}

func (r *DeliveryVendorRepository) GetProfileByUserID(ctx context.Context, userID string) (*models.DeliveryVendorProfile, error) {
	query := `
		SELECT id, user_id, company_name, owner_name, email, phone, address, city,
			pincode, gst_number, invite_code, kyc_status, is_active, rating,
			created_on, last_updated_on
		FROM delivery_vendor_profiles
		WHERE user_id = $1`

	var p models.DeliveryVendorProfile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&p.ID, &p.UserID, &p.CompanyName, &p.OwnerName, &p.Email, &p.Phone,
		&p.Address, &p.City, &p.Pincode, &p.GSTNumber, &p.InviteCode,
		&p.KYCStatus, &p.IsActive, &p.Rating, &p.CreatedOn, &p.LastUpdatedOn,
	)
	if err == pgx.ErrNoRows {
		return nil, errors.New("delivery vendor profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery vendor profile: %w", err)
	}
	return &p, nil
}

// GetProfileByInviteCode resolves the fleet a rider is registering under.
// Only an ACTIVE (platform-approved) fleet can be joined — an invite code
// from a still-pending or rejected fleet application isn't usable yet.
func (r *DeliveryVendorRepository) GetProfileByInviteCode(ctx context.Context, code string) (*models.DeliveryVendorProfile, error) {
	query := `
		SELECT id, user_id, company_name, owner_name, email, phone, address, city,
			pincode, gst_number, invite_code, kyc_status, is_active, rating,
			created_on, last_updated_on
		FROM delivery_vendor_profiles
		WHERE invite_code = $1`

	var p models.DeliveryVendorProfile
	err := r.db.QueryRow(ctx, query, code).Scan(
		&p.ID, &p.UserID, &p.CompanyName, &p.OwnerName, &p.Email, &p.Phone,
		&p.Address, &p.City, &p.Pincode, &p.GSTNumber, &p.InviteCode,
		&p.KYCStatus, &p.IsActive, &p.Rating, &p.CreatedOn, &p.LastUpdatedOn,
	)
	if err == pgx.ErrNoRows {
		return nil, errors.New("invite code not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to look up invite code: %w", err)
	}
	return &p, nil
}

// queryRidersByCondition is shared by GetPendingRiders/GetActiveRiders/
// GetSuspendedRiders — same SELECT+scan shape, ownership-scoped by
// vendorUserID (never a caller-supplied vendor id), differing only in which
// dp.* rows qualify.
func (r *DeliveryVendorRepository) queryRidersByCondition(ctx context.Context, vendorUserID, extraWhere string) ([]models.DeliveryProfile, error) {
	query := `
		SELECT dp.id, dp.user_id, dp.name, dp.phone, dp.city, dp.vehicle_type,
			dp.vehicle_number, dp.license_number, dp.kyc_status, dp.rating,
			dp.total_trips, dp.is_online, dp.vendor_id, dp.is_active,
			dp.created_on, dp.last_updated_on
		FROM delivery_profiles dp
		JOIN delivery_vendor_profiles dv ON dp.vendor_id = dv.id
		WHERE dv.user_id = $1 AND ` + extraWhere + `
		ORDER BY dp.created_on ASC`

	rows, err := r.db.Query(ctx, query, vendorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query riders: %w", err)
	}
	defer rows.Close()

	riders := []models.DeliveryProfile{}
	for rows.Next() {
		var p models.DeliveryProfile
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Name, &p.Phone, &p.City, &p.VehicleType,
			&p.VehicleNumber, &p.LicenseNumber, &p.KYCStatus, &p.Rating,
			&p.TotalTrips, &p.IsOnline, &p.VendorID, &p.IsActive,
			&p.CreatedOn, &p.LastUpdatedOn,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rider: %w", err)
		}
		riders = append(riders, p)
	}
	return riders, rows.Err()
}

// GetPendingRiders lists riders awaiting THIS vendor's own FIRST review.
func (r *DeliveryVendorRepository) GetPendingRiders(ctx context.Context, vendorUserID string) ([]models.DeliveryProfile, error) {
	return r.queryRidersByCondition(ctx, vendorUserID, "dp.is_active = false AND dp.kyc_status = 'pending'")
}

// GetActiveRiders lists riders currently approved and operational under this
// vendor — the pool a vendor can choose to suspend.
func (r *DeliveryVendorRepository) GetActiveRiders(ctx context.Context, vendorUserID string) ([]models.DeliveryProfile, error) {
	return r.queryRidersByCondition(ctx, vendorUserID, "dp.is_active = true")
}

// GetSuspendedRiders lists riders this vendor previously approved and then
// suspended (kyc_status stays 'verified' — distinct from a rider rejected at
// first review, whose kyc_status is 'rejected') — the pool a vendor can
// choose to reinstate.
func (r *DeliveryVendorRepository) GetSuspendedRiders(ctx context.Context, vendorUserID string) ([]models.DeliveryProfile, error) {
	return r.queryRidersByCondition(ctx, vendorUserID, "dp.is_active = false AND dp.kyc_status = 'verified'")
}

// SuspendRider deactivates a currently-approved rider (kicking them offline
// immediately too) and logs the suspension in partner_suspensions —
// reusing the same table platform admin suspensions already use, scoped by
// ownership rather than an admin role.
func (r *DeliveryVendorRepository) SuspendRider(ctx context.Context, vendorUserID, riderID, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var inFlightOrders int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE delivery_person_id = $1 AND status IN ('partner_assigned', 'picked_up')`,
		riderID,
	).Scan(&inFlightOrders); err != nil {
		return fmt.Errorf("failed to check rider's in-progress deliveries: %w", err)
	}
	if inFlightOrders > 0 {
		return errors.New("rider has an in-progress delivery — reassign the order before suspending")
	}

	tag, err := tx.Exec(ctx, `
		UPDATE delivery_profiles
		SET is_active = false, is_online = false, last_updated_on = $1
		WHERE id = $2
		  AND vendor_id = (SELECT id FROM delivery_vendor_profiles WHERE user_id = $3)
		  AND is_active = true`,
		time.Now(), riderID, vendorUserID)
	if err != nil {
		return fmt.Errorf("failed to suspend rider: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("rider not found, not part of your fleet, or not currently active")
	}

	suspensionID := GetIdToRecord("SUSP")
	if _, err := tx.Exec(ctx, `
		INSERT INTO partner_suspensions (id, partner_id, partner_type, reason, suspended_by, suspended_on)
		VALUES ($1, $2, 'delivery', $3, $4, $5)`,
		suspensionID, riderID, reason, vendorUserID, time.Now(),
	); err != nil {
		return fmt.Errorf("failed to log suspension: %w", err)
	}

	return tx.Commit(ctx)
}

// ReinstateRider reactivates a rider this vendor previously suspended.
func (r *DeliveryVendorRepository) ReinstateRider(ctx context.Context, vendorUserID, riderID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE delivery_profiles
		SET is_active = true, last_updated_on = $1
		WHERE id = $2
		  AND vendor_id = (SELECT id FROM delivery_vendor_profiles WHERE user_id = $3)
		  AND is_active = false AND kyc_status = 'verified'`,
		time.Now(), riderID, vendorUserID)
	if err != nil {
		return fmt.Errorf("failed to reinstate rider: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("rider not found, not part of your fleet, or not currently suspended")
	}

	if _, err := tx.Exec(ctx, `
		UPDATE partner_suspensions
		SET reinstated_on = $1, reinstated_by = $2
		WHERE partner_id = $3 AND partner_type = 'delivery' AND reinstated_on IS NULL`,
		time.Now(), vendorUserID, riderID,
	); err != nil {
		return fmt.Errorf("failed to update suspension record: %w", err)
	}

	return tx.Commit(ctx)
}

// reviewRider flips a rider's status, but ONLY if riderID's vendor_id
// actually belongs to vendorUserID — prevents one fleet from
// approving/rejecting another fleet's riders by guessing an id. Also mirrors
// the decision onto the rider's OWN vendor_registrations row (registration
// status here is 'submitted' | 'approved' | 'rejected', mapped from
// kycStatus 'verified' | 'rejected') so the rider's app — which reads its
// onboarding status from GET /vendor/registration, same as every other
// vendor type — actually reflects their fleet's decision instead of showing
// "submitted" forever.
func (r *DeliveryVendorRepository) reviewRider(ctx context.Context, vendorUserID, riderID, kycStatus string, isActive bool) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var riderUserID string
	updateQuery := `
		UPDATE delivery_profiles
		SET kyc_status = $1, is_active = $2, last_updated_on = $3
		WHERE id = $4
		  AND vendor_id = (SELECT id FROM delivery_vendor_profiles WHERE user_id = $5)
		  AND kyc_status = 'pending'
		RETURNING user_id`

	err = tx.QueryRow(ctx, updateQuery, kycStatus, isActive, time.Now(), riderID, vendorUserID).Scan(&riderUserID)
	if err == pgx.ErrNoRows {
		return errors.New("rider not found, not part of your fleet, or already reviewed")
	}
	if err != nil {
		return fmt.Errorf("failed to update rider status: %w", err)
	}

	registrationStatus := "rejected"
	if kycStatus == "verified" {
		registrationStatus = "approved"
	}
	if _, err := tx.Exec(ctx,
		`UPDATE vendor_registrations SET status = $1, last_updated_on = $2 WHERE user_id = $3`,
		registrationStatus, time.Now(), riderUserID,
	); err != nil {
		return fmt.Errorf("failed to sync rider's vendor registration status: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *DeliveryVendorRepository) ApproveRider(ctx context.Context, vendorUserID, riderID string) error {
	return r.reviewRider(ctx, vendorUserID, riderID, "verified", true)
}

func (r *DeliveryVendorRepository) RejectRider(ctx context.Context, vendorUserID, riderID string) error {
	return r.reviewRider(ctx, vendorUserID, riderID, "rejected", false)
}

// GetComplaints lists complaints on orders THIS vendor's own riders
// delivered — the full order context (not filtered to delivery-specific
// complaint_type only), since a fleet reasonably wants visibility into any
// complaint on an order their rider carried, even where fault may lie
// elsewhere (e.g. food_quality). Ownership is enforced by the join itself:
// a complaint only surfaces here if its order's delivery_person_id resolves
// to one of THIS vendor's riders.
func (r *DeliveryVendorRepository) GetComplaints(ctx context.Context, vendorUserID string) ([]models.Complaint, error) {
	query := `
		SELECT c.id, c.order_id, c.user_id, c.complaint_type, c.description,
			c.status, c.priority, COALESCE(c.culprit, ''), COALESCE(c.reason_code, ''),
			COALESCE(c.confidence, 0), c.evidence_timeline, c.created_on, c.resolved_on,
			c.resolved_by, COALESCE(c.resolution_notes, ''), c.refund_amount,
			COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), '') as customer_name,
			COALESCE(u.mobileno, '') as customer_phone,
			COALESCE(o.total_amount, 0) as order_amount,
			dp.name as delivery_person_name
		FROM complaints c
		JOIN orders o ON c.order_id = o.id
		JOIN delivery_profiles dp ON o.delivery_person_id = dp.user_id
		JOIN delivery_vendor_profiles dv ON dp.vendor_id = dv.id
		LEFT JOIN users u ON c.user_id = u.id
		WHERE dv.user_id = $1
		ORDER BY
			CASE c.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
			c.created_on DESC`

	rows, err := r.db.Query(ctx, query, vendorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query complaints: %w", err)
	}
	defer rows.Close()

	complaints := []models.Complaint{}
	for rows.Next() {
		var c models.Complaint
		if err := rows.Scan(
			&c.ID, &c.OrderID, &c.UserID, &c.ComplaintType, &c.Description,
			&c.Status, &c.Priority, &c.Culprit, &c.ReasonCode, &c.Confidence,
			&c.EvidenceTimeline, &c.CreatedOn, &c.ResolvedOn, &c.ResolvedBy,
			&c.ResolutionNotes, &c.RefundAmount, &c.CustomerName,
			&c.CustomerPhone, &c.OrderAmount, &c.DeliveryPersonName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan complaint: %w", err)
		}
		complaints = append(complaints, c)
	}
	return complaints, rows.Err()
}

// isVendorsComplaint verifies complaintID's order was delivered by one of
// vendorUserID's own riders — the ownership gate ResolveComplaint/
// EscalateComplaint/UpdateComplaintNotes share.
func (r *DeliveryVendorRepository) isVendorsComplaint(ctx context.Context, vendorUserID, complaintID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM complaints c
			JOIN orders o ON c.order_id = o.id
			JOIN delivery_profiles dp ON o.delivery_person_id = dp.user_id
			JOIN delivery_vendor_profiles dv ON dp.vendor_id = dv.id
			WHERE c.id = $1 AND dv.user_id = $2
		)`, complaintID, vendorUserID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to verify complaint ownership: %w", err)
	}
	return exists, nil
}

// appendNote is a poor-man's activity log: complaints.resolution_notes is a
// single TEXT field, not a notes table, so each update appends a timestamped
// line rather than overwriting the previous one.
func appendNote(prefix, note string, at time.Time) string {
	return fmt.Sprintf("[%s] %s: %s", at.Format("2006-01-02 15:04"), prefix, note)
}

// ResolveComplaint marks a complaint resolved with the vendor's notes. This
// intentionally never sets refund_amount — refund authorization stays
// admin-only (see AdminRepository.ResolveComplaint); a fleet can resolve the
// service issue (e.g. retrain the rider, apologize to the customer) without
// touching money.
func (r *DeliveryVendorRepository) ResolveComplaint(ctx context.Context, vendorUserID, complaintID, notes string) error {
	ok, err := r.isVendorsComplaint(ctx, vendorUserID, complaintID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("complaint not found or not related to your fleet's deliveries")
	}

	now := time.Now()
	var existingNotes string
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(resolution_notes, '') FROM complaints WHERE id = $1`, complaintID).Scan(&existingNotes)
	combined := existingNotes
	if combined != "" {
		combined += "\n"
	}
	combined += appendNote("Resolved by delivery fleet", notes, now)

	tag, err := r.db.Exec(ctx, `
		UPDATE complaints
		SET status = 'resolved', resolved_on = $1, resolved_by = $2, resolution_notes = $3, last_updated_on = $1
		WHERE id = $4 AND status != 'resolved'`,
		now, vendorUserID, combined, complaintID)
	if err != nil {
		return fmt.Errorf("failed to resolve complaint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("complaint not found or already resolved")
	}
	return nil
}

// EscalateComplaint hands a complaint off to platform admin — for issues a
// fleet can't or shouldn't resolve itself (e.g. it needs a refund decision,
// or a policy call above the fleet's authority). Bumping priority to
// 'critical' surfaces it at the top of admin's existing GetComplaints
// ordering without requiring any change on the admin side.
func (r *DeliveryVendorRepository) EscalateComplaint(ctx context.Context, vendorUserID, complaintID, notes string) error {
	ok, err := r.isVendorsComplaint(ctx, vendorUserID, complaintID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("complaint not found or not related to your fleet's deliveries")
	}

	now := time.Now()
	var existingNotes string
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(resolution_notes, '') FROM complaints WHERE id = $1`, complaintID).Scan(&existingNotes)
	combined := existingNotes
	if combined != "" {
		combined += "\n"
	}
	combined += appendNote("Escalated to admin by delivery fleet", notes, now)

	tag, err := r.db.Exec(ctx, `
		UPDATE complaints
		SET status = 'escalated', priority = 'critical', resolution_notes = $1, last_updated_on = $2
		WHERE id = $3 AND status NOT IN ('resolved', 'closed')`,
		combined, now, complaintID)
	if err != nil {
		return fmt.Errorf("failed to escalate complaint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("complaint not found or already closed")
	}
	return nil
}

// UpdateComplaintNotes lets a fleet log an investigation update without
// closing the complaint — moves 'open' to 'investigating' so it's visible
// that someone picked it up, but never touches resolved/closed/escalated
// complaints (those need Resolve/re-open, not a passive note).
func (r *DeliveryVendorRepository) UpdateComplaintNotes(ctx context.Context, vendorUserID, complaintID, note string) error {
	ok, err := r.isVendorsComplaint(ctx, vendorUserID, complaintID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("complaint not found or not related to your fleet's deliveries")
	}

	now := time.Now()
	var existingNotes, status string
	err = r.db.QueryRow(ctx, `SELECT COALESCE(resolution_notes, ''), status FROM complaints WHERE id = $1`, complaintID).Scan(&existingNotes, &status)
	if err != nil {
		return fmt.Errorf("failed to load complaint: %w", err)
	}
	if status == "resolved" || status == "closed" || status == "escalated" {
		return errors.New("complaint is already resolved, closed, or escalated")
	}

	combined := existingNotes
	if combined != "" {
		combined += "\n"
	}
	combined += appendNote("Update from delivery fleet", note, now)

	newStatus := status
	if status == "open" {
		newStatus = "investigating"
	}

	_, err = r.db.Exec(ctx, `
		UPDATE complaints SET status = $1, resolution_notes = $2, last_updated_on = $3 WHERE id = $4`,
		newStatus, combined, now, complaintID)
	if err != nil {
		return fmt.Errorf("failed to update complaint notes: %w", err)
	}
	return nil
}
