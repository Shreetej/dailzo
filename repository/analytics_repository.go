package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// platformCommissionPct is the share the platform retains from a vendor's
// delivered-order revenue; the remainder is the vendor payout.
const platformCommissionPct = 0.12

// AnalyticsRepository powers the vendor "Product Intelligence" dashboard
// sections (top products, top categories, low/no-sales). Sales are aggregated
// from the orders.items jsonb array — order_items/product_variants are not
// populated on the customer golden path, so the jsonb is the system of record
// for what actually sold. Products are joined to food_products on the trimmed
// id (ids carry trailing padding) and filtered by kind ('food'|'grocery').
type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// ProductTrend is one top-selling product with period-over-period movement.
type ProductTrend struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	UnitsSold int     `json:"units_sold"`
	Revenue   float64 `json:"revenue"`
	TrendPct  float64 `json:"trend_pct"`
	Rank      int     `json:"rank"`
}

// CategoryTrend is one category aggregate with period-over-period movement.
type CategoryTrend struct {
	Category  string  `json:"category"`
	UnitsSold int     `json:"units_sold"`
	Revenue   float64 `json:"revenue"`
	TrendPct  float64 `json:"trend_pct"`
}

// LowSeller is a product that has sold before but is moving slowly / has gone
// quiet in the recent window — a candidate for a promo or delisting.
type LowSeller struct {
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	UnitsSold     int    `json:"units_sold"`
	LastSoldAt    string `json:"last_sold_at"`
	DaysSinceSold int    `json:"days_since_sold"`
}

// roleToKind maps the app's vendor role onto food_products.kind.
func roleToKind(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "grocery":
		return "grocery"
	default: // restaurant / food / unknown → food menu
		return "food"
	}
}

// timeframeDays parses "7d"/"30d"/"90" into a day count (default 7).
func timeframeDays(tf string) int {
	tf = strings.TrimSpace(strings.ToLower(tf))
	tf = strings.TrimSuffix(tf, "d")
	if n, err := strconv.Atoi(tf); err == nil && n > 0 {
		return n
	}
	return 7
}

