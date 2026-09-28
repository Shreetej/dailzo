package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StorefrontRepository backs the customer-facing storefront: catalog browse,
// server-side cart, checkout, order history, complaints and refunds. It returns
// map[string]any rows (via pgx.RowToMap) so the JSON contract stays flexible and
// aligns directly with the Flutter models.
type StorefrontRepository struct {
	db *pgxpool.Pool
}

func NewStorefrontRepository(db *pgxpool.Pool) *StorefrontRepository {
	return &StorefrontRepository{db: db}
}

func collectMaps(rows pgx.Rows) ([]map[string]any, error) {
	defer rows.Close()
	out, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

// ─── Catalog ────────────────────────────────────────────────────────────────

func (r *StorefrontRepository) GetCategories(ctx context.Context, kind string) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, kind, parent_id, image_url, description, sort_order,
		       restaurant_id, outlet_id, is_active, department
		FROM categories
		WHERE kind = $1 AND is_active = true
		ORDER BY sort_order, name`, kind)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// GetProducts returns products filtered by optional category, outlet, restaurant and search text.
func (r *StorefrontRepository) GetProducts(ctx context.Context, categoryID, outletID, restaurantID, q string) ([]map[string]any, error) {
	// LEFT JOINs + the trailing AND clauses exclude products belonging to a
	// restaurant/grocery outlet whose vendor registration hasn't been
	// approved yet (is_active = false). A product with neither restaurant_id
	// nor outlet_id set (shouldn't normally happen) is left visible rather
	// than silently dropped.
	query := `
		SELECT fp.id, fp.name, fp.description, fp.category, fp.category_id, fp.kind, fp.type::text AS type,
		       fp.price::float8 AS price, fp.image_url, fp.is_active,
		       fp.stock_quantity, fp.is_promo, fp.promo_price::float8 AS promo_price,
		       fp.brand, fp.unit, fp.weight::float8 AS weight, fp.outlet_id, fp.restaurant_id
		FROM food_products fp
		LEFT JOIN restaurants r ON fp.restaurant_id = r.id
		LEFT JOIN grocery_profiles gp ON fp.outlet_id = gp.id
		WHERE fp.is_active = true
		  AND (fp.restaurant_id IS NULL OR r.is_active = true)
		  AND (fp.outlet_id IS NULL OR gp.is_active = true)`
	args := []any{}
	i := 1
	if categoryID != "" {
		query += fmt.Sprintf(" AND fp.category_id = $%d", i)
		args = append(args, categoryID)
		i++
	}
	if outletID != "" {
		query += fmt.Sprintf(" AND fp.outlet_id = $%d", i)
		args = append(args, outletID)
		i++
	}
	if restaurantID != "" {
		query += fmt.Sprintf(" AND fp.restaurant_id = $%d", i)
		args = append(args, restaurantID)
		i++
	}
	if q != "" {
		query += fmt.Sprintf(" AND (fp.name ILIKE $%d OR fp.description ILIKE $%d)", i, i)
		args = append(args, "%"+q+"%")
		i++
	}
	// High cap so the customer app can pull the whole catalog in one call to
	// build the home/grocery views; category/restaurant-filtered calls return
	// far fewer rows anyway.
	query += " ORDER BY fp.name LIMIT 5000"
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// GetRestaurants lists seeded restaurants for the food home / detail lookup.
func (r *StorefrontRepository) GetRestaurants(ctx context.Context) ([]map[string]any, error) {
	// is_active excludes restaurants whose vendor registration hasn't been
	// approved yet (see repository/registration_repository.go
	// createOperationalProfile) — this query previously had no filter at
	// all, so an unapproved restaurant was visible to every customer the
	// moment its owner submitted registration.
	rows, err := r.db.Query(ctx, `
		SELECT id, name, image_path, cuisines, delivery_time, location, address
		FROM restaurants
		WHERE is_active = true
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// ─── Cart (server-side, per user) ─────────────────────────────────────────────

// GetCart returns cart lines joined to product/variant details.
func (r *StorefrontRepository) GetCart(ctx context.Context, userID, cartType string) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ci.id, ci.cart_type, ci.product_id, ci.variant_id, ci.quantity,
		       fp.name, fp.image_url, fp.price::float8 AS price,
		       fp.promo_price::float8 AS promo_price, fp.is_promo, fp.unit, fp.brand,
		       pv.variant_name, pv.price::float8 AS variant_price
		FROM cart_items ci
		JOIN food_products fp ON fp.id = ci.product_id
		LEFT JOIN product_variants pv ON pv.id = ci.variant_id
		WHERE ci.user_id = $1 AND ci.cart_type = $2
		ORDER BY ci.added_on`, userID, cartType)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// AddCartItem upserts a line: if the same (user,type,product,variant) exists,
// its quantity is incremented by qty.
func (r *StorefrontRepository) AddCartItem(ctx context.Context, userID, cartType, productID, variantID string, qty int) (string, error) {
	if qty <= 0 {
		qty = 1
	}
	id := GetIdToRecord("CART")
	var variant any
	if variantID != "" {
		variant = variantID
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO cart_items (id, user_id, cart_type, product_id, variant_id, quantity, added_on, last_updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (user_id, cart_type, product_id, COALESCE(variant_id, ''))
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity,
		              last_updated_on = EXCLUDED.last_updated_on`,
		id, userID, cartType, productID, variant, qty, time.Now())
	return id, err
}

