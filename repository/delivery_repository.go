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

// DeliveryRepository handles delivery partner data operations
type DeliveryRepository struct {
	db *pgxpool.Pool
}

// NewDeliveryRepository creates a new delivery repository
func NewDeliveryRepository(db *pgxpool.Pool) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

// CreateProfile creates a new delivery partner profile
func (r *DeliveryRepository) CreateProfile(ctx context.Context, profile *models.DeliveryProfileCreate) (string, error) {
	id := GetIdToRecord("DLVR")
	query := `
		INSERT INTO delivery_profiles (
			id, user_id, name, phone, city, vehicle_type, vehicle_number,
			license_number, kyc_status, rating, total_trips, is_online,
			created_on, last_updated_on
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', 0.0, 0, false, $9, $9)
		RETURNING id`

	now := time.Now()
	err := r.db.QueryRow(ctx, query,
		id, profile.UserID, profile.Name, profile.Phone, profile.City,
		profile.VehicleType, profile.VehicleNumber, profile.LicenseNumber, now,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to create delivery profile: %w", err)
	}
	return id, nil
}

// GetProfileByUserID gets delivery profile by user ID
func (r *DeliveryRepository) GetProfileByUserID(ctx context.Context, userID string) (*models.DeliveryProfile, error) {
	query := `
		SELECT id, user_id, name, phone, city, vehicle_type, vehicle_number,
			license_number, kyc_status, kyc_documents, rating, total_trips,
			is_online, current_lat, current_lng, last_location_update,
			working_hours, created_on, last_updated_on
		FROM delivery_profiles
		WHERE user_id = $1`

	var profile models.DeliveryProfile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.Name, &profile.Phone, &profile.City,
		&profile.VehicleType, &profile.VehicleNumber, &profile.LicenseNumber,
		&profile.KYCStatus, &profile.KYCDocuments, &profile.Rating, &profile.TotalTrips,
		&profile.IsOnline, &profile.CurrentLat, &profile.CurrentLng, &profile.LastLocationUpdate,
		&profile.WorkingHours, &profile.CreatedOn, &profile.LastUpdatedOn,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.New("delivery profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery profile: %w", err)
	}
	return &profile, nil
}

// GetProfileByID gets delivery profile by profile ID
func (r *DeliveryRepository) GetProfileByID(ctx context.Context, id string) (*models.DeliveryProfile, error) {
	query := `
		SELECT id, user_id, name, phone, city, vehicle_type, vehicle_number,
			license_number, kyc_status, kyc_documents, rating, total_trips,
			is_online, current_lat, current_lng, last_location_update,
			working_hours, created_on, last_updated_on
		FROM delivery_profiles
		WHERE id = $1`

	var profile models.DeliveryProfile
	err := r.db.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.UserID, &profile.Name, &profile.Phone, &profile.City,
		&profile.VehicleType, &profile.VehicleNumber, &profile.LicenseNumber,
		&profile.KYCStatus, &profile.KYCDocuments, &profile.Rating, &profile.TotalTrips,
		&profile.IsOnline, &profile.CurrentLat, &profile.CurrentLng, &profile.LastLocationUpdate,
		&profile.WorkingHours, &profile.CreatedOn, &profile.LastUpdatedOn,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.New("delivery profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery profile: %w", err)
	}
	return &profile, nil
}

