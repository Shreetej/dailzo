package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
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
	// Fees and VAT are a server-side policy, not client input — see
	// BillingRepository.QuoteCart, which both the quote endpoint and Checkout
	// price through.
	billing *BillingRepository
}

func NewStorefrontRepository(db *pgxpool.Pool) *StorefrontRepository {
	return &StorefrontRepository{db: db, billing: NewBillingRepository(db)}
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
	// rating/total_ratings are returned so the apps can show the real score.
	// They were absent, which is why the customer app fell back to a
	// hardcoded "4.2 (100)" on every restaurant card.
	rows, err := r.db.Query(ctx, `
		SELECT id, name, image_path, cuisines, delivery_time, location, address,
		       COALESCE(rating, 0)::float8 AS rating,
		       COALESCE(total_ratings, 0)  AS total_ratings
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
	// The selected options come back both as raw ids (so the client can
	// re-open the line for editing) and as a priced, ordered label list, so a
	// restored cart shows "Large, Extra cheese" without a second round trip.
	rows, err := r.db.Query(ctx, `
		SELECT ci.id, ci.cart_type, ci.product_id, ci.variant_id, ci.quantity,
		       fp.name, fp.image_url, fp.price::float8 AS price,
		       fp.promo_price::float8 AS promo_price, fp.is_promo, fp.unit, fp.brand,
		       pv.variant_name, pv.price::float8 AS variant_price,
		       ci.selected_option_ids,
		       COALESCE(opt.options, '[]'::jsonb) AS selected_options,
		       COALESCE(opt.delta, 0)::float8     AS options_delta
		FROM cart_items ci
		JOIN food_products fp ON fp.id = ci.product_id
		LEFT JOIN product_variants pv ON pv.id = ci.variant_id
		LEFT JOIN LATERAL (
		    SELECT jsonb_agg(jsonb_build_object(
		               'id', po.id, 'label', po.label,
		               'price_delta', po.price_delta::float8,
		               'group', g.name) ORDER BY g.sort_order, po.sort_order) AS options,
		           sum(po.price_delta) AS delta
		    FROM product_options po
		    JOIN product_option_groups g ON g.id = po.group_id
		    WHERE po.id IN (SELECT jsonb_array_elements_text(ci.selected_option_ids))
		      AND g.product_id = ci.product_id
		) opt ON TRUE
		WHERE ci.user_id = $1 AND ci.cart_type = $2
		ORDER BY ci.added_on`, userID, cartType)
	if err != nil {
		return nil, err
	}
	return collectMaps(rows)
}

// AddCartItem upserts a line. Two lines for the same product with different
// option selections are different lines, so the selection participates in the
// conflict key alongside (user, type, product, variant).
func (r *StorefrontRepository) AddCartItem(
	ctx context.Context,
	userID, cartType, productID, variantID string,
	qty int,
	selectedOptionIDs []string,
) (string, error) {
	if qty <= 0 {
		qty = 1
	}
	id := GetIdToRecord("CART")
	var variant any
	if variantID != "" {
		variant = variantID
	}

	// Sorted + de-duplicated so the same choices always hash identically —
	// otherwise picking "cheese, gravy" and "gravy, cheese" would create two
	// separate lines for the same thing.
	optionsJSON, err := canonicalOptionIDsJSON(selectedOptionIDs)
	if err != nil {
		return "", err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO cart_items (id, user_id, cart_type, product_id, variant_id, quantity, selected_option_ids, added_on, last_updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (user_id, cart_type, product_id, COALESCE(variant_id, ''::bpchar), md5(selected_option_ids::text))
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity,
		              last_updated_on = EXCLUDED.last_updated_on`,
		id, userID, cartType, productID, variant, qty, optionsJSON, time.Now())
	return id, err
}

