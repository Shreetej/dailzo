package repository

import (
	"context"
	"dailzo/models"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminRepository handles admin-related data operations
type AdminRepository struct {
	db *pgxpool.Pool
}

// NewAdminRepository creates a new admin repository
func NewAdminRepository(db *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{db: db}
}

// GetApprovals gets all pending approvals
func (r *AdminRepository) GetApprovals(ctx context.Context) ([]models.Approval, error) {
	// COALESCE(notes, '') — models.Approval.Notes is a plain (non-pointer)
	// string, but freshly-queued approvals (registration_repository.go
	// queueApproval) leave notes NULL until an admin reviews them; scanning
	// NULL into *string fails, which silently 500'd this entire endpoint the
	// first time the approvals table ever had a real row in it.
	query := `
		SELECT id, entity_type, entity_id, status, submitted_at, reviewed_at,
			reviewed_by, COALESCE(notes, ''), documents
		FROM approvals
		WHERE status = 'pending'
		ORDER BY submitted_at ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query approvals: %w", err)
	}
	defer rows.Close()

	var approvals []models.Approval
	for rows.Next() {
		var approval models.Approval
		err := rows.Scan(
			&approval.ID, &approval.EntityType, &approval.EntityID,
			&approval.Status, &approval.SubmittedAt, &approval.ReviewedAt,
			&approval.ReviewedBy, &approval.Notes, &approval.Documents,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}

		// Fetch entity details based on type
		r.enrichApprovalWithEntityDetails(ctx, &approval)

		approvals = append(approvals, approval)
	}

	return approvals, nil
}

// enrichApprovalWithEntityDetails adds entity name/phone/city to approval
func (r *AdminRepository) enrichApprovalWithEntityDetails(ctx context.Context, approval *models.Approval) {
	switch approval.EntityType {
	case "delivery":
		var name, phone, city string
		query := `SELECT name, phone, city FROM delivery_profiles WHERE id = $1`
		r.db.QueryRow(ctx, query, approval.EntityID).Scan(&name, &phone, &city)
		approval.EntityName = name
		approval.EntityPhone = phone
		approval.EntityCity = city
	case "grocery":
		var name, phone, city string
		query := `SELECT store_name, phone, city FROM grocery_profiles WHERE id = $1`
		r.db.QueryRow(ctx, query, approval.EntityID).Scan(&name, &phone, &city)
		approval.EntityName = name
		approval.EntityPhone = phone
		approval.EntityCity = city
	case "restaurant":
		var name, phone string
		query := `SELECT name, phone_number FROM restaurants WHERE id = $1`
		r.db.QueryRow(ctx, query, approval.EntityID).Scan(&name, &phone)
		approval.EntityName = name
		approval.EntityPhone = phone
	case "delivery_vendor":
		var name, phone, city string
		query := `SELECT company_name, phone, city FROM delivery_vendor_profiles WHERE id = $1`
		r.db.QueryRow(ctx, query, approval.EntityID).Scan(&name, &phone, &city)
		approval.EntityName = name
		approval.EntityPhone = phone
		approval.EntityCity = city
	}
}

// Approve approves a pending approval
func (r *AdminRepository) Approve(ctx context.Context, id, notes, reviewerID string) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get approval details first
	var entityType, entityID string
	query := `SELECT entity_type, entity_id FROM approvals WHERE id = $1 AND status = 'pending'`
	err = tx.QueryRow(ctx, query, id).Scan(&entityType, &entityID)
	if err == pgx.ErrNoRows {
		return errors.New("approval not found or already processed")
	}
	if err != nil {
		return fmt.Errorf("failed to get approval: %w", err)
	}

	// Update approval status
	updateQuery := `
		UPDATE approvals
		SET status = 'approved', reviewed_at = $1, reviewed_by = $2, notes = $3,
			last_updated_on = $1
		WHERE id = $4`

	now := time.Now()
	_, err = tx.Exec(ctx, updateQuery, now, reviewerID, notes, id)
	if err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	// Update entity KYC status based on type
	var entityQuery string
	switch entityType {
	case "delivery":
		entityQuery = `UPDATE delivery_profiles SET kyc_status = 'verified', last_updated_on = $1 WHERE id = $2`
	case "grocery":
		entityQuery = `UPDATE grocery_profiles SET kyc_status = 'verified', is_active = true, last_updated_on = $1 WHERE id = $2`
	case "restaurant":
		entityQuery = `UPDATE restaurants SET kyc_status = 'verified', is_active = true, last_updated_on = $1 WHERE id = $2`
	case "delivery_vendor":
		entityQuery = `UPDATE delivery_vendor_profiles SET kyc_status = 'verified', is_active = true, last_updated_on = $1 WHERE id = $2`
	}

	if entityQuery != "" {
		_, err = tx.Exec(ctx, entityQuery, now, entityID)
		if err != nil {
			return fmt.Errorf("failed to update entity status: %w", err)
		}
	}

	// Mirror the decision onto vendor_registrations so the vendor app (which
	// reads its onboarding status from GetVendorRegistration, not from
	// grocery_profiles/restaurants/delivery_vendor_profiles directly) can
	// tell the user they're approved. entity_id here is the profile row just
	// updated above, not a user id, so resolve the owning user first.
	if entityType == "grocery" || entityType == "restaurant" || entityType == "delivery_vendor" {
		if err := r.syncVendorRegistrationStatus(ctx, tx, entityType, entityID, "approved", now); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// syncVendorRegistrationStatus resolves the user_id that owns the given
// grocery_profiles/restaurants/delivery_vendor_profiles row and updates
// their vendor_registrations status to match the approval decision. A
// missing vendor_registrations row (e.g. the profile was created some other
// way) is not an error — there's simply nothing to sync.
func (r *AdminRepository) syncVendorRegistrationStatus(ctx context.Context, tx pgx.Tx, entityType, entityID, status string, now time.Time) error {
	var ownerUserID string
	var err error
	switch entityType {
	case "grocery":
		err = tx.QueryRow(ctx, `SELECT user_id FROM grocery_profiles WHERE id = $1`, entityID).Scan(&ownerUserID)
	case "restaurant":
		err = tx.QueryRow(ctx, `SELECT created_by FROM restaurants WHERE id = $1`, entityID).Scan(&ownerUserID)
	case "delivery_vendor":
		err = tx.QueryRow(ctx, `SELECT user_id FROM delivery_vendor_profiles WHERE id = $1`, entityID).Scan(&ownerUserID)
	}
	if err == pgx.ErrNoRows || ownerUserID == "" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to resolve registration owner: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE vendor_registrations SET status = $1, last_updated_on = $2 WHERE user_id = $3`,
		status, now, ownerUserID)
	if err != nil {
		return fmt.Errorf("failed to sync vendor registration status: %w", err)
	}
	return nil
}