// GetDailyKpis gets daily KPIs for a delivery partner
func (r *DeliveryRepository) GetDailyKpis(ctx context.Context, userID string, date time.Time) (*models.DeliveryKpi, error) {
	// First get the delivery profile ID
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, delivery_person_id, date, today_earnings, completed_trips,
			online_minutes, acceptance_rate, on_time_rate, avg_delivery_time_mins,
			total_distance_km
		FROM daily_delivery_kpis
		WHERE delivery_person_id = $1 AND date = $2`

	var kpi models.DeliveryKpi
	err = r.db.QueryRow(ctx, query, profile.ID, date.Format("2006-01-02")).Scan(
		&kpi.ID, &kpi.DeliveryPersonID, &kpi.Date, &kpi.TodayEarnings,
		&kpi.CompletedTrips, &kpi.OnlineMinutes, &kpi.AcceptanceRatePct,
		&kpi.OnTimeRate, &kpi.AvgDeliveryTimeMins, &kpi.TotalDistanceKm,
	)

	if err == pgx.ErrNoRows {
		// Return empty KPIs if none exist for today
		return &models.DeliveryKpi{
			DeliveryPersonID:  profile.ID,
			Date:              date.Format("2006-01-02"),
			TodayEarnings:     0,
			CompletedTrips:    0,
			OnlineMinutes:     0,
			AcceptanceRatePct: 0,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get daily kpis: %w", err)
	}

	// Format online hours string
	hours := kpi.OnlineMinutes / 60
	mins := kpi.OnlineMinutes % 60
	kpi.OnlineHours = fmt.Sprintf("%dh %dm", hours, mins)

	return &kpi, nil
}

// GetActiveTask gets the current active delivery task for a partner
func (r *DeliveryRepository) GetActiveTask(ctx context.Context, userID string) (*models.DeliveryTask, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, order_id, delivery_person_id, status, pickup_lat, pickup_lng,
			dropoff_lat, dropoff_lng, eta_mins, pickup_point, drop_point,
			distance_km, assigned_at, accepted_at, picked_up_at, delivered_at,
			cancelled_at, cancel_reason, notes
		FROM delivery_tasks
		WHERE delivery_person_id = $1 AND status NOT IN ('delivered', 'cancelled')
		ORDER BY assigned_at DESC
		LIMIT 1`

	var task models.DeliveryTask
	err = r.db.QueryRow(ctx, query, profile.ID).Scan(
		&task.ID, &task.OrderID, &task.DeliveryPersonID, &task.Status,
		&task.PickupLat, &task.PickupLng, &task.DropoffLat, &task.DropoffLng,
		&task.ETAMins, &task.PickupPoint, &task.DropPoint, &task.DistanceKm,
		&task.AssignedAt, &task.AcceptedAt, &task.PickedUpAt, &task.DeliveredAt,
		&task.CancelledAt, &task.CancelReason, &task.Notes,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // No active task
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active task: %w", err)
	}
	return &task, nil
}

// GetTraceEvents gets delivery trace events for an order
func (r *DeliveryRepository) GetTraceEvents(ctx context.Context, orderID string) ([]models.DeliveryTraceEvent, error) {
	query := `
		SELECT id, order_id, delivery_task_id, status, timestamp, actor_type, lat, lng, notes
		FROM delivery_trace_events
		WHERE order_id = $1
		ORDER BY timestamp ASC`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query trace events: %w", err)
	}
	defer rows.Close()

	var events []models.DeliveryTraceEvent
	for rows.Next() {
		var event models.DeliveryTraceEvent
		err := rows.Scan(
			&event.ID, &event.OrderID, &event.DeliveryTaskID, &event.Status,
			&event.Timestamp, &event.ActorType, &event.Lat, &event.Lng, &event.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trace event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetSlaKpis calculates SLA KPIs for a delivery partner
func (r *DeliveryRepository) GetSlaKpis(ctx context.Context, userID string) (*models.DeliverySlaKpi, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Calculate SLA metrics from completed deliveries in last 30 days
	query := `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (picked_up_at - assigned_at))/60), 0) as avg_pickup_delay,
			COALESCE(AVG(EXTRACT(EPOCH FROM (delivered_at - picked_up_at))/60), 0) as avg_delivery_time,
			COALESCE(COUNT(CASE WHEN delivered_at IS NOT NULL THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0), 0) as on_time_pct,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as breach_count
		FROM delivery_tasks
		WHERE delivery_person_id = $1
			AND assigned_at >= NOW() - INTERVAL '30 days'`

	var kpi models.DeliverySlaKpi
	err = r.db.QueryRow(ctx, query, profile.ID).Scan(
		&kpi.AvgPickupDelayMins, &kpi.AvgDeliveryDelayMins,
		&kpi.OnTimePct, &kpi.BreachCount,
	)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get SLA kpis: %w", err)
	}

	return &kpi, nil
}

// GetEarningsInsights gets earnings optimization insights
func (r *DeliveryRepository) GetEarningsInsights(ctx context.Context, userID string) (*models.DeliveryEarningsInsight, error) {
	// This would typically involve complex analytics queries
	// For now, return placeholder data that can be enhanced with ML/analytics
	return &models.DeliveryEarningsInsight{
		BestZones:         []string{"Downtown", "Tech Park", "Mall Area"},
		BestShifts:        []string{"Lunch (11AM-2PM)", "Dinner (6PM-10PM)"},
		AvgHourlyEarnings: 150.0,
		ExpectedUpliftPct: 15.0,
	}, nil
}

// GetRecommendations gets unacknowledged recommendations for a partner
func (r *DeliveryRepository) GetRecommendations(ctx context.Context, userID string) ([]models.DeliveryRecommendation, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, delivery_person_id, type, title, message, priority, cta_route,
			data, acknowledged, acknowledged_at, created_at, expires_at
		FROM delivery_recommendations
		WHERE delivery_person_id = $1
			AND acknowledged = false
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY
			CASE priority WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
			created_at DESC`

	rows, err := r.db.Query(ctx, query, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query recommendations: %w", err)
	}
	defer rows.Close()

	var recommendations []models.DeliveryRecommendation
	for rows.Next() {
		var rec models.DeliveryRecommendation
		err := rows.Scan(
			&rec.ID, &rec.DeliveryPersonID, &rec.Type, &rec.Title, &rec.Message,
			&rec.Priority, &rec.CTARoute, &rec.Data, &rec.Acknowledged,
			&rec.AcknowledgedAt, &rec.CreatedAt, &rec.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommendation: %w", err)
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// AckRecommendation marks a recommendation as acknowledged
func (r *DeliveryRepository) AckRecommendation(ctx context.Context, id string) error {
	query := `
		UPDATE delivery_recommendations
		SET acknowledged = true, acknowledged_at = $1
		WHERE id = $2`

	_, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to acknowledge recommendation: %w", err)
	}
	return nil
}

// GetWeeklyEarnings gets earnings breakdown for the last 7 days
func (r *DeliveryRepository) GetWeeklyEarnings(ctx context.Context, userID string) (*models.WeeklyEarnings, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT date, today_earnings, completed_trips, online_minutes
		FROM daily_delivery_kpis
		WHERE delivery_person_id = $1
			AND date >= CURRENT_DATE - INTERVAL '7 days'
		ORDER BY date ASC`

	rows, err := r.db.Query(ctx, query, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly earnings: %w", err)
	}
	defer rows.Close()

	weekly := &models.WeeklyEarnings{
		DailyBreakdown: make([]models.DailyEarning, 0),
	}

	for rows.Next() {
		var date time.Time
		var daily models.DailyEarning
		var onlineMins int

		err := rows.Scan(&date, &daily.Earnings, &daily.Trips, &onlineMins)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily earning: %w", err)
		}

		daily.Date = date.Format("2006-01-02")
		daily.Hours = float64(onlineMins) / 60.0

		weekly.TotalEarnings += daily.Earnings
		weekly.TotalTrips += daily.Trips
		weekly.TotalHours += daily.Hours
		weekly.DailyBreakdown = append(weekly.DailyBreakdown, daily)
	}

	return weekly, nil
}

