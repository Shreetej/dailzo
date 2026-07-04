package repository

import (
	"context"
	"errors"
	"time"

	"dailzo/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketingRepository struct {
	db *pgxpool.Pool
}

func NewMarketingRepository(db *pgxpool.Pool) *MarketingRepository {
	return &MarketingRepository{db: db}
}

func (r *MarketingRepository) GetDiscounts(ctx context.Context) ([]models.DiscountCampaign, error) {
	query := `SELECT id, restaurant_id, code, type, segment, percent, flat_amount,
	                 min_order_value, capping_amount, start_date, end_date, status
	          FROM discount_campaigns
	          ORDER BY created_on DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	discounts := []models.DiscountCampaign{}
	for rows.Next() {
		var d models.DiscountCampaign
		if err := rows.Scan(
			&d.ID, &d.RestaurantID, &d.Code, &d.Type, &d.Segment, &d.Percent,
			&d.FlatAmount, &d.MinOrderValue, &d.CappingAmount, &d.StartDate,
			&d.EndDate, &d.Status,
		); err != nil {
			return nil, err
		}
		discounts = append(discounts, d)
	}
	return discounts, rows.Err()
}

func (r *MarketingRepository) CreateDiscount(ctx context.Context, d models.DiscountCampaign) (models.DiscountCampaign, error) {
	if d.ID == "" {
		d.ID = GetIdToRecord("DISC")
	}
	if d.Status == "" {
		d.Status = "upcoming"
	}

	query := `INSERT INTO discount_campaigns
	          (id, restaurant_id, code, type, segment, percent, flat_amount,
	           min_order_value, capping_amount, start_date, end_date, status,
	           created_on, last_updated_on)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)`

	_, err := r.db.Exec(ctx, query,
		d.ID, d.RestaurantID, d.Code, d.Type, d.Segment, d.Percent, d.FlatAmount,
		d.MinOrderValue, d.CappingAmount, d.StartDate, d.EndDate, d.Status,
		time.Now(),
	)
	return d, err
}

func (r *MarketingRepository) GetAdCampaigns(ctx context.Context) ([]models.AdCampaign, error) {
	query := `SELECT id, restaurant_id, kind, name, status, cpc, target_customers,
	                 timeslot, outlet_ids, start_date, duration_days, budget_line,
	                 clicks, spend
	          FROM ad_campaigns
	          ORDER BY created_on DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ads := []models.AdCampaign{}
	for rows.Next() {
		var a models.AdCampaign
		if err := rows.Scan(
			&a.ID, &a.RestaurantID, &a.Kind, &a.Name, &a.Status, &a.Cpc,
			&a.TargetCustomers, &a.Timeslot, &a.OutletIDs, &a.StartDate,
			&a.DurationDays, &a.BudgetLine, &a.Clicks, &a.Spend,
		); err != nil {
			return nil, err
		}
		ads = append(ads, a)
	}
	return ads, rows.Err()
}

func (r *MarketingRepository) CreateAdCampaign(ctx context.Context, a models.AdCampaign) (models.AdCampaign, error) {
	if a.ID == "" {
		a.ID = GetIdToRecord("ADCP")
	}
	if a.Status == "" {
		a.Status = "upcoming"
	}
	if a.OutletIDs == nil {
		a.OutletIDs = []string{}
	}

	query := `INSERT INTO ad_campaigns
	          (id, restaurant_id, kind, name, status, cpc, target_customers,
	           timeslot, outlet_ids, start_date, duration_days, budget_line,
	           clicks, spend, created_on, last_updated_on)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)`

	_, err := r.db.Exec(ctx, query,
		a.ID, a.RestaurantID, a.Kind, a.Name, a.Status, a.Cpc, a.TargetCustomers,
		a.Timeslot, a.OutletIDs, a.StartDate, a.DurationDays, a.BudgetLine,
		a.Clicks, a.Spend, time.Now(),
	)
	return a, err
}

func (r *MarketingRepository) StopAdCampaign(ctx context.Context, id string) error {
	query := `UPDATE ad_campaigns
	          SET status = 'inactive', last_updated_on = $1
	          WHERE id = $2`

	result, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("ad campaign not found")
	}
	return nil
}

func (r *MarketingRepository) GetAdPacks(ctx context.Context) ([]models.AdPack, error) {
	query := `SELECT id, tier, description, clicks, price, cpc, duration_days
	          FROM ad_packs ORDER BY price ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packs := []models.AdPack{}
	for rows.Next() {
		var p models.AdPack
		if err := rows.Scan(
			&p.ID, &p.Tier, &p.Description, &p.Clicks, &p.Price, &p.Cpc,
			&p.DurationDays,
		); err != nil {
			return nil, err
		}
		packs = append(packs, p)
	}
	return packs, rows.Err()
}

func (r *MarketingRepository) MarkOrderItemsOutOfStock(ctx context.Context, orderID string, productIDs []string) error {
	query := `INSERT INTO order_out_of_stock_items (order_id, product_id)
	          VALUES ($1, $2)
	          ON CONFLICT (order_id, product_id) DO NOTHING`

	for _, productID := range productIDs {
		if _, err := r.db.Exec(ctx, query, orderID, productID); err != nil {
			return err
		}
	}
	return nil
}

func (r *MarketingRepository) GetOrderOutOfStockItems(ctx context.Context, orderID string) ([]string, error) {
	query := `SELECT product_id FROM order_out_of_stock_items WHERE order_id = $1`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	productIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		productIDs = append(productIDs, id)
	}
	return productIDs, rows.Err()
}

func (r *MarketingRepository) CreateVendorOutlet(ctx context.Context, o models.VendorOutlet) (models.VendorOutlet, error) {
	if o.ID == "" {
		o.ID = GetIdToRecord("OUTL")
	}
	if o.Cuisines == nil {
		o.Cuisines = []string{}
	}

	query := `INSERT INTO vendor_outlets
	          (id, restaurant_id, outlet_type, menu, cuisines, cost_for_two,
	           avg_delivery_time, address, packaging_charge, operating_hours,
	           is_active, created_on, last_updated_on)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11, $12, $12)`

	_, err := r.db.Exec(ctx, query,
		o.ID, o.RestaurantID, o.OutletType, o.Menu, o.Cuisines, o.CostForTwo,
		o.AvgDeliveryTime, jsonbOrEmpty(o.Address), jsonbOrEmpty(o.PackagingCharge),
		jsonbOrEmpty(o.OperatingHours), o.IsActive, time.Now(),
	)
	return o, err
}

func jsonbOrEmpty(raw []byte) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}