// SetCartItemQty sets an absolute quantity; qty<=0 deletes the line.
func (r *StorefrontRepository) SetCartItemQty(ctx context.Context, itemID string, qty int) error {
	if qty <= 0 {
		return r.DeleteCartItem(ctx, itemID)
	}
	_, err := r.db.Exec(ctx,
		`UPDATE cart_items SET quantity = $2, last_updated_on = $3 WHERE id = $1`,
		itemID, qty, time.Now())
	return err
}

func (r *StorefrontRepository) DeleteCartItem(ctx context.Context, itemID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE id = $1`, itemID)
	return err
}

func (r *StorefrontRepository) ClearCart(ctx context.Context, userID, cartType string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND cart_type = $2`, userID, cartType)
	return err
}

// ─── Checkout (cart → order, transactional) ───────────────────────────────────

// Checkout reads the user's server cart for cartType, snapshots it into an order,
// decrements product stock, clears the cart — all in one transaction.
func (r *StorefrontRepository) Checkout(ctx context.Context, userID string, meta map[string]any) (map[string]any, error) {
	cartType, _ := meta["cart_type"].(string)
	if cartType == "" {
		cartType = "food"
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT ci.product_id, ci.variant_id, ci.quantity,
		       fp.name, fp.image_url, fp.price::float8 AS price,
		       fp.promo_price::float8 AS promo_price, fp.is_promo,
		       pv.variant_name, pv.price::float8 AS variant_price
		FROM cart_items ci
		JOIN food_products fp ON fp.id = ci.product_id
		LEFT JOIN product_variants pv ON pv.id = ci.variant_id
		WHERE ci.user_id = $1 AND ci.cart_type = $2`, userID, cartType)
	if err != nil {
		return nil, err
	}
	cart, err := collectMaps(rows)
	if err != nil {
		return nil, err
	}
	if len(cart) == 0 {
		return nil, errors.New("cart is empty")
	}

	items := make([]map[string]any, 0, len(cart))
	var subtotal float64
	restaurantName := strVal(meta["restaurant_name"])
	var firstImage string
	for _, line := range cart {
		qty := toInt(line["quantity"])
		price := toFloat(line["price"])
		if line["variant_price"] != nil {
			price = toFloat(line["variant_price"])
		} else if b, ok := line["is_promo"].(bool); ok && b && line["promo_price"] != nil {
			price = toFloat(line["promo_price"])
		}
		subtotal += price * float64(qty)
		if firstImage == "" {
			firstImage, _ = line["image_url"].(string)
		}
		items = append(items, map[string]any{
			"product_id": line["product_id"],
			"name":       line["name"],
			"price":      price,
			"quantity":   qty,
			"variant":    line["variant_name"],
			"image_url":  line["image_url"],
		})
	}
	itemsJSON, _ := json.Marshal(items)

	deliveryFee := toFloat(meta["delivery_fee"])
	tax := toFloat(meta["tax_amount"])
	tip := toFloat(meta["tip"])
	total := subtotal + deliveryFee + tax + tip

	orderID := GetIdToRecord("ORDR")
	restaurantID, _ := meta["restaurant_id"].(string)
	now := time.Now()
	// Server-generated OTPs so they are stable and authoritative. delivery_otp
	// is the customer→partner handoff; pickup_otp is the restaurant→partner one.
	deliveryOTP := fmt.Sprintf("%04d", now.UnixNano()%10000)
	pickupOTP := fmt.Sprintf("%04d", (now.UnixNano()/7)%10000)

	// The client only ever sends restaurant_name (not restaurant_id), so look
	// the restaurant's stated delivery_time ("30-40") up by name to compute a
	// real ETA instead of leaving estimated_delivery_time NULL forever.
	var restaurantDeliveryTime string
	if restaurantName != "" {
		_ = tx.QueryRow(ctx,
			`SELECT delivery_time FROM restaurants WHERE name = $1 LIMIT 1`,
			restaurantName).Scan(&restaurantDeliveryTime)
	}
	etaMinutes := parseDeliveryMinutes(restaurantDeliveryTime)
	estimatedDeliveryTime := now.Add(time.Duration(etaMinutes) * time.Minute)

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, user_id, restaurant_id, status, delivery_status, total_amount,
		                    order_date, cart_type, items, customer_name, restaurant_name,
		                    delivery_address, payment_method, tip, subtotal, delivery_fee,
		                    tax_amount, image_path, delivery_otp, pickup_otp, estimated_delivery_time,
		                    created_on, last_updated_on)
		VALUES ($1,$2,$3,'pending','preparing',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$5,$5)`,
		orderID, userID, nullIfEmpty(restaurantID), total, now, cartType, string(itemsJSON),
		strVal(meta["customer_name"]), restaurantName, strVal(meta["delivery_address"]),
		strVal(meta["payment_method"]), tip, subtotal, deliveryFee, tax, firstImage, deliveryOTP, pickupOTP,
		estimatedDeliveryTime)
	if err != nil {
		return nil, err
	}

	// decrement stock best-effort (only for products that track stock)
	for _, line := range cart {
		pid, _ := line["product_id"].(string)
		qty := toInt(line["quantity"])
		_, _ = tx.Exec(ctx,
			`UPDATE food_products SET stock_quantity = GREATEST(stock_quantity - $2, 0)
			 WHERE id = $1 AND stock_quantity IS NOT NULL`, pid, qty)
	}

	if _, err = tx.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND cart_type = $2`, userID, cartType); err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return map[string]any{
		"id": orderID, "orderId": orderID, "status": "pending",
		"total_amount": total, "orderAmt": total, "items": items,
		"cart_type": cartType, "restaurant_name": restaurantName,
		"delivery_otp": deliveryOTP, "estimated_delivery_time": estimatedDeliveryTime,
	}, nil
}

// ─── Orders (customer view) ───────────────────────────────────────────────────

func (r *StorefrontRepository) GetMyOrders(ctx context.Context, userID string) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, restaurant_id, restaurant_name, status, delivery_status,
		       total_amount::float8 AS total_amount, subtotal::float8 AS subtotal,
		       delivery_fee::float8 AS delivery_fee, tax_amount::float8 AS tax_amount,
		       tip::float8 AS tip, order_date, cart_type, items, customer_name,
		       delivery_address, payment_method, image_path, delivery_otp,
		       estimated_delivery_time
		FROM orders
		WHERE user_id = $1
		ORDER BY order_date DESC NULLS LAST LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// GetAllOrders returns recent orders (vendor view), newest first, with the
// same rich snapshot (items, customer, totals). Optional status filter (empty
// = all). outletID, when non-empty, restricts results to that outlet or
// restaurant (matches either column, mirroring OrderRepository.GetOrdersWithFilters)
// so one vendor's login can't see another outlet's queue.
//
// Note: outletID is trusted as supplied by the caller — there is currently no
// server-side mapping from the authenticated user to the outlet(s) they own,
// so this filters what the client asks for rather than enforcing ownership.
// Closing that trust gap needs a real user->restaurant/outlet ownership table.
func (r *StorefrontRepository) GetAllOrders(ctx context.Context, status, outletID string) ([]map[string]any, error) {
	query := `
		SELECT id, user_id, restaurant_id, restaurant_name, status, delivery_status,
		       total_amount::float8 AS total_amount, subtotal::float8 AS subtotal,
		       delivery_fee::float8 AS delivery_fee, tax_amount::float8 AS tax_amount,
		       tip::float8 AS tip, order_date, cart_type, items, customer_name,
		       delivery_address, payment_method, image_path, delivery_otp
		FROM orders`
	conditions := []string{}
	args := []any{}
	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if outletID != "" {
		args = append(args, outletID)
		conditions = append(conditions, fmt.Sprintf("(restaurant_id = $%d OR outlet_id = $%d)", len(args), len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY order_date DESC NULLS LAST LIMIT 200`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// UpdateOrderStatus sets an order's status (vendor/delivery progressing the
// order); the change is immediately visible to the customer via GetMyOrders /
// GetOrder. When the status reflects a delivery milestone it also mirrors the
// delivery_status column, advances the matching delivery_tasks row, and logs a
// trace event so the delivery dashboard + customer tracking stay in sync.
func (r *StorefrontRepository) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	now := time.Now()
	// Map the order status onto a customer-facing delivery_status.
	deliveryStatus := ""
	switch status {
	case "picked_up", "out_for_delivery":
		deliveryStatus = "out_for_delivery"
	case "delivered":
		deliveryStatus = "delivered"
	case "ready":
		deliveryStatus = "ready"
	}
	if deliveryStatus != "" {
		_, err := r.db.Exec(ctx,
			`UPDATE orders SET status = $2, delivery_status = $3, last_updated_on = $4 WHERE id = $1`,
			orderID, status, deliveryStatus, now)
		if err != nil {
			return err
		}
	} else {
		if _, err := r.db.Exec(ctx,
			`UPDATE orders SET status = $2, last_updated_on = $3 WHERE id = $1`,
			orderID, status, now); err != nil {
			return err
		}
	}

	// Advance the delivery task + log a trace event for delivery milestones.
	taskStatus, traceStatus, note := "", "", ""
	tsCol := ""
	switch status {
	case "picked_up":
		taskStatus, traceStatus, note, tsCol = "picked_up", "picked_up", "Order picked up by delivery partner", "picked_up_at"
	case "out_for_delivery":
		taskStatus, traceStatus, note = "in_transit", "out_for_delivery", "Out for delivery"
	case "delivered":
		taskStatus, traceStatus, note, tsCol = "delivered", "delivered", "Order delivered to customer", "delivered_at"
	}
	if taskStatus != "" {
		q := `UPDATE delivery_tasks SET status=$2, last_updated_on=$3`
		if tsCol != "" {
			q += fmt.Sprintf(", %s=$3", tsCol)
		}
		q += ` WHERE order_id=$1 AND status NOT IN ('delivered','cancelled')`
		_, _ = r.db.Exec(ctx, q, orderID, taskStatus, now)
		_, _ = r.db.Exec(ctx, `
			INSERT INTO delivery_trace_events (id, order_id, status, actor_type, notes)
			VALUES ($1,$2,$3,'delivery_partner',$4)`,
			GetIdToRecord("DTRC"), orderID, traceStatus, note)
	}
	return nil
}