// Reject rejects a pending approval
func (r *AdminRepository) Reject(ctx context.Context, id, notes, reviewerID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get approval details first
	var entityType, entityID string
	query := `SELECT entity_type, entity_id FROM approvals WHERE id = $1 AND status = 'pending'`
	err = tx.QueryRow(ctx, query, id).Scan(&entityType, &entityID)
	if err == pgx.ErrNoRows {
		return errors.New("approval not found or already processed")
	}
	if err != nil {
		return fmt.Errorf("failed to get approval: %w", err)
	}

	// Update approval status
	updateQuery := `
		UPDATE approvals
		SET status = 'rejected', reviewed_at = $1, reviewed_by = $2, notes = $3,
			last_updated_on = $1
		WHERE id = $4`

	now := time.Now()
	_, err = tx.Exec(ctx, updateQuery, now, reviewerID, notes, id)
	if err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	// Update entity KYC status based on type
	var entityQuery string
	switch entityType {
	case "delivery":
		entityQuery = `UPDATE delivery_profiles SET kyc_status = 'rejected', last_updated_on = $1 WHERE id = $2`
	case "grocery":
		entityQuery = `UPDATE grocery_profiles SET kyc_status = 'rejected', last_updated_on = $1 WHERE id = $2`
	case "restaurant":
		entityQuery = `UPDATE restaurants SET kyc_status = 'rejected', last_updated_on = $1 WHERE id = $2`
	case "delivery_vendor":
		entityQuery = `UPDATE delivery_vendor_profiles SET kyc_status = 'rejected', last_updated_on = $1 WHERE id = $2`
	}

	if entityQuery != "" {
		_, err = tx.Exec(ctx, entityQuery, now, entityID)
		if err != nil {
			return fmt.Errorf("failed to update entity status: %w", err)
		}
	}

	if entityType == "grocery" || entityType == "restaurant" || entityType == "delivery_vendor" {
		if err := r.syncVendorRegistrationStatus(ctx, tx, entityType, entityID, "rejected", now); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetPartners gets all partners (delivery, grocery, restaurant, delivery_vendor)
func (r *AdminRepository) GetPartners(ctx context.Context) ([]models.Partner, error) {
	var partners []models.Partner

	// Get delivery partners
	deliveryQuery := `
		SELECT id, 'delivery' as type, name, phone, '' as email, city,
			CASE WHEN is_online THEN 'active' ELSE 'inactive' END as status,
			kyc_status, rating, 0 as total_orders, total_trips, created_on
		FROM delivery_profiles
		ORDER BY created_on DESC`

	rows, err := r.db.Query(ctx, deliveryQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query delivery partners: %w", err)
	}

	for rows.Next() {
		var p models.Partner
		err := rows.Scan(
			&p.ID, &p.Type, &p.Name, &p.Phone, &p.Email, &p.City,
			&p.Status, &p.KYCStatus, &p.Rating, &p.TotalOrders,
			&p.TotalTrips, &p.JoinedOn,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan delivery partner: %w", err)
		}
		partners = append(partners, p)
	}
	rows.Close()

	// Get grocery partners
	groceryQuery := `
		SELECT id, 'grocery' as type, store_name as name, phone, email, city,
			CASE WHEN is_active THEN 'active' ELSE 'inactive' END as status,
			kyc_status, rating, total_orders, 0 as total_trips, created_on
		FROM grocery_profiles
		ORDER BY created_on DESC`

	rows, err = r.db.Query(ctx, groceryQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query grocery partners: %w", err)
	}

	for rows.Next() {
		var p models.Partner
		err := rows.Scan(
			&p.ID, &p.Type, &p.Name, &p.Phone, &p.Email, &p.City,
			&p.Status, &p.KYCStatus, &p.Rating, &p.TotalOrders,
			&p.TotalTrips, &p.JoinedOn,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan grocery partner: %w", err)
		}
		partners = append(partners, p)
	}
	rows.Close()

	// Get delivery vendor (fleet/agency) partners
	deliveryVendorQuery := `
		SELECT id, 'delivery_vendor' as type, company_name as name, phone, email, city,
			CASE WHEN is_active THEN 'active' ELSE 'inactive' END as status,
			kyc_status, rating, 0 as total_orders, 0 as total_trips, created_on
		FROM delivery_vendor_profiles
		ORDER BY created_on DESC`

	rows, err = r.db.Query(ctx, deliveryVendorQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query delivery vendor partners: %w", err)
	}

	for rows.Next() {
		var p models.Partner
		err := rows.Scan(
			&p.ID, &p.Type, &p.Name, &p.Phone, &p.Email, &p.City,
			&p.Status, &p.KYCStatus, &p.Rating, &p.TotalOrders,
			&p.TotalTrips, &p.JoinedOn,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan delivery vendor partner: %w", err)
		}
		partners = append(partners, p)
	}
	rows.Close()

	// Check for suspensions
	for i := range partners {
		r.checkPartnerSuspension(ctx, &partners[i])
	}

	return partners, nil
}

// checkPartnerSuspension checks if a partner is suspended
func (r *AdminRepository) checkPartnerSuspension(ctx context.Context, partner *models.Partner) {
	query := `
		SELECT reason, suspended_on
		FROM partner_suspensions
		WHERE partner_id = $1 AND partner_type = $2 AND reinstated_on IS NULL
		LIMIT 1`

	var reason string
	var suspendedOn time.Time
	err := r.db.QueryRow(ctx, query, partner.ID, partner.Type).Scan(&reason, &suspendedOn)
	if err == nil {
		partner.IsSuspended = true
		partner.SuspendedOn = &suspendedOn
		partner.SuspendReason = reason
		partner.Status = "suspended"
	}
}

// SuspendPartner suspends a partner
func (r *AdminRepository) SuspendPartner(ctx context.Context, id, partnerType, reason, adminID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create suspension record
	suspensionID := GetIdToRecord("SUSP")
	insertQuery := `
		INSERT INTO partner_suspensions (id, partner_id, partner_type, reason, suspended_by, suspended_on)
		VALUES ($1, $2, $3, $4, $5, $6)`

	now := time.Now()
	_, err = tx.Exec(ctx, insertQuery, suspensionID, id, partnerType, reason, adminID, now)
	if err != nil {
		return fmt.Errorf("failed to create suspension: %w", err)
	}

	// Update partner status based on type. "delivery" also clears is_active
	// (not just is_online) — is_online is merely "currently on shift", which
	// a suspended rider could otherwise flip back on and keep receiving jobs
	// through, since PostDeliveryAvailableClaim gates on is_active, not
	// is_online.
	var updateQuery string
	switch partnerType {
	case "delivery":
		updateQuery = `UPDATE delivery_profiles SET is_online = false, is_active = false, last_updated_on = $1 WHERE id = $2`
	case "grocery":
		updateQuery = `UPDATE grocery_profiles SET is_active = false, last_updated_on = $1 WHERE id = $2`
	case "restaurant":
		updateQuery = `UPDATE restaurants SET is_active = false, last_updated_on = $1 WHERE id = $2`
	case "delivery_vendor":
		updateQuery = `UPDATE delivery_vendor_profiles SET is_active = false, last_updated_on = $1 WHERE id = $2`
	}

	if updateQuery != "" {
		_, err = tx.Exec(ctx, updateQuery, now, id)
		if err != nil {
			return fmt.Errorf("failed to update partner status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ResumePartner reinstates a suspended partner: closes out the open
// suspension record (reinstated_on/reinstated_by, mirroring the fields
// SuspendPartner leaves for it) and flips the partner back to active. A
// reinstated rider's is_online stays false — they go through the normal
// go-online flow rather than being auto-placed back on shift.
func (r *AdminRepository) ResumePartner(ctx context.Context, id, partnerType, adminID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	tag, err := tx.Exec(ctx, `
		UPDATE partner_suspensions
		SET reinstated_on = $1, reinstated_by = $2
		WHERE partner_id = $3 AND partner_type = $4 AND reinstated_on IS NULL`,
		now, adminID, id, partnerType)
	if err != nil {
		return fmt.Errorf("failed to close suspension record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("partner is not currently suspended")
	}

	var updateQuery string
	switch partnerType {
	case "delivery":
		updateQuery = `UPDATE delivery_profiles SET is_active = true, last_updated_on = $1 WHERE id = $2`
	case "grocery":
		updateQuery = `UPDATE grocery_profiles SET is_active = true, last_updated_on = $1 WHERE id = $2`
	case "restaurant":
		updateQuery = `UPDATE restaurants SET is_active = true, last_updated_on = $1 WHERE id = $2`
	case "delivery_vendor":
		updateQuery = `UPDATE delivery_vendor_profiles SET is_active = true, last_updated_on = $1 WHERE id = $2`
	}

	if updateQuery != "" {
		_, err = tx.Exec(ctx, updateQuery, now, id)
		if err != nil {
			return fmt.Errorf("failed to update partner status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetComplaints gets all complaints
func (r *AdminRepository) GetComplaints(ctx context.Context) ([]models.Complaint, error) {
	// culprit/reason_code/confidence/resolution_notes are all nullable
	// columns with no default (nothing currently populates the first three —
	// there's no automated fraud-attribution service yet), and
	// customer_name/customer_phone go NULL whenever the LEFT JOIN misses.
	// models.Complaint scans these into plain (non-pointer) string/float64
	// fields, so without COALESCE this crashes the instant a real complaint
	// exists — untriggered until now only because the table was empty.
	query := `
		SELECT c.id, c.order_id, c.user_id, c.complaint_type, c.description,
			c.status, c.priority, COALESCE(c.culprit, ''), COALESCE(c.reason_code, ''),
			COALESCE(c.confidence, 0), c.evidence_timeline, c.created_on, c.resolved_on,
			c.resolved_by, COALESCE(c.resolution_notes, ''), c.refund_amount,
			COALESCE(u.first_name || ' ' || COALESCE(u.last_name, ''), '') as customer_name,
			COALESCE(u.mobileno, '') as customer_phone,
			COALESCE(o.total_amount, 0) as order_amount
		FROM complaints c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN orders o ON c.order_id = o.id
		ORDER BY
			CASE c.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
			c.created_on DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query complaints: %w", err)
	}
	defer rows.Close()

	var complaints []models.Complaint
	for rows.Next() {
		var c models.Complaint
		err := rows.Scan(
			&c.ID, &c.OrderID, &c.UserID, &c.ComplaintType, &c.Description,
			&c.Status, &c.Priority, &c.Culprit, &c.ReasonCode, &c.Confidence,
			&c.EvidenceTimeline, &c.CreatedOn, &c.ResolvedOn, &c.ResolvedBy,
			&c.ResolutionNotes, &c.RefundAmount, &c.CustomerName,
			&c.CustomerPhone, &c.OrderAmount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan complaint: %w", err)
		}
		complaints = append(complaints, c)
	}

	return complaints, nil
}

// ResolveComplaint resolves a complaint
func (r *AdminRepository) ResolveComplaint(ctx context.Context, id, notes string, refundAmount float64, adminID string) error {
	query := `
		UPDATE complaints
		SET status = 'resolved', resolved_on = $1, resolved_by = $2,
			resolution_notes = $3, refund_amount = $4, last_updated_on = $1
		WHERE id = $5 AND status != 'resolved'`

	now := time.Now()
	result, err := r.db.Exec(ctx, query, now, adminID, notes, refundAmount, id)
	if err != nil {
		return fmt.Errorf("failed to resolve complaint: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("complaint not found or already resolved")
	}

	return nil
}

// GetInvestigation gets investigation details for a complaint
func (r *AdminRepository) GetInvestigation(ctx context.Context, id string) (*models.AdminComplaintInvestigation, error) {
	query := `
		SELECT id, order_id, culprit, reason_code, confidence, evidence_timeline
		FROM complaints
		WHERE id = $1`

	var inv models.AdminComplaintInvestigation
	var evidenceJSON *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&inv.ComplaintID, &inv.OrderID, &inv.Culprit, &inv.ReasonCode,
		&inv.Confidence, &evidenceJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.New("complaint not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get investigation: %w", err)
	}

	// Parse evidence timeline
	inv.EvidenceTimeline = []string{}
	// In production, you'd parse the JSON here

	return &inv, nil
}

// GetPlatformKpis gets platform-wide KPIs
func (r *AdminRepository) GetPlatformKpis(ctx context.Context) (*models.AdminKpis, error) {
	kpis := &models.AdminKpis{}

	// Orders today
	ordersQuery := `
		SELECT
			COUNT(*) as total_orders,
			COALESCE(SUM(total_amount), 0) as total_revenue,
			COALESCE(AVG(total_amount), 0) as avg_order_value
		FROM orders
		WHERE DATE(order_date) = CURRENT_DATE`

	r.db.QueryRow(ctx, ordersQuery).Scan(
		&kpis.TotalOrdersToday, &kpis.TotalRevenueToday, &kpis.AvgOrderValue,
	)

	// Yesterday's orders for growth calculation
	var yesterdayOrders int
	yesterdayQuery := `SELECT COUNT(*) FROM orders WHERE DATE(order_date) = CURRENT_DATE - INTERVAL '1 day'`
	r.db.QueryRow(ctx, yesterdayQuery).Scan(&yesterdayOrders)

	if yesterdayOrders > 0 {
		kpis.OrderGrowthPct = float64(kpis.TotalOrdersToday-yesterdayOrders) / float64(yesterdayOrders) * 100
	}

	// Active delivery partners
	deliveryQuery := `SELECT COUNT(*) FROM delivery_profiles WHERE is_online = true`
	r.db.QueryRow(ctx, deliveryQuery).Scan(&kpis.ActiveDeliveryPartners)

	// Active grocery stores
	groceryQuery := `SELECT COUNT(*) FROM grocery_profiles WHERE is_active = true`
	r.db.QueryRow(ctx, groceryQuery).Scan(&kpis.ActiveGroceryStores)

	// Open complaints
	complaintsQuery := `SELECT COUNT(*) FROM complaints WHERE status IN ('open', 'investigating')`
	r.db.QueryRow(ctx, complaintsQuery).Scan(&kpis.OpenComplaints)

	// Pending approvals
	approvalsQuery := `SELECT COUNT(*) FROM approvals WHERE status = 'pending'`
	r.db.QueryRow(ctx, approvalsQuery).Scan(&kpis.PendingApprovals)

	// Stalled onboardings
	onboardingQuery := `SELECT COUNT(*) FROM onboarding_leads WHERE last_active_at < CURRENT_DATE - INTERVAL '3 days'`
	r.db.QueryRow(ctx, onboardingQuery).Scan(&kpis.StalledOnboardings)

	return kpis, nil
}

// GetOnboardingLeads gets incomplete onboarding registrations
func (r *AdminRepository) GetOnboardingLeads(ctx context.Context) ([]models.OnboardingLead, error) {
	query := `
		SELECT id, entity_type, name, email, phone, city, last_step, total_steps,
			step_completed, last_active_at, days_since_start, potential_loss,
			notifications_sent, last_notification_at, source, created_at
		FROM onboarding_leads
		WHERE step_completed < total_steps
		ORDER BY potential_loss DESC, days_since_start DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query onboarding leads: %w", err)
	}
	defer rows.Close()

	var leads []models.OnboardingLead
	for rows.Next() {
		var lead models.OnboardingLead
		err := rows.Scan(
			&lead.ID, &lead.EntityType, &lead.Name, &lead.Email, &lead.Phone,
			&lead.City, &lead.LastStep, &lead.TotalSteps, &lead.StepCompleted,
			&lead.LastActiveAt, &lead.DaysSinceStart, &lead.PotentialLoss,
			&lead.NotificationsSent, &lead.LastNotificationAt, &lead.Source,
			&lead.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan onboarding lead: %w", err)
		}
		leads = append(leads, lead)
	}

	return leads, nil
}

// NotifyLead sends a notification to an onboarding lead
func (r *AdminRepository) NotifyLead(ctx context.Context, id, channel, message string) error {
	// Update notification count
	query := `
		UPDATE onboarding_leads
		SET notifications_sent = notifications_sent + 1,
			last_notification_at = $1
		WHERE id = $2`

	_, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update notification count: %w", err)
	}

	// In production, you'd integrate with SMS/Email/Push service here
	// based on the channel parameter

	return nil
}

// ordersFilterForPartner returns the SQL WHERE fragment (and its single $1
// arg) that scopes the orders table to one partner. delivery_vendor and
// delivery (individual rider) both resolve through delivery_profiles since
// orders.delivery_person_id references users.id, not delivery_profiles.id.
// Known limitation: grocery orders currently never get their restaurant_id
// populated at checkout (a pre-existing bug in storefront_repository.go's
// order creation, outside this change's scope), so grocery performance will
// read zero orders until that's fixed — the query itself is correct.
func ordersFilterForPartner(partnerType string) (whereClause string, ok bool) {
	switch partnerType {
	case "restaurant":
		return `restaurant_id = $1 AND cart_type = 'food'`, true
	case "grocery":
		return `restaurant_id = $1 AND cart_type = 'grocery'`, true
	case "delivery_vendor":
		return `delivery_person_id IN (SELECT user_id FROM delivery_profiles WHERE vendor_id = $1)`, true
	case "delivery":
		return `delivery_person_id = (SELECT user_id FROM delivery_profiles WHERE id = $1)`, true
	default:
		return "", false
	}
}

// GetPartnerPerformance returns a partner's real order/revenue trend over the
// last `days` days, plus lifetime totals, completion rate, and current
// rating — backing the admin "View Performance" action.
func (r *AdminRepository) GetPartnerPerformance(ctx context.Context, id, partnerType string, days int) (*models.PartnerPerformance, error) {
	where, ok := ordersFilterForPartner(partnerType)
	if !ok {
		return nil, fmt.Errorf("unknown partner type %q", partnerType)
	}

	perf := &models.PartnerPerformance{PartnerID: id, PartnerType: partnerType, Daily: []models.DailyStat{}}

	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*),
		       COALESCE(SUM(total_amount), 0),
		       COUNT(*) FILTER (WHERE status = 'delivered')
		FROM orders WHERE %s`, where)
	var delivered int
	if err := r.db.QueryRow(ctx, summaryQuery, id).Scan(&perf.TotalOrders, &perf.TotalRevenue, &delivered); err != nil {
		return nil, fmt.Errorf("failed to summarize partner orders: %w", err)
	}
	if perf.TotalOrders > 0 {
		perf.CompletionRate = float64(delivered) / float64(perf.TotalOrders) * 100
	}

	dailyQuery := fmt.Sprintf(`
		SELECT DATE(order_date) AS d, COUNT(*), COALESCE(SUM(total_amount), 0)
		FROM orders
		WHERE %s AND order_date >= now() - make_interval(days => $2)
		GROUP BY DATE(order_date)
		ORDER BY d`, where)
	rows, err := r.db.Query(ctx, dailyQuery, id, days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch partner daily trend: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d models.DailyStat
		var date time.Time
		if err := rows.Scan(&date, &d.Orders, &d.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan partner daily trend: %w", err)
		}
		d.Date = date.Format("2006-01-02")
		perf.Daily = append(perf.Daily, d)
	}

	var ratingTable string
	switch partnerType {
	case "restaurant":
		ratingTable = "restaurants"
	case "grocery":
		ratingTable = "grocery_profiles"
	case "delivery_vendor":
		ratingTable = "delivery_vendor_profiles"
	case "delivery":
		ratingTable = "delivery_profiles"
	}
	if ratingTable != "" {
		_ = r.db.QueryRow(ctx, fmt.Sprintf(`SELECT COALESCE(rating, 0) FROM %s WHERE id = $1`, ratingTable), id).
			Scan(&perf.Rating)
	}

	return perf, nil
}

// commissionTableForPartner returns the table (all admin-commission-eligible
// partner types: restaurant, grocery, delivery_vendor — individual riders are
// paid per-delivery, not on a commission split) holding commission_rate.
func commissionTableForPartner(partnerType string) (table string, ok bool) {
	switch partnerType {
	case "restaurant":
		return "restaurants", true
	case "grocery":
		return "grocery_profiles", true
	case "delivery_vendor":
		return "delivery_vendor_profiles", true
	default:
		return "", false
	}
}

// GetPartnerCommission reads a partner's current commission rate.
func (r *AdminRepository) GetPartnerCommission(ctx context.Context, id, partnerType string) (*models.PartnerCommission, error) {
	table, ok := commissionTableForPartner(partnerType)
	if !ok {
		return nil, fmt.Errorf("partner type %q has no commission rate", partnerType)
	}
	c := &models.PartnerCommission{PartnerID: id, PartnerType: partnerType}
	err := r.db.QueryRow(ctx, fmt.Sprintf(`SELECT commission_rate FROM %s WHERE id = $1`, table), id).
		Scan(&c.CommissionRate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commission rate: %w", err)
	}
	return c, nil
}

// SetPartnerCommission updates a partner's commission rate (0-100).
func (r *AdminRepository) SetPartnerCommission(ctx context.Context, id, partnerType string, rate float64) error {
	table, ok := commissionTableForPartner(partnerType)
	if !ok {
		return fmt.Errorf("partner type %q has no commission rate", partnerType)
	}
	if rate < 0 || rate > 100 {
		return errors.New("commission rate must be between 0 and 100")
	}
	tag, err := r.db.Exec(ctx, fmt.Sprintf(`UPDATE %s SET commission_rate = $1 WHERE id = $2`, table), rate, id)
	if err != nil {
		return fmt.Errorf("failed to update commission rate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("partner not found")
	}
	return nil
}

// GetRevenueTimeseries returns real day-by-day platform order count + revenue
// for the last `days` days, backing the admin Reports "Revenue Trend" chart
// (previously 7 hardcoded fake values).
func (r *AdminRepository) GetRevenueTimeseries(ctx context.Context, days int) ([]models.DailyStat, error) {
	query := `
		SELECT DATE(order_date) AS d, COUNT(*), COALESCE(SUM(total_amount), 0)
		FROM orders
		WHERE order_date >= now() - make_interval(days => $1)
		GROUP BY DATE(order_date)
		ORDER BY d`
	rows, err := r.db.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch revenue timeseries: %w", err)
	}
	defer rows.Close()

	out := []models.DailyStat{}
	for rows.Next() {
		var d models.DailyStat
		var date time.Time
		if err := rows.Scan(&date, &d.Orders, &d.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan revenue timeseries: %w", err)
		}
		d.Date = date.Format("2006-01-02")
		out = append(out, d)
	}
	return out, rows.Err()
}

// LogActivity logs an admin action
func (r *AdminRepository) LogActivity(ctx context.Context, log *models.AdminActivityLog) error {
	id := GetIdToRecord("ALOG")
	query := `
		INSERT INTO admin_activity_log (id, admin_id, action, entity_type, entity_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.Exec(ctx, query,
		id, log.AdminID, log.Action, log.EntityType, log.EntityID,
		log.Details, log.IPAddress, log.UserAgent, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to log activity: %w", err)
	}
	return nil
}
