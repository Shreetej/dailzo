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

// GroceryRepository handles grocery vendor data operations
type GroceryRepository struct {
	db *pgxpool.Pool
}

// NewGroceryRepository creates a new grocery repository
func NewGroceryRepository(db *pgxpool.Pool) *GroceryRepository {
	return &GroceryRepository{db: db}
}

// CreateProfile creates a new grocery vendor profile
func (r *GroceryRepository) CreateProfile(ctx context.Context, profile *models.GroceryProfileCreate) (string, error) {
	id := GetIdToRecord("GROC")
	query := `
		INSERT INTO grocery_profiles (
			id, user_id, store_name, owner_name, email, phone, address, city,
			pincode, fssai_license, gst_number, pan_number, working_hours,
			kyc_status, payout_status, is_active, rating, total_orders,
			commission_rate, created_on, last_updated_on
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			'pending', 'pending', false, 0.0, 0, 15.0, $14, $14)
		RETURNING id`

	now := time.Now()
	err := r.db.QueryRow(ctx, query,
		id, profile.UserID, profile.StoreName, profile.OwnerName, profile.Email,
		profile.Phone, profile.Address, profile.City, profile.Pincode,
		profile.FSSAILicense, profile.GSTNumber, profile.PANNumber,
		profile.WorkingHours, now,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to create grocery profile: %w", err)
	}
	return id, nil
}

// GetProfileByUserID gets grocery profile by user ID
func (r *GroceryRepository) GetProfileByUserID(ctx context.Context, userID string) (*models.GroceryProfile, error) {
	query := `
		SELECT id, user_id, store_name, owner_name, email, phone, address, city,
			pincode, kyc_status, kyc_documents, fssai_license, gst_number, pan_number,
			payout_status, bank_details, working_hours, is_active, rating,
			total_orders, commission_rate, created_on, last_updated_on
		FROM grocery_profiles
		WHERE user_id = $1`

	var profile models.GroceryProfile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.StoreName, &profile.OwnerName,
		&profile.Email, &profile.Phone, &profile.Address, &profile.City,
		&profile.Pincode, &profile.KYCStatus, &profile.KYCDocuments,
		&profile.FSSAILicense, &profile.GSTNumber, &profile.PANNumber,
		&profile.PayoutStatus, &profile.BankDetails, &profile.WorkingHours,
		&profile.IsActive, &profile.Rating, &profile.TotalOrders,
		&profile.CommissionRate, &profile.CreatedOn, &profile.LastUpdatedOn,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.New("grocery profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get grocery profile: %w", err)
	}
	return &profile, nil
}

// GetProfileByID gets grocery profile by profile ID
func (r *GroceryRepository) GetProfileByID(ctx context.Context, id string) (*models.GroceryProfile, error) {
	query := `
		SELECT id, user_id, store_name, owner_name, email, phone, address, city,
			pincode, kyc_status, kyc_documents, fssai_license, gst_number, pan_number,
			payout_status, bank_details, working_hours, is_active, rating,
			total_orders, commission_rate, created_on, last_updated_on
		FROM grocery_profiles
		WHERE id = $1`

	var profile models.GroceryProfile
	err := r.db.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.UserID, &profile.StoreName, &profile.OwnerName,
		&profile.Email, &profile.Phone, &profile.Address, &profile.City,
		&profile.Pincode, &profile.KYCStatus, &profile.KYCDocuments,
		&profile.FSSAILicense, &profile.GSTNumber, &profile.PANNumber,
		&profile.PayoutStatus, &profile.BankDetails, &profile.WorkingHours,
		&profile.IsActive, &profile.Rating, &profile.TotalOrders,
		&profile.CommissionRate, &profile.CreatedOn, &profile.LastUpdatedOn,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.New("grocery profile not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get grocery profile: %w", err)
	}
	return &profile, nil
}