// GetShifts gets all available shifts
func (r *DeliveryRepository) GetShifts(ctx context.Context) ([]models.DeliveryShift, error) {
	query := `
		SELECT s.id, s.name, s.zone, s.start_time, s.end_time, s.max_partners,
			s.incentive_multiplier, s.is_active,
			COALESCE(COUNT(b.id), 0) as current_bookings
		FROM delivery_shifts s
		LEFT JOIN delivery_shift_bookings b ON s.id = b.shift_id AND b.date = CURRENT_DATE
		WHERE s.is_active = true
		GROUP BY s.id
		ORDER BY s.start_time`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query shifts: %w", err)
	}
	defer rows.Close()

	var shifts []models.DeliveryShift
	for rows.Next() {
		var shift models.DeliveryShift
		var startTime, endTime time.Time

		err := rows.Scan(
			&shift.ID, &shift.Name, &shift.Zone, &startTime, &endTime,
			&shift.MaxPartners, &shift.IncentiveMultiplier, &shift.IsActive,
			&shift.CurrentBookings,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shift: %w", err)
		}

		shift.StartTime = startTime.Format("15:04")
		shift.EndTime = endTime.Format("15:04")
		shift.AvailableSlots = shift.MaxPartners - shift.CurrentBookings

		shifts = append(shifts, shift)
	}

	return shifts, nil
}

// UpdateLocation updates the delivery partner's current location
func (r *DeliveryRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	query := `
		UPDATE delivery_profiles
		SET current_lat = $1, current_lng = $2, last_location_update = $3
		WHERE user_id = $4`

	_, err := r.db.Exec(ctx, query, lat, lng, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update location: %w", err)
	}
	return nil
}

// UpdateOnlineStatus updates the delivery partner's online status
func (r *DeliveryRepository) UpdateOnlineStatus(ctx context.Context, userID string, isOnline bool) error {
	query := `
		UPDATE delivery_profiles
		SET is_online = $1, last_updated_on = $2
		WHERE user_id = $3`

	_, err := r.db.Exec(ctx, query, isOnline, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update online status: %w", err)
	}
	return nil
}

// CreateTraceEvent creates a new trace event for an order
func (r *DeliveryRepository) CreateTraceEvent(ctx context.Context, event *models.DeliveryTraceEvent) (string, error) {
	id := GetIdToRecord("TRACE")
	query := `
		INSERT INTO delivery_trace_events (
			id, order_id, delivery_task_id, status, timestamp, actor_type, lat, lng, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id, event.OrderID, event.DeliveryTaskID, event.Status,
		time.Now(), event.ActorType, event.Lat, event.Lng, event.Notes,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to create trace event: %w", err)
	}
	return id, nil
}