// canonicalOptionIDsJSON normalises a selection into the JSONB array stored on
// the cart line: trimmed, de-duplicated and sorted.
func canonicalOptionIDsJSON(ids []string) (string, error) {
	seen := map[string]bool{}
	clean := make([]string, 0, len(ids))
	for _, raw := range ids {
		v := strings.TrimSpace(raw)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		clean = append(clean, v)
	}
	sort.Strings(clean)

	encoded, err := json.Marshal(clean)
	if err != nil {
		return "", fmt.Errorf("failed to encode option selection: %w", err)
	}
	return string(encoded), nil
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
	// The apps present ONE cart ("Mix Food & Grocery in one order — one cart,
	// one delivery"), so an order can span both types. "mixed" takes every
	// line the user has; "food"/"grocery" keep the single-type behaviour for
	// any caller that still asks for it.
	mixed := cartType == "mixed" || cartType == "all"
	typeFilter := " AND ci.cart_type = $2"
	queryArgs := []any{userID, cartType}
	if mixed {
		typeFilter = ""
		queryArgs = []any{userID}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// fp.outlet_id / fp.restaurant_id come along so the order can record WHICH
	// vendor it belongs to. Grocery carts in particular never carried that:
	// the client only ever sends restaurant_name, groceries have no
	// restaurant row at all, and outlet_id was never written — so a grocery
	// order landed attributable to nobody and no grocery vendor could list
	// their own orders.
	rows, err := tx.Query(ctx, `
		SELECT ci.product_id, ci.variant_id, ci.quantity,
		       fp.name, fp.image_url, fp.price::float8 AS price,
		       fp.promo_price::float8 AS promo_price, fp.is_promo,
		       fp.outlet_id, fp.restaurant_id,
		       pv.variant_name, pv.price::float8 AS variant_price,
		       COALESCE(opt.labels, '[]'::jsonb) AS option_labels,
		       COALESCE(opt.delta, 0)::float8    AS options_delta
		FROM cart_items ci
		JOIN food_products fp ON fp.id = ci.product_id
		LEFT JOIN product_variants pv ON pv.id = ci.variant_id
		-- Price the customer's choices HERE, from product_options, rather than
		-- trusting anything the client sends. The uplift was previously lost
		-- entirely, so a customised line was billed at its base price.
		LEFT JOIN LATERAL (
		    SELECT jsonb_agg(jsonb_build_object(
		               'id', po.id, 'label', po.label,
		               'price_delta', po.price_delta::float8,
		               'group', g.name) ORDER BY g.sort_order, po.sort_order) AS labels,
		           sum(po.price_delta) AS delta
		    FROM product_options po
		    JOIN product_option_groups g ON g.id = po.group_id
		    WHERE po.id IN (SELECT jsonb_array_elements_text(ci.selected_option_ids))
		      AND g.product_id = ci.product_id
		      AND po.is_active AND g.is_active
		) opt ON TRUE
		WHERE ci.user_id = $1`+typeFilter, queryArgs...)
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
	// Pre-discount total, for the savings line on the bill.
	var mrpTotal float64
	restaurantName := strVal(meta["restaurant_name"])
	var firstImage string
	// The vendor this cart belongs to, taken from the products themselves
	// rather than trusted from the client. A cart is single-vendor by
	// construction (the apps clear it when switching store), so the first
	// non-empty value wins.
	var cartOutletID, cartRestaurantID string
	for _, line := range cart {
		if cartOutletID == "" {
			cartOutletID, _ = line["outlet_id"].(string)
		}
		if cartRestaurantID == "" {
			cartRestaurantID, _ = line["restaurant_id"].(string)
		}
		qty := toInt(line["quantity"])
		price := toFloat(line["price"])
		if line["variant_price"] != nil {
			price = toFloat(line["variant_price"])
		} else if b, ok := line["is_promo"].(bool); ok && b && line["promo_price"] != nil {
			price = toFloat(line["promo_price"])
		}
		// The chosen options are an uplift on whichever base price applies,
		// priced from product_options above. Without this the order recorded
		// the base price while the payment captured the customised one.
		optionsDelta := toFloat(line["options_delta"])
		price += optionsDelta

		// List price before any promo, with the same option uplift applied,
		// so savings compares like with like.
		mrp := toFloat(line["price"]) + optionsDelta
		if mrp < price {
			mrp = price
		}
		mrpTotal += mrp * float64(qty)

		subtotal += price * float64(qty)
		if firstImage == "" {
			firstImage, _ = line["image_url"].(string)
		}

		item := map[string]any{
			"product_id": line["product_id"],
			"name":       line["name"],
			"price":      price,
			"quantity":   qty,
			"variant":    line["variant_name"],
			"image_url":  line["image_url"],
			// Which vendor owes this line. A mixed food+grocery order spans
			// two vendors by design, and the order row has room for only one
			// (restaurant_id / outlet_id); without per-line attribution the
			// grocery half of such an order was invisible to the grocery
			// store, while the restaurant saw groceries it had not sold.
			// Trimmed: some id columns are bpchar, so a padded value would
			// never compare equal to the vendor's own id.
			"outlet_id":     strings.TrimSpace(strVal(line["outlet_id"])),
			"restaurant_id": strings.TrimSpace(strVal(line["restaurant_id"])),
		}
		// Carry the selection onto the order so the vendor knows what to make
		// and the line can be explained back to the customer.
		if labels, ok := line["option_labels"]; ok && labels != nil {
			item["options"] = labels
			item["options_delta"] = optionsDelta
		}
		items = append(items, item)
	}
	itemsJSON, _ := json.Marshal(items)

	// Fees and VAT are priced from the server's billing policy for this
	// market, NOT from the request body. The client used to send
	// delivery_fee and tax_amount — constants compiled into the app — and
	// this function accepted them, so the amount charged was whatever the
	// device claimed and could only be changed by shipping a new build.
	//
	// Only the tip is taken from the client: it is the customer's own choice,
	// and it is excluded from VAT below.
	tip := toFloat(meta["tip"])
	regionCode, _ := meta["region_code"].(string)
	billingCfg, err := r.billing.GetBillingConfig(ctx, regionCode)
	if err != nil {
		return nil, fmt.Errorf("failed to price order: %w", err)
	}
	quote := r.billing.QuoteCart(ctx, billingCfg, subtotal, mrpTotal,
		toFloat(meta["discount"]), tip, len(items))

	deliveryFee := quote.DeliveryFee + quote.HandlingFee + quote.SmallOrderFee
	tax := quote.Tax
	total := quote.Total
	subtotal = quote.Subtotal

	orderID := GetIdToRecord("ORDR")
	restaurantID, _ := meta["restaurant_id"].(string)
	// Fall back to the vendor derived from the cart's own products when the
	// client didn't name one — which is always, for grocery.
	if restaurantID == "" {
		restaurantID = cartRestaurantID
	}
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
		INSERT INTO orders (id, user_id, restaurant_id, outlet_id, status, delivery_status, total_amount,
		                    order_date, cart_type, items, customer_name, restaurant_name,
		                    delivery_address, payment_method, tip, subtotal, delivery_fee,
		                    tax_amount, image_path, delivery_otp, pickup_otp, estimated_delivery_time,
		                    created_on, last_updated_on)
		VALUES ($1,$2,$3,$20,'pending','preparing',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$5,$5)`,
		orderID, userID, nullIfEmpty(restaurantID), total, now, cartType, string(itemsJSON),
		strVal(meta["customer_name"]), restaurantName, strVal(meta["delivery_address"]),
		strVal(meta["payment_method"]), tip, subtotal, deliveryFee, tax, firstImage, deliveryOTP, pickupOTP,
		estimatedDeliveryTime, nullIfEmpty(cartOutletID))
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

	// Clear exactly what was just ordered — both types for a mixed order.
	if mixed {
		_, err = tx.Exec(ctx, `DELETE FROM cart_items WHERE user_id = $1`, userID)
	} else {
		_, err = tx.Exec(ctx,
			`DELETE FROM cart_items WHERE user_id = $1 AND cart_type = $2`, userID, cartType)
	}
	if err != nil {
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
// orderSelectColumns is shared by the vendor-scoped and unscoped order reads
// so both return identical row shapes to the apps.
const orderSelectColumns = `
		SELECT id, user_id, restaurant_id, restaurant_name, status, delivery_status,
		       total_amount::float8 AS total_amount, subtotal::float8 AS subtotal,
		       delivery_fee::float8 AS delivery_fee, tax_amount::float8 AS tax_amount,
		       tip::float8 AS tip, order_date, cart_type, items, customer_name,
		       delivery_address, payment_method, image_path, delivery_otp`

// GetVendorOrders lists orders belonging to one vendor. scope is the caller's
// own businesses/outlets and is always applied; outletID, when given, narrows
// further within that scope (the controller has already checked it is one of
// the caller's). A vendor who owns nothing gets an empty list, never the
// unfiltered table.
func (r *StorefrontRepository) GetVendorOrders(
	ctx context.Context,
	status, outletID string,
	scope VendorScope,
) ([]map[string]any, error) {
	if scope.IsEmpty() {
		return []map[string]any{}, nil
	}

	// Match the caller's full id set against BOTH columns — see
	// VendorScope.AllIDs: a grocery store's id is a "business" id but is
	// written to orders.outlet_id, so pairing each set with one column hid
	// every grocery order from its own store.
	// The order's own columns name at most one vendor, so a mixed order would
	// otherwise reach only whichever vendor happened to be first in the cart.
	// The items snapshot carries each line's vendor, so match that as well.
	query := orderSelectColumns + ` FROM orders o`
	conditions := []string{`(
		o.restaurant_id = ANY($1) OR o.outlet_id = ANY($1)
		OR EXISTS (
		    SELECT 1 FROM jsonb_array_elements(COALESCE(o.items, '[]'::jsonb)) line
		    WHERE line->>'outlet_id' = ANY($1) OR line->>'restaurant_id' = ANY($1)
		)
	)`}
	args := []any{scope.AllIDs()}

	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if outletID != "" {
		// An outlet belongs to a business, and orders are attributed to
		// whichever of the two the product named — a restaurant's dishes carry
		// its restaurant_id, not the id of the outlet that cooks them. Narrow
		// to the outlet AND its parent business, or asking for "this outlet's
		// orders" returned nothing for every restaurant vendor.
		args = append(args, outletID)
		conditions = append(conditions, fmt.Sprintf(`(
			o.restaurant_id = $%[1]d OR o.outlet_id = $%[1]d
			OR o.restaurant_id = (SELECT restaurant_id FROM vendor_outlets WHERE id = $%[1]d)
			OR EXISTS (
			    SELECT 1 FROM jsonb_array_elements(COALESCE(o.items, '[]'::jsonb)) line
			    WHERE line->>'outlet_id' = $%[1]d
			       OR line->>'restaurant_id' = $%[1]d
			       OR line->>'restaurant_id' = (SELECT restaurant_id FROM vendor_outlets WHERE id = $%[1]d)
			)
		)`, len(args)))
	}

	query += " WHERE " + strings.Join(conditions, " AND ")
	query += ` ORDER BY o.order_date DESC NULLS LAST LIMIT 200`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	orders, err := collectMaps(rows)
	if err != nil {
		return nil, err
	}
	for _, order := range orders {
		restrictItemsToScope(order, scope)
	}
	return orders, nil
}

// restrictItemsToScope trims a mixed order's items down to the lines the
// caller actually sells, in place.
//
// One order can now span two vendors, and both of them list it. Neither should
// read the other's lines, and a kitchen shown someone else's groceries would
// try to prepare them.
//
// Lines without vendor attribution (orders placed before it was recorded) are
// kept as they are: there is nothing to decide with, and dropping them would
// empty historical orders.
func restrictItemsToScope(order map[string]any, scope VendorScope) {
	lines, ok := order["items"].([]any)
	if !ok || len(lines) == 0 {
		return
	}

	attributed := false
	mine := make([]any, 0, len(lines))
	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		if !ok {
			mine = append(mine, raw)
			continue
		}
		outletID := strings.TrimSpace(strVal(line["outlet_id"]))
		restaurantID := strings.TrimSpace(strVal(line["restaurant_id"]))
		if outletID == "" && restaurantID == "" {
			mine = append(mine, raw)
			continue
		}
		attributed = true
		if scope.Owns(outletID) || scope.Owns(restaurantID) {
			mine = append(mine, raw)
		}
	}
	if attributed {
		order["items"] = mine
	}
}

// GetAllOrders is the unscoped read, kept for admin/platform callers.
func (r *StorefrontRepository) GetAllOrders(ctx context.Context, status, outletID string) ([]map[string]any, error) {
	query := orderSelectColumns + ` FROM orders`
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

// ─── Product options (customisation) ─────────────────────────────────────────

// GetProductOptionGroups returns one product's customisation groups, each with
// its selectable options, ordered for display.
//
// This replaces the const option lists that were hardcoded in the customer
// app's customisation screen: those were keyed off tag strings, applied the
// same "Portion / Spice level / Add-ons" to every dish, carried invented
// prices, and — because nothing server-side modelled them — could never be
// priced or fulfilled.
func (r *StorefrontRepository) GetProductOptionGroups(ctx context.Context, productID string) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT g.id, g.name, g.selection, g.is_required, g.sort_order,
		       COALESCE(jsonb_agg(jsonb_build_object(
		           'id', o.id,
		           'label', o.label,
		           'price_delta', o.price_delta::float8
		       ) ORDER BY o.sort_order, o.label)
		       FILTER (WHERE o.id IS NOT NULL), '[]'::jsonb) AS options
		FROM product_option_groups g
		LEFT JOIN product_options o ON o.group_id = g.id AND o.is_active
		WHERE g.product_id = $1 AND g.is_active
		GROUP BY g.id, g.name, g.selection, g.is_required, g.sort_order
		ORDER BY g.sort_order, g.name`, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to query product options: %w", err)
	}
	return collectMaps(rows)
}