// GetDailyKpis gets daily KPIs for a grocery vendor
func (r *GroceryRepository) GetDailyKpis(ctx context.Context, userID string, date time.Time) (*models.GroceryKPI, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, grocery_id, date, today_revenue, pending_orders, completed_orders,
			cancelled_orders, cancel_rate, avg_prep_time_mins, low_stock_items,
			expiry_risk_items
		FROM daily_grocery_kpis
		WHERE grocery_id = $1 AND date = $2`

	var kpi models.GroceryKPI
	err = r.db.QueryRow(ctx, query, profile.ID, date.Format("2006-01-02")).Scan(
		&kpi.ID, &kpi.GroceryID, &kpi.Date, &kpi.TodayRevenue,
		&kpi.PendingOrders, &kpi.CompletedOrders, &kpi.CancelledOrders,
		&kpi.CancelRate, &kpi.AvgPrepTimeMins, &kpi.LowStockItems,
		&kpi.ExpiryRiskItems,
	)

	if err == pgx.ErrNoRows {
		// Calculate KPIs dynamically if no pre-calculated record exists
		return r.calculateKpis(ctx, profile.ID, date)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get daily kpis: %w", err)
	}

	// Format prep time string
	kpi.AvgPrepTime = fmt.Sprintf("%d mins", kpi.AvgPrepTimeMins)

	return &kpi, nil
}

// calculateKpis calculates KPIs dynamically
func (r *GroceryRepository) calculateKpis(ctx context.Context, groceryID string, date time.Time) (*models.GroceryKPI, error) {
	kpi := &models.GroceryKPI{
		GroceryID: groceryID,
		Date:      date.Format("2006-01-02"),
	}

	// Get order stats for the day
	orderQuery := `
		SELECT
			COALESCE(SUM(total_amount), 0) as revenue,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled
		FROM orders
		WHERE restaurant_id = $1 AND DATE(order_date) = $2`

	var revenue float64
	var pending, completed, cancelled int
	err := r.db.QueryRow(ctx, orderQuery, groceryID, date.Format("2006-01-02")).Scan(
		&revenue, &pending, &completed, &cancelled,
	)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to calculate order kpis: %w", err)
	}

	kpi.TodayRevenue = revenue
	kpi.PendingOrders = pending
	kpi.CompletedOrders = completed
	kpi.CancelledOrders = cancelled
	total := pending + completed + cancelled
	if total > 0 {
		kpi.CancelRate = float64(cancelled) / float64(total) * 100
	}

	// Get stock alerts
	stockQuery := `
		SELECT
			COUNT(CASE WHEN stock_quantity <= low_stock_threshold THEN 1 END) as low_stock,
			COUNT(CASE WHEN expiry_date <= CURRENT_DATE + INTERVAL '7 days' THEN 1 END) as expiry_risk
		FROM food_products
		WHERE outlet_id = $1 AND is_active = true`

	err = r.db.QueryRow(ctx, stockQuery, groceryID).Scan(
		&kpi.LowStockItems, &kpi.ExpiryRiskItems,
	)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to calculate stock kpis: %w", err)
	}

	kpi.AvgPrepTime = "15 mins" // Default
	kpi.AvgPrepTimeMins = 15

	return kpi, nil
}

// GetExpiryAlerts gets products nearing expiry
func (r *GroceryRepository) GetExpiryAlerts(ctx context.Context, userID string) ([]models.GroceryExpiryAlert, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, name, expiry_date, stock_quantity, category, price
		FROM food_products
		WHERE outlet_id = $1
			AND is_active = true
			AND expiry_date IS NOT NULL
			AND expiry_date <= CURRENT_DATE + INTERVAL '7 days'
			AND expiry_date >= CURRENT_DATE
		ORDER BY expiry_date ASC`

	rows, err := r.db.Query(ctx, query, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query expiry alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.GroceryExpiryAlert
	for rows.Next() {
		var alert models.GroceryExpiryAlert
		var expiryDate time.Time

		err := rows.Scan(
			&alert.ProductID, &alert.Name, &expiryDate,
			&alert.StockQuantity, &alert.Category, &alert.Price,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expiry alert: %w", err)
		}

		alert.ExpiryDate = expiryDate.Format("2006-01-02")
		alert.DaysLeft = int(time.Until(expiryDate).Hours() / 24)
		if alert.DaysLeft < 0 {
			alert.DaysLeft = 0
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetStockAlerts gets products with low stock
func (r *GroceryRepository) GetStockAlerts(ctx context.Context, userID string) ([]models.GroceryStockAlert, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, name, stock_quantity, low_stock_threshold, category, price
		FROM food_products
		WHERE outlet_id = $1
			AND is_active = true
			AND stock_quantity <= low_stock_threshold
		ORDER BY stock_quantity ASC`

	rows, err := r.db.Query(ctx, query, profile.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query stock alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.GroceryStockAlert
	for rows.Next() {
		var alert models.GroceryStockAlert
		err := rows.Scan(
			&alert.ProductID, &alert.Name, &alert.StockQuantity,
			&alert.Threshold, &alert.Category, &alert.Price,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stock alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetPayoutSummary gets payout summary for a vendor
func (r *GroceryRepository) GetPayoutSummary(ctx context.Context, userID string) (*models.GroceryPayoutSummary, error) {
	profile, err := r.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get total earnings and pending amount
	earningsQuery := `
		SELECT
			COALESCE(SUM(total_amount), 0) as total_earnings,
			COALESCE(SUM(total_amount) * $2 / 100, 0) as commission
		FROM orders
		WHERE restaurant_id = $1 AND status = 'completed'`

	var totalEarnings, commission float64
	err = r.db.QueryRow(ctx, earningsQuery, profile.ID, profile.CommissionRate).Scan(
		&totalEarnings, &commission,
	)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get earnings: %w", err)
	}

	// Get last payout
	lastPayoutQuery := `
		SELECT amount, payout_date, status
		FROM grocery_payouts
		WHERE grocery_id = $1 AND status = 'completed'
		ORDER BY payout_date DESC
		LIMIT 1`

	var lastPayoutAmount float64
	var lastPayoutDate *string
	var lastPayoutStatus string

	err = r.db.QueryRow(ctx, lastPayoutQuery, profile.ID).Scan(
		&lastPayoutAmount, &lastPayoutDate, &lastPayoutStatus,
	)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get last payout: %w", err)
	}

	// Get pending payout
	pendingPayoutQuery := `
		SELECT amount, payout_date, status
		FROM grocery_payouts
		WHERE grocery_id = $1 AND status = 'pending'
		ORDER BY created_on DESC
		LIMIT 1`

	var pendingAmount float64
	var nextPayoutDate *string
	var pendingStatus string

	err = r.db.QueryRow(ctx, pendingPayoutQuery, profile.ID).Scan(
		&pendingAmount, &nextPayoutDate, &pendingStatus,
	)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get pending payout: %w", err)
	}

	summary := &models.GroceryPayoutSummary{
		TotalEarnings:      totalEarnings - commission,
		CommissionDeducted: commission,
		Status:             profile.PayoutStatus,
	}

	if pendingAmount > 0 {
		summary.Amount = pendingAmount
		summary.PendingAmount = pendingAmount
		summary.Status = "pending"
		if nextPayoutDate != nil {
			summary.NextPayoutDate = *nextPayoutDate
		}
	}

	if lastPayoutAmount > 0 {
		summary.LastPayoutAmount = lastPayoutAmount
		if lastPayoutDate != nil {
			summary.LastPayoutDate = *lastPayoutDate
		}
	}

	return summary, nil
}

// UpdateProfile updates a grocery profile
func (r *GroceryRepository) UpdateProfile(ctx context.Context, id string, updates map[string]interface{}) error {
	// Build dynamic update query based on provided fields
	// For now, use a simple update for common fields
	query := `
		UPDATE grocery_profiles
		SET last_updated_on = $1
		WHERE id = $2`

	_, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update grocery profile: %w", err)
	}
	return nil
}
