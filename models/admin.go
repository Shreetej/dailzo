package models

import "time"

// Approval represents a KYC/onboarding approval request
type Approval struct {
	ID          string     `json:"id"`
	EntityType  string     `json:"entity_type"` // delivery, grocery, restaurant
	EntityID    string     `json:"entity_id"`
	Status      string     `json:"status"` // pending, approved, rejected
	SubmittedAt time.Time  `json:"submitted_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy  *string    `json:"reviewed_by,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	Documents   *string    `json:"documents,omitempty"` // JSON string
	// Additional display fields
	EntityName  string     `json:"entity_name,omitempty"`
	EntityPhone string     `json:"entity_phone,omitempty"`
	EntityCity  string     `json:"entity_city,omitempty"`
}

// ApprovalAction represents approve/reject action
type ApprovalAction struct {
	Notes string `json:"notes,omitempty"`
}

// Complaint represents a customer complaint
type Complaint struct {
	ID               string     `json:"id"`
	OrderID          *string    `json:"order_id,omitempty"`
	UserID           *string    `json:"user_id,omitempty"`
	ComplaintType    string     `json:"complaint_type"`
	Description      string     `json:"description"`
	Status           string     `json:"status"` // open, investigating, resolved, closed
	Priority         string     `json:"priority"` // low, medium, high, critical
	Culprit          string     `json:"culprit,omitempty"`
	ReasonCode       string     `json:"reason_code,omitempty"`
	Confidence       float64    `json:"confidence,omitempty"`
	EvidenceTimeline *string    `json:"evidence_timeline,omitempty"` // JSON array string
	CreatedOn        time.Time  `json:"created_on"`
	ResolvedOn       *time.Time `json:"resolved_on,omitempty"`
	ResolvedBy       *string    `json:"resolved_by,omitempty"`
	ResolutionNotes  string     `json:"resolution_notes,omitempty"`
	RefundAmount     float64    `json:"refund_amount"`
	// Additional display fields
	CustomerName     string     `json:"customer_name,omitempty"`
	CustomerPhone    string     `json:"customer_phone,omitempty"`
	OrderAmount      float64    `json:"order_amount,omitempty"`
}

// ComplaintResolve represents complaint resolution action
type ComplaintResolve struct {
	ResolutionNotes string  `json:"resolution_notes,omitempty"`
	RefundAmount    float64 `json:"refund_amount,omitempty"`
}

// ComplaintInvestigation represents detailed investigation info
type AdminComplaintInvestigation struct {
	ComplaintID      string   `json:"complaint_id"`
	OrderID          string   `json:"order_id"`
	Culprit          string   `json:"culprit"`
	ReasonCode       string   `json:"reason_code"`
	Confidence       float64  `json:"confidence"`
	EvidenceTimeline []string `json:"evidence_timeline"`
}

// Partner represents a delivery/grocery/restaurant partner for admin listing
type Partner struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"` // delivery, grocery, restaurant
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email,omitempty"`
	City         string    `json:"city"`
	Status       string    `json:"status"` // active, suspended, pending
	KYCStatus    string    `json:"kyc_status"`
	Rating       float64   `json:"rating"`
	TotalOrders  int       `json:"total_orders,omitempty"`
	TotalTrips   int       `json:"total_trips,omitempty"`
	JoinedOn     time.Time `json:"joined_on"`
	IsSuspended  bool      `json:"is_suspended"`
	SuspendedOn  *time.Time `json:"suspended_on,omitempty"`
	SuspendReason string   `json:"suspend_reason,omitempty"`
}

// PartnerSuspend represents partner suspension action
type PartnerSuspend struct {
	Reason string `json:"reason,omitempty"`
}

// PartnerSuspension represents a suspension record
type PartnerSuspension struct {
	ID              string     `json:"id"`
	PartnerID       string     `json:"partner_id"`
	PartnerType     string     `json:"partner_type"`
	Reason          string     `json:"reason"`
	SuspendedBy     string     `json:"suspended_by"`
	SuspendedOn     time.Time  `json:"suspended_on"`
	SuspensionEndDate *time.Time `json:"suspension_end_date,omitempty"`
	ReinstatedOn    *time.Time `json:"reinstated_on,omitempty"`
	ReinstatedBy    *string    `json:"reinstated_by,omitempty"`
	IsPermanent     bool       `json:"is_permanent"`
}

// OnboardingLead represents an incomplete registration
type OnboardingLead struct {
	ID               string    `json:"id"`
	EntityType       string    `json:"type"` // delivery, grocery, restaurant (named "type" for API response)
	Name             string    `json:"name"`
	Email            string    `json:"email,omitempty"`
	Phone            string    `json:"phone"`
	City             string    `json:"city"`
	LastStep         string    `json:"last_step"`
	TotalSteps       int       `json:"total_steps,omitempty"`
	StepCompleted    int       `json:"step_completed,omitempty"`
	LastActiveAt     time.Time `json:"last_active_at"`
	DaysSinceStart   int       `json:"days_since_start"`
	PotentialLoss    float64   `json:"potential_loss"`
	NotificationsSent int      `json:"notifications_sent,omitempty"`
	LastNotificationAt *time.Time `json:"last_notification_at,omitempty"`
	Source           string    `json:"source,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
}

// OnboardingLeadNotify represents notification action
type OnboardingLeadNotify struct {
	Channel string `json:"channel,omitempty"` // sms, email, push
	Message string `json:"message,omitempty"`
}

// AdminKpis represents platform-wide KPIs for admin dashboard
type AdminKpis struct {
	// Orders
	TotalOrdersToday     int     `json:"total_orders_today"`
	TotalRevenueToday    float64 `json:"total_revenue_today"`
	AvgOrderValue        float64 `json:"avg_order_value"`
	OrderGrowthPct       float64 `json:"order_growth_pct"`

	// Delivery
	ActiveDeliveryPartners int     `json:"active_delivery_partners"`
	AvgDeliveryTimeMins    int     `json:"avg_delivery_time_mins"`
	OnTimeDeliveryPct      float64 `json:"on_time_delivery_pct"`

	// Vendors
	ActiveGroceryStores    int     `json:"active_grocery_stores"`
	ActiveRestaurants      int     `json:"active_restaurants"`

	// Issues
	OpenComplaints         int     `json:"open_complaints"`
	PendingApprovals       int     `json:"pending_approvals"`
	StalledOnboardings     int     `json:"stalled_onboardings"`

	// Trends (compared to yesterday)
	OrdersTrend            float64 `json:"orders_trend"`     // percentage change
	RevenueTrend           float64 `json:"revenue_trend"`    // percentage change
	ComplaintsTrend        float64 `json:"complaints_trend"` // percentage change
}

// AdminActivityLog represents an admin action log entry
type AdminActivityLog struct {
	ID         string    `json:"id"`
	AdminID    string    `json:"admin_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type,omitempty"`
	EntityID   string    `json:"entity_id,omitempty"`
	Details    *string   `json:"details,omitempty"` // JSON string
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