// QuoteCart prices the user's current cart under their market's policy.
//
// The customer app renders this verbatim: subtotal, fees, VAT and total all
// come from here, so the bill shown before checkout and the amount recorded on
// the order are produced by the same code. Previously the app computed the
// bill from constants it carried and sent the result along to be trusted.
func (r *StorefrontRepository) QuoteCart(
	ctx context.Context,
	userID, cartType, regionCode string,
	tip, discount float64,
) (CartQuote, error) {
	cfg, err := r.billing.GetBillingConfig(ctx, regionCode)
	if err != nil {
		return CartQuote{}, err
	}

	lines, err := r.GetCart(ctx, userID, cartType)
	if err != nil {
		return CartQuote{}, err
	}

	var subtotal, mrpTotal float64
	itemCount := 0
	for _, line := range lines {
		qty := toInt(line["quantity"])
		if qty <= 0 {
			continue
		}

		price := toFloat(line["price"])
		if line["variant_price"] != nil {
			price = toFloat(line["variant_price"])
		} else if b, ok := line["is_promo"].(bool); ok && b && line["promo_price"] != nil {
			price = toFloat(line["promo_price"])
		}
		optionsDelta := toFloat(line["options_delta"])
		price += optionsDelta

		mrp := toFloat(line["price"]) + optionsDelta
		if mrp < price {
			mrp = price
		}

		subtotal += price * float64(qty)
		mrpTotal += mrp * float64(qty)
		itemCount += qty
	}

	return r.billing.QuoteCart(ctx, cfg, subtotal, mrpTotal, discount, tip, itemCount), nil
}

