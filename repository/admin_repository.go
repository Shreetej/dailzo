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
	query := `
		SELECT id, entity_type, entity_id, status, submitted_at, reviewed_at,
			reviewed_by, notes, documents
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
		entityQuery = `UPDATE restaurants SET last_updated_on = $1 WHERE id = $2`
	}

	if entityQuery != "" {
		_, err = tx.Exec(ctx, entityQuery, now, entityID)
		if err != nil {
			return fmt.Errorf("failed to update entity status: %w", err)
		}
	}

	return tx.Commit(ctx)
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
	}

	if entityQuery != "" {
		_, err = tx.Exec(ctx, entityQuery, now, entityID)
		if err != nil {
			return fmt.Errorf("failed to update entity status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetPartners gets all partners (delivery, grocery, restaurant)
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

	// Update partner status based on type
	var updateQuery string
	switch partnerType {
	case "delivery":
		updateQuery = `UPDATE delivery_profiles SET is_online = false, last_updated_on = $1 WHERE id = $2`
	case "grocery":
		updateQuery = `UPDATE grocery_profiles SET is_active = false, last_updated_on = $1 WHERE id = $2`
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
	query := `
		SELECT c.id, c.order_id, c.user_id, c.complaint_type, c.description,
			c.status, c.priority, c.culprit, c.reason_code, c.confidence,
			c.evidence_timeline, c.created_on, c.resolved_on, c.resolved_by,
			c.resolution_notes, c.refund_amount,
			u.first_name || ' ' || COALESCE(u.last_name, '') as customer_name,
			u.mobileno as customer_phone,
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