func (r *StorefrontRepository) GetOrder(ctx context.Context, orderID string) (map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, restaurant_id, restaurant_name, status, delivery_status,
		       total_amount::float8 AS total_amount, subtotal::float8 AS subtotal,
		       delivery_fee::float8 AS delivery_fee, tax_amount::float8 AS tax_amount,
		       tip::float8 AS tip, order_date, cart_type, items, customer_name,
		       delivery_address, payment_method, image_path, delivery_otp, pickup_otp,
		       estimated_delivery_time
		FROM orders WHERE id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	list, err := collectMaps(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("order not found")
	}
	return list[0], nil
}

// ─── Complaints (customer) ────────────────────────────────────────────────────

func (r *StorefrontRepository) CreateComplaint(ctx context.Context, userID string, body map[string]any) (map[string]any, error) {
	id := GetIdToRecord("CMP")
	now := time.Now()
	// Resolve order_id via a subquery so a stale/unknown reference collapses to
	// NULL instead of violating the FK (which would 500 the whole request).
	_, err := r.db.Exec(ctx, `
		INSERT INTO complaints (id, order_id, user_id, complaint_type, description,
		                        status, priority, created_on, last_updated_on)
		VALUES ($1,(SELECT id FROM orders WHERE id=$2),$3,$4,$5,'open',$6,$7,$7)`,
		id, nullIfEmpty(strVal(body["order_id"])), userID,
		strVal(body["complaint_type"]), strVal(body["description"]),
		defaultStr(strVal(body["priority"]), "medium"), now)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "status": "open"}, nil
}