// QuoteLine is one cart line as the client currently shows it.
type QuoteLine struct {
	ProductID         string   `json:"product_id"`
	Quantity          int      `json:"quantity"`
	SelectedOptionIDs []string `json:"selected_option_ids"`
}

// QuoteLines prices the lines the client is displaying.
//
// The quote originally priced the SERVER cart, which meant that whenever the
// local cart and the server cart drifted apart — an unauthenticated session, a
// failed sync, a reorder — the customer saw their items listed above a bill of
// zero. Pricing what is on screen removes that whole class of mismatch.
//
// The client only says WHICH products and how many; every price still comes
// from the database here (base/promo price, option deltas), and fees and VAT
// still come from the billing policy, so this is not client-supplied pricing.
func (r *StorefrontRepository) QuoteLines(
	ctx context.Context,
	cartType, regionCode string,
	tip, discount float64,
	lines []QuoteLine,
) (CartQuote, error) {
	cfg, err := r.billing.GetBillingConfig(ctx, regionCode)
	if err != nil {
		return CartQuote{}, err
	}

	var subtotal, mrpTotal float64
	itemCount := 0

	for _, line := range lines {
		if line.Quantity <= 0 || strings.TrimSpace(line.ProductID) == "" {
			continue
		}

		var price, promoPrice *float64
		var isPromo bool
		err := r.db.QueryRow(ctx, `
			SELECT price::float8, promo_price::float8, COALESCE(is_promo, false)
			FROM food_products WHERE id = $1`, line.ProductID).
			Scan(&price, &promoPrice, &isPromo)
		if err != nil {
			// An unknown product contributes nothing rather than failing the
			// whole bill.
			continue
		}
		if price == nil {
			continue
		}

		unit := *price
		if isPromo && promoPrice != nil {
			unit = *promoPrice
		}

		// Option deltas, restricted to options that really belong to this
		// product — a client can't invent a discount by sending another
		// product's option ids.
		var delta float64
		if len(line.SelectedOptionIDs) > 0 {
			_ = r.db.QueryRow(ctx, `
				SELECT COALESCE(sum(po.price_delta), 0)::float8
				FROM product_options po
				JOIN product_option_groups g ON g.id = po.group_id
				WHERE po.id = ANY($1) AND g.product_id = $2
				  AND po.is_active AND g.is_active`,
				line.SelectedOptionIDs, line.ProductID).Scan(&delta)
		}

		subtotal += (unit + delta) * float64(line.Quantity)
		mrpTotal += (*price + delta) * float64(line.Quantity)
		itemCount += line.Quantity
	}

	return r.billing.QuoteCart(ctx, cfg, subtotal, mrpTotal, discount, tip, itemCount), nil
}