// ProductTrends returns the top-selling products of a kind in the window, with
// a trend % vs the immediately preceding window of equal length.
func (r *AnalyticsRepository) ProductTrends(ctx context.Context, role, timeframe string) ([]ProductTrend, error) {
	days := timeframeDays(timeframe)
	kind := roleToKind(role)
	const q = `
WITH cur AS (
  SELECT trim(e->>'product_id') AS pid,
         max(e->>'name') AS name,
         sum((e->>'quantity')::int) AS units,
         sum((e->>'quantity')::int * (e->>'price')::numeric) AS revenue
  FROM orders o CROSS JOIN LATERAL jsonb_array_elements(o.items) e
  WHERE o.order_date >= now() - make_interval(days => $1)
  GROUP BY 1
),
prev AS (
  SELECT trim(e->>'product_id') AS pid, sum((e->>'quantity')::int) AS units
  FROM orders o CROSS JOIN LATERAL jsonb_array_elements(o.items) e
  WHERE o.order_date >= now() - make_interval(days => $1 * 2)
    AND o.order_date <  now() - make_interval(days => $1)
  GROUP BY 1
)
SELECT trim(fp.id), cur.name, coalesce(fp.category,''),
       cur.units, cur.revenue,
       CASE WHEN coalesce(prev.units,0) = 0 THEN 100.0
            ELSE round((cur.units - prev.units) * 100.0 / prev.units, 1) END AS trend_pct
FROM cur
JOIN food_products fp ON trim(fp.id) = cur.pid AND fp.kind = $2
LEFT JOIN prev ON prev.pid = cur.pid
ORDER BY cur.units DESC, cur.revenue DESC
LIMIT 10`
	rows, err := r.db.Query(ctx, q, days, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProductTrend{}
	rank := 0
	for rows.Next() {
		rank++
		var t ProductTrend
		if err := rows.Scan(&t.ProductID, &t.Name, &t.Category, &t.UnitsSold, &t.Revenue, &t.TrendPct); err != nil {
			return nil, err
		}
		t.Rank = rank
		out = append(out, t)
	}
	return out, rows.Err()
}

// CategoryTrends returns the top categories of a kind in the window, with a
// trend % vs the preceding window.
func (r *AnalyticsRepository) CategoryTrends(ctx context.Context, role, timeframe string) ([]CategoryTrend, error) {
	days := timeframeDays(timeframe)
	kind := roleToKind(role)
	const q = `
WITH cur AS (
  SELECT coalesce(fp.category,'Uncategorised') AS category,
         sum((e->>'quantity')::int) AS units,
         sum((e->>'quantity')::int * (e->>'price')::numeric) AS revenue
  FROM orders o CROSS JOIN LATERAL jsonb_array_elements(o.items) e
  JOIN food_products fp ON trim(fp.id) = trim(e->>'product_id') AND fp.kind = $2
  WHERE o.order_date >= now() - make_interval(days => $1)
  GROUP BY 1
),
prev AS (
  SELECT coalesce(fp.category,'Uncategorised') AS category,
         sum((e->>'quantity')::int) AS units
  FROM orders o CROSS JOIN LATERAL jsonb_array_elements(o.items) e
  JOIN food_products fp ON trim(fp.id) = trim(e->>'product_id') AND fp.kind = $2
  WHERE o.order_date >= now() - make_interval(days => $1 * 2)
    AND o.order_date <  now() - make_interval(days => $1)
  GROUP BY 1
)
SELECT cur.category, cur.units, cur.revenue,
       CASE WHEN coalesce(prev.units,0) = 0 THEN 100.0
            ELSE round((cur.units - prev.units) * 100.0 / prev.units, 1) END AS trend_pct
FROM cur LEFT JOIN prev ON prev.category = cur.category
ORDER BY cur.units DESC, cur.revenue DESC
LIMIT 8`
	rows, err := r.db.Query(ctx, q, days, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CategoryTrend{}
	for rows.Next() {
		var t CategoryTrend
		if err := rows.Scan(&t.Category, &t.UnitsSold, &t.Revenue, &t.TrendPct); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// PayoutSummary is the vendor's next scheduled payout: net of delivered-order
// revenue for the role, with the payout date driven by the reconciliation
// cadence (docs/PAYMENTS_REDESIGN.md).
type PayoutSummary struct {
	NextPayoutDate string  `json:"next_payout_date"`
	Amount         float64 `json:"amount"`
	Status         string  `json:"status"`
}

// PayoutSummary computes the vendor payout from delivered orders of the given
// role (cart_type), net of platform commission, dated to the next
// reconciliation run.
func (r *AnalyticsRepository) PayoutSummary(ctx context.Context, role string) (PayoutSummary, error) {
	cartType := "grocery"
	if roleToKind(role) == "food" {
		cartType = "food"
	}
	var gross float64
	if err := r.db.QueryRow(ctx,
		`SELECT coalesce(sum(total_amount),0) FROM orders
		 WHERE lower(status) = 'delivered' AND cart_type = $1`, cartType).
		Scan(&gross); err != nil {
		return PayoutSummary{}, err
	}
	net := gross * (1 - platformCommissionPct)

	// Payout date follows the reconciliation cadence. Prefer the config's
	// precomputed next_run_on; fall back to a cadence-based estimate.
	var cadence string
	var nextRun *time.Time
	_ = r.db.QueryRow(ctx,
		`SELECT cadence, next_run_on FROM reconciliation_config LIMIT 1`).
		Scan(&cadence, &nextRun)

	var date time.Time
	switch {
	case nextRun != nil:
		date = *nextRun
	case cadence == "weekly":
		date = time.Now().AddDate(0, 0, 7)
	case cadence == "monthly":
		date = time.Now().AddDate(0, 1, 0)
	default:
		date = time.Now().AddDate(0, 0, 1)
	}

	status := "pending"
	if net > 0 {
		status = "scheduled"
	}
	return PayoutSummary{
		NextPayoutDate: date.Format("2006-01-02"),
		Amount:         net,
		Status:         status,
	}, nil
}

// ── Vendor store status (open/closed) ───────────────────────────────────────

// GetStoreStatus returns whether the vendor's store is open (defaults to open
// when no row exists yet).
func (r *AnalyticsRepository) GetStoreStatus(ctx context.Context, userID string) (bool, error) {
	var isOpen bool
	err := r.db.QueryRow(ctx,
		`SELECT is_open FROM vendor_store_status WHERE user_id = $1`, userID).
		Scan(&isOpen)
	if err != nil {
		if err == pgx.ErrNoRows {
			return true, nil
		}
		return true, err
	}
	return isOpen, nil
}

// SetStoreStatus upserts the vendor's open/closed switch.
func (r *AnalyticsRepository) SetStoreStatus(ctx context.Context, userID string, isOpen bool) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO vendor_store_status (user_id, is_open, updated_on)
		 VALUES ($1, $2, CURRENT_TIMESTAMP)
		 ON CONFLICT (user_id) DO UPDATE SET is_open = $2, updated_on = CURRENT_TIMESTAMP`,
		userID, isOpen)
	return err
}

// ── Search-demand logging ────────────────────────────────────────────────────

// LogSearch records one customer search query and how many catalog items it
// matched. Zero/low result_count is what surfaces as unmet demand for vendors.
func (r *AnalyticsRepository) LogSearch(ctx context.Context, userID, query, city, kind string, resultCount int) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	id := "SRCH" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err := r.db.Exec(ctx,
		`INSERT INTO search_logs (id, user_id, query, city, result_count, kind)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, nullIfEmptyStr(userID), query, nullIfEmptyStr(city), resultCount, nullIfEmptyStr(kind))
	return err
}

func nullIfEmptyStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// UnmetDemand aggregates search_logs into the queries that repeatedly returned
// no (or almost no) catalog results in the window — the strongest signal of
// products a vendor should add.
func (r *AnalyticsRepository) UnmetDemand(ctx context.Context, role, timeframe string) ([]UnmetDemand, error) {
	days := timeframeDays(timeframe)
	kind := roleToKind(role)
	const q = `
SELECT lower(query) AS q,
       count(*) AS search_count,
       coalesce(max(city), '') AS city,
       to_char(max(created_on), 'YYYY-MM-DD"T"HH24:MI:SS') AS last_seen
FROM search_logs
WHERE created_on >= now() - make_interval(days => $1)
  AND result_count = 0
  AND (kind IS NULL OR kind = $2)
GROUP BY lower(query)
ORDER BY search_count DESC, last_seen DESC
LIMIT 10`
	rows, err := r.db.Query(ctx, q, days, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UnmetDemand{}
	for rows.Next() {
		var u UnmetDemand
		if err := rows.Scan(&u.Query, &u.SearchCount, &u.City, &u.LastSeenAt); err != nil {
			return nil, err
		}
		// A rough uplift proxy: more repeated misses ⇒ more potential revenue.
		u.PotentialUplift = float64(u.SearchCount) * 25.0
		out = append(out, u)
	}
	return out, rows.Err()
}

// UnmetDemand is one search query that repeatedly returned no results.
type UnmetDemand struct {
	Query           string  `json:"query"`
	SearchCount     int     `json:"search_count"`
	City            string  `json:"city"`
	LastSeenAt      string  `json:"last_seen_at"`
	PotentialUplift float64 `json:"potential_uplift"`
}

// UnmetDemandAdmin is the platform-wide view of UnmetDemand: unlike the
// per-role vendor version, it doesn't filter by kind (grocery vs food) at
// all, since an admin needs visibility across every partner type.
func (r *AnalyticsRepository) UnmetDemandAdmin(ctx context.Context, timeframe string) ([]UnmetDemand, error) {
	days := timeframeDays(timeframe)
	const q = `
SELECT lower(query) AS q,
       count(*) AS search_count,
       coalesce(max(city), '') AS city,
       to_char(max(created_on), 'YYYY-MM-DD"T"HH24:MI:SS') AS last_seen
FROM search_logs
WHERE created_on >= now() - make_interval(days => $1)
  AND result_count = 0
GROUP BY lower(query)
ORDER BY search_count DESC, last_seen DESC
LIMIT 20`
	rows, err := r.db.Query(ctx, q, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UnmetDemand{}
	for rows.Next() {
		var u UnmetDemand
		if err := rows.Scan(&u.Query, &u.SearchCount, &u.City, &u.LastSeenAt); err != nil {
			return nil, err
		}
		u.PotentialUplift = float64(u.SearchCount) * 25.0
		out = append(out, u)
	}
	return out, rows.Err()
}

// NotifyUnmetDemand records that an admin flagged a query's unmet demand for
// partner follow-up. There's no push/SMS delivery infrastructure in this
// codebase yet, so this persists a real, auditable action rather than the
// button being a no-op.
func (r *AnalyticsRepository) NotifyUnmetDemand(ctx context.Context, query, adminID string) error {
	id := GetIdToRecord("UDN")
	_, err := r.db.Exec(ctx,
		`INSERT INTO unmet_demand_notifications (id, query, notified_by) VALUES ($1, $2, $3)`,
		id, query, adminID)
	return err
}

// LowSellers returns products of a kind that have sold at least once but are
// moving slowly — ranked by staleness (days since last sale) then fewest units
// in the window, so the partner can promote or delist them.
func (r *AnalyticsRepository) LowSellers(ctx context.Context, role, timeframe string) ([]LowSeller, error) {
	days := timeframeDays(timeframe)
	kind := roleToKind(role)
	const q = `
WITH sales AS (
  SELECT trim(e->>'product_id') AS pid,
         max(o.order_date) AS last_sold,
         sum(CASE WHEN o.order_date >= now() - make_interval(days => $1)
                  THEN (e->>'quantity')::int ELSE 0 END) AS units_win
  FROM orders o CROSS JOIN LATERAL jsonb_array_elements(o.items) e
  GROUP BY 1
)
SELECT trim(fp.id), fp.name, coalesce(fp.category,''),
       s.units_win,
       to_char(s.last_sold, 'YYYY-MM-DD"T"HH24:MI:SS'),
       greatest(extract(day from now() - s.last_sold)::int, 0) AS days_since
FROM sales s
JOIN food_products fp ON trim(fp.id) = s.pid AND fp.kind = $2
ORDER BY days_since DESC, s.units_win ASC
LIMIT 10`
	rows, err := r.db.Query(ctx, q, days, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LowSeller{}
	for rows.Next() {
		var t LowSeller
		if err := rows.Scan(&t.ProductID, &t.Name, &t.Category, &t.UnitsSold, &t.LastSoldAt, &t.DaysSinceSold); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