func (r *StorefrontRepository) GetComplaints(ctx context.Context, userID string) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, user_id, complaint_type, description, status, priority,
		       resolution_notes, refund_amount::float8 AS refund_amount, created_on, resolved_on
		FROM complaints WHERE user_id = $1 ORDER BY created_on DESC`, userID)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// ─── Refunds (customer) ───────────────────────────────────────────────────────

func (r *StorefrontRepository) CreateRefund(ctx context.Context, userID string, body map[string]any) (map[string]any, error) {
	id := GetIdToRecord("RFND")
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		INSERT INTO refunds (id, payment_id, user_id, amount, reason, status,
		                     refund_date, created_on, last_updated_on, created_by)
		VALUES ($1,$2,$3,$4,$5,'requested',$6,$6,$6,$3)`,
		id, nullIfEmpty(strVal(body["payment_id"])), userID,
		toFloat(body["amount"]), strVal(body["reason"]), now)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "status": "requested"}, nil
}

func (r *StorefrontRepository) GetRefunds(ctx context.Context, userID string) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, payment_id, user_id, amount::float8 AS amount, reason, status, refund_date, created_on
		FROM refunds WHERE user_id = $1 ORDER BY created_on DESC`, userID)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// ─── small helpers ────────────────────────────────────────────────────────────

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

var deliveryMinutesPattern = regexp.MustCompile(`\d+`)

// parseDeliveryMinutes reads a restaurant's "delivery_time" string (e.g.
// "30-40" or "25") and returns its upper bound in minutes, defaulting to 30
// when the string is empty or has no digits.
func parseDeliveryMinutes(s string) int {
	matches := deliveryMinutesPattern.FindAllString(s, -1)
	if len(matches) == 0 {
		return 30
	}
	mins, err := strconv.Atoi(matches[len(matches)-1])
	if err != nil || mins <= 0 {
		return 30
	}
	return mins
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
