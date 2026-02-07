package models

import "time"

// DeliveryProfile represents a delivery partner profile
type DeliveryProfile struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	Name               string     `json:"name"`
	Phone              string     `json:"phone"`
	City               string     `json:"city"`
	VehicleType        string     `json:"vehicle_type,omitempty"`
	VehicleNumber      string     `json:"vehicle_number,omitempty"`
	LicenseNumber      string     `json:"license_number,omitempty"`
	KYCStatus          string     `json:"kyc_status"`
	KYCDocuments       *string    `json:"kyc_documents,omitempty"` // JSON string
	Rating             float64    `json:"rating"`
	TotalTrips         int        `json:"total_trips"`
	IsOnline           bool       `json:"is_online"`
	CurrentLat         *float64   `json:"current_lat,omitempty"`
	CurrentLng         *float64   `json:"current_lng,omitempty"`
	LastLocationUpdate *time.Time `json:"last_location_update,omitempty"`
	WorkingHours       string     `json:"working_hours,omitempty"`
	CreatedOn          time.Time  `json:"created_on"`
	LastUpdatedOn      time.Time  `json:"last_updated_on"`
	CreatedBy          *string    `json:"created_by,omitempty"`
	LastModifiedBy     *string    `json:"last_modified_by,omitempty"`
}

// DeliveryProfileCreate is used for onboarding
type DeliveryProfileCreate struct {
	UserID        string `json:"user_id"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	City          string `json:"city"`
	VehicleType   string `json:"vehicle_type"`
	VehicleNumber string `json:"vehicle_number"`
	LicenseNumber string `json:"license_number"`
}

// DeliveryTask represents an active delivery assignment
type DeliveryTask struct {
	ID               string     `json:"id"`
	OrderID          string     `json:"order_id"`
	DeliveryPersonID string     `json:"delivery_person_id"`
	Status           string     `json:"status"`
	PickupLat        *float64   `json:"pickup_lat,omitempty"`
	PickupLng        *float64   `json:"pickup_lng,omitempty"`
	DropoffLat       *float64   `json:"dropoff_lat,omitempty"`
	DropoffLng       *float64   `json:"dropoff_lng,omitempty"`
	ETAMins          int        `json:"eta_mins"`
	PickupPoint      string     `json:"pickup_point"`
	DropPoint        string     `json:"drop_point"`
	DistanceKm       float64    `json:"distance_km,omitempty"`
	AssignedAt       time.Time  `json:"assigned_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	PickedUpAt       *time.Time `json:"picked_up_at,omitempty"`
	DeliveredAt      *time.Time `json:"delivered_at,omitempty"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty"`
	CancelReason     string     `json:"cancel_reason,omitempty"`
	Notes            string     `json:"notes,omitempty"`
}

// DeliveryTraceEvent represents delivery task trace events
type DeliveryTraceEvent struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"order_id"`
	DeliveryTaskID string    `json:"delivery_task_id,omitempty"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
	ActorType      string    `json:"actor_type"`
	Lat            *float64  `json:"lat,omitempty"`
	Lng            *float64  `json:"lng,omitempty"`
	Notes          string    `json:"notes,omitempty"`
}

// DeliveryShift represents an available work shift
type DeliveryShift struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Zone                 string    `json:"zone,omitempty"`
	StartTime            string    `json:"start_time"` // HH:MM format
	EndTime              string    `json:"end_time"`   // HH:MM format
	MaxPartners          int       `json:"max_partners"`
	IncentiveMultiplier  float64   `json:"incentive_multiplier"`
	IsActive             bool      `json:"is_active"`
	CurrentBookings      int       `json:"current_bookings,omitempty"` // Computed field
	AvailableSlots       int       `json:"available_slots,omitempty"`  // Computed field
	CreatedOn            time.Time `json:"created_on,omitempty"`
}

// DeliveryShiftBooking represents a partner's shift reservation
type DeliveryShiftBooking struct {
	ID               string     `json:"id"`
	ShiftID          string     `json:"shift_id"`
	DeliveryPersonID string     `json:"delivery_person_id"`
	Date             string     `json:"date"` // YYYY-MM-DD format
	Status           string     `json:"status"`
	CheckInTime      *time.Time `json:"check_in_time,omitempty"`
	CheckOutTime     *time.Time `json:"check_out_time,omitempty"`
}

// DeliveryKpi represents daily KPIs for a delivery partner
type DeliveryKpi struct {
	ID                  string    `json:"id,omitempty"`
	DeliveryPersonID    string    `json:"delivery_person_id,omitempty"`
	Date                string    `json:"date,omitempty"`
	TodayEarnings       float64   `json:"today_earnings"`
	CompletedTrips      int       `json:"completed_trips"`
	OnlineMinutes       int       `json:"online_minutes"`
	OnlineHours         string    `json:"online_hours,omitempty"` // Formatted string like "4h 30m"
	AcceptanceRatePct   float64   `json:"acceptance_rate_pct"`
	OnTimeRate          float64   `json:"on_time_rate,omitempty"`
	AvgDeliveryTimeMins int       `json:"avg_delivery_time_mins,omitempty"`
	TotalDistanceKm     float64   `json:"total_distance_km,omitempty"`
}

// DeliverySlaKpi represents SLA-related KPIs
type DeliverySlaKpi struct {
	AvgPickupDelayMins   float64 `json:"avg_pickup_delay"`
	AvgDeliveryDelayMins float64 `json:"avg_delivery_delay"`
	OnTimePct            float64 `json:"on_time_pct"`
	BreachCount          int     `json:"breach_count"`
	IdleTimeMins         int     `json:"idle_time_mins"`
}

// DeliveryEarningsInsight represents earnings optimization insights
type DeliveryEarningsInsight struct {
	BestZones         []string `json:"best_zones"`
	BestShifts        []string `json:"best_shifts"`
	AvgHourlyEarnings float64  `json:"avg_hourly"`
	ExpectedUpliftPct float64  `json:"expected_uplift_pct"`
}

// DeliveryRecommendation represents an AI recommendation for delivery partner
type DeliveryRecommendation struct {
	ID             string     `json:"id"`
	DeliveryPersonID string   `json:"delivery_person_id,omitempty"`
	Type           string     `json:"type"`
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	Priority       string     `json:"priority"`
	CTARoute       string     `json:"cta_route,omitempty"`
	Data           *string    `json:"data,omitempty"` // JSON string
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// DailyEarning represents earnings for a single day (for weekly breakdown)
type DailyEarning struct {
	Date     string  `json:"date"`
	Earnings float64 `json:"earnings"`
	Trips    int     `json:"trips"`
	Hours    float64 `json:"hours"`
}

// WeeklyEarnings represents a week's worth of earnings data
type WeeklyEarnings struct {
	TotalEarnings float64        `json:"total_earnings"`
	TotalTrips    int            `json:"total_trips"`
	TotalHours    float64        `json:"total_hours"`
	DailyBreakdown []DailyEarning `json:"daily_breakdown"`
}

// LocationUpdate represents a real-time location update from delivery partner
type LocationUpdate struct {
	OrderID   string  `json:"order_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Status    string  `json:"status,omitempty"`
	ETAMins   int     `json:"eta_mins,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
}
