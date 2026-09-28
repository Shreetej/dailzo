package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInviteCodeRequired/ErrInvalidInviteCode are returned by
// SaveVendorRegistration for a "delivery_partner" submission so the
// controller can surface a 400 with a clear message instead of a generic
// 500 — riders must join an already-approved fleet via its invite code.
var (
	ErrInviteCodeRequired = errors.New("an invite code from your delivery fleet/agency is required")
	ErrInvalidInviteCode  = errors.New("invite code not found or the fleet is not yet approved")
)

// RegistrationRepository stores the Partner app's restaurant registration
// documents as JSONB and lists vendor outlets in the shape the app expects.
type RegistrationRepository struct {
	db *pgxpool.Pool
}

func NewRegistrationRepository(db *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

// CreateRegistration creates (or, if the caller already owns id, updates) a
// restaurant registration keyed to the authenticated user. userID must be
// non-empty — ownership is what makes the later payment/complete/read calls
// on this row safe to expose without a fresh ownership lookup elsewhere.
func (r *RegistrationRepository) CreateRegistration(ctx context.Context, userID string, payload map[string]interface{}) (map[string]interface{}, error) {
	id, _ := payload["id"].(string)
	if id == "" {
		id = GetIdToRecord("REST")
		payload["id"] = id
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO restaurant_registrations (id, user_id, payload, created_on, last_updated_on)
	          VALUES ($1, $2, $3::jsonb, $4, $4)
	          ON CONFLICT (id) DO UPDATE
	          SET payload = EXCLUDED.payload, last_updated_on = EXCLUDED.last_updated_on
	          WHERE restaurant_registrations.user_id = $2`

	tag, err := r.db.Exec(ctx, query, id, userID, string(raw), time.Now())
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errors.New("registration id already exists and is owned by another user")
	}
	return payload, nil
}

// SaveVendorRegistration upserts a partner's full onboarding submission
// (restaurant | grocery | delivery) as JSONB, keyed by the authenticated user.
//
// On the FIRST submission only (detected via the `xmax = 0` insert-vs-update
// trick on the upsert below), this also creates the operational profile row
// the rest of the system actually reads (grocery_profiles / restaurants) and
// queues it in `approvals` for admin review. Previously this method only
// ever wrote to vendor_registrations — a table nothing else in the backend
// joined against — so registering never produced a grocery_profiles or
// restaurants row, never appeared in any admin approval queue, and vendors
// were fully operational (could create products, receive orders) from the
// moment they submitted, regardless of approval.
func (r *RegistrationRepository) SaveVendorRegistration(ctx context.Context, userID, userType string, payload map[string]interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO vendor_registrations (user_id, user_type, payload, status, created_on, last_updated_on)
	          VALUES ($1, $2, $3::jsonb, 'submitted', $4, $4)
	          ON CONFLICT (user_id) DO UPDATE
	          SET user_type = EXCLUDED.user_type,
	              payload = EXCLUDED.payload,
	              last_updated_on = EXCLUDED.last_updated_on
	          RETURNING (xmax = 0) AS inserted`

	var isNew bool
	if err := tx.QueryRow(ctx, query, userID, userType, string(raw), time.Now()).Scan(&isNew); err != nil {
		return err
	}

	if isNew {
		if err := r.createOperationalProfile(ctx, tx, userID, userType, payload); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// createOperationalProfile creates the customer/admin-facing entity for a
// brand-new vendor registration and queues it for approval. Both entities
// start deactivated (grocery_profiles.is_active / restaurants.is_active =
// false) — they only become discoverable to customers, and their owner only
// able to create products, once AdminRepository.Approve flips that (see
// admin_repository.go Approve/Reject, and the is_active filters added to
// storefront_repository.go's GetRestaurants/GetProducts and
// food_products_repository.go's ownership check).
func (r *RegistrationRepository) createOperationalProfile(ctx context.Context, tx pgx.Tx, userID, userType string, payload map[string]interface{}) error {
	str := func(key string) string {
		v, _ := payload[key].(string)
		return v
	}
	boolean := func(key string) bool {
		v, _ := payload[key].(bool)
		return v
	}

	switch userType {
	case "grocery":
		var existingID string
		err := tx.QueryRow(ctx, `SELECT id FROM grocery_profiles WHERE user_id = $1`, userID).Scan(&existingID)
		if err == nil {
			return nil // profile already exists (e.g. re-registration after a prior run) — don't duplicate
		}
		if err != pgx.ErrNoRows {
			return fmt.Errorf("failed to check existing grocery profile: %w", err)
		}

		bankDetails, _ := json.Marshal(map[string]string{
			"bank_name":        str("bank_name"),
			"account_number":   str("account_number"),
			"ifsc":             str("ifsc"),
			"beneficiary_name": str("beneficiary_name"),
		})

		groceryID := GetIdToRecord("GROC")
		now := time.Now()
		_, err = tx.Exec(ctx, `
			INSERT INTO grocery_profiles (
				id, user_id, store_name, owner_name, email, phone, address, city,
				pincode, fssai_license, gst_number, pan_number, working_hours,
				home_delivery, accepts_online_payment, store_type, store_size, service_area,
				bank_details, kyc_status, payout_status, is_active, rating, total_orders,
				commission_rate, created_on, last_updated_on
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,
				$19::jsonb, 'pending', 'pending', false, 0.0, 0, 15.0, $20, $20)`,
			groceryID, userID, str("store_name"), str("first_name"), str("email"), str("mobile_no"),
			str("address"), str("city"), str("pincode"), str("fssai_number"), str("gst_number"),
			str("pan_number"), "", boolean("home_delivery"), boolean("accepts_online_payment"),
			str("store_type"), str("store_size"), str("service_area"), string(bankDetails), now,
		)
		if err != nil {
			return fmt.Errorf("failed to create grocery profile: %w", err)
		}
		return r.queueApproval(ctx, tx, "grocery", groceryID)

	case "restaurant":
		var existingID string
		err := tx.QueryRow(ctx, `SELECT id FROM restaurants WHERE created_by = $1`, userID).Scan(&existingID)
		if err == nil {
			return nil
		}
		if err != pgx.ErrNoRows {
			return fmt.Errorf("failed to check existing restaurant: %w", err)
		}

		bankDetails, _ := json.Marshal(map[string]string{
			"bank_name":        str("bank_name"),
			"account_number":   str("account_number"),
			"ifsc":             str("ifsc"),
			"beneficiary_name": str("beneficiary_name"),
		})
		var kycDocuments []byte
		if docs, ok := payload["docs"].(map[string]interface{}); ok {
			kycDocuments, _ = json.Marshal(docs)
		} else {
			kycDocuments, _ = json.Marshal(map[string]string{})
		}

		restaurantID := GetIdToRecord("REST")
		now := time.Now()
		_, err = tx.Exec(ctx, `
			INSERT INTO restaurants (
				id, name, address, phone_number, email, is_active, kyc_status,
				bank_details, kyc_documents,
				created_on, last_updated_on, created_by, last_modified_by
			) VALUES ($1,$2,$3,$4,$5,false,'pending',$6,$7,$8,$8,$9,$9)`,
			restaurantID, str("restaurant_name"), str("address"), str("mobile_no"), str("email"),
			string(bankDetails), string(kycDocuments), now, userID,
		)
		if err != nil {
			return fmt.Errorf("failed to create restaurant: %w", err)
		}
		return r.queueApproval(ctx, tx, "restaurant", restaurantID)

	case "delivery_vendor":
		// A fleet/agency entity. Platform-approved like grocery/restaurant;
		// once active, its invite_code is how individual riders link
		// themselves to it (see the "delivery_partner" case below) instead
		// of every rider going straight into the platform's own queue.
		var existingID string
		err := tx.QueryRow(ctx, `SELECT id FROM delivery_vendor_profiles WHERE user_id = $1`, userID).Scan(&existingID)
		if err == nil {
			return nil
		}
		if err != pgx.ErrNoRows {
			return fmt.Errorf("failed to check existing delivery vendor profile: %w", err)
		}

		email := strings.TrimSpace(str("email"))
		phone := strings.TrimSpace(str("mobile_no"))
		pincode := strings.TrimSpace(str("pincode"))
		if !strings.Contains(email, "@") {
			return errors.New("a valid email address is required")
		}
		digits := 0
		for _, ch := range phone {
			if ch >= '0' && ch <= '9' {
				digits++
			}
		}
		if digits < 7 {
			return errors.New("a valid phone number is required")
		}
		if pincode == "" {
			return errors.New("pincode is required")
		}

		inviteCode, err := GenerateInviteCode(ctx, tx)
		if err != nil {
			return err
		}

		vendorID := GetIdToRecord("FLTV")
		now := time.Now()
		_, err = tx.Exec(ctx, `
			INSERT INTO delivery_vendor_profiles (
				id, user_id, company_name, owner_name, email, phone, address, city,
				pincode, gst_number, invite_code, kyc_status, is_active, rating,
				created_on, last_updated_on
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'pending',false,0.0,$12,$12)`,
			vendorID, userID, str("company_name"), str("first_name"), str("email"), str("mobile_no"),
			str("address"), str("city"), str("pincode"), str("gst_number"), inviteCode, now,
		)
		if err != nil {
			return fmt.Errorf("failed to create delivery vendor profile: %w", err)
		}
		return r.queueApproval(ctx, tx, "delivery_vendor", vendorID)

	case "delivery_partner":
		// Individual riders link to a fleet via invite code rather than
		// queuing for platform approval directly — the fleet reviews and
		// approves/rejects its own riders (DeliveryVendorRepository
		// ApproveRider/RejectRider), so no `approvals` row is queued here.
		var existingID string
		err := tx.QueryRow(ctx, `SELECT id FROM delivery_profiles WHERE user_id = $1`, userID).Scan(&existingID)
		if err == nil {
			return nil
		}
		if err != pgx.ErrNoRows {
			return fmt.Errorf("failed to check existing delivery profile: %w", err)
		}

		inviteCode := strings.TrimSpace(str("invite_code"))
		if inviteCode == "" {
			return ErrInviteCodeRequired
		}

		var vendorID, vendorActive string
		err = tx.QueryRow(ctx,
			`SELECT id, is_active::text FROM delivery_vendor_profiles WHERE invite_code = $1`, inviteCode,
		).Scan(&vendorID, &vendorActive)
		if err == pgx.ErrNoRows || vendorActive != "true" {
			return ErrInvalidInviteCode
		}
		if err != nil {
			return fmt.Errorf("failed to look up invite code: %w", err)
		}

		riderID := GetIdToRecord("DLVR")
		now := time.Now()
		_, err = tx.Exec(ctx, `
			INSERT INTO delivery_profiles (
				id, user_id, name, phone, city, vehicle_type, vehicle_number,
				license_number, kyc_status, rating, total_trips, is_online,
				vendor_id, is_active, created_on, last_updated_on
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending',0.0,0,false,$9,false,$10,$10)`,
			riderID, userID, str("first_name"), str("mobile_no"), str("city"), str("vehicle_type"),
			str("vehicle_number"), str("license_number"), vendorID, now,
		)
		if err != nil {
			return fmt.Errorf("failed to create delivery profile: %w", err)
		}
		return nil
	}

	return nil
}

// queueApproval inserts a pending row into `approvals` — previously nothing
// in the entire backend ever wrote to this table, so the admin approvals
// queue (GET /admin/approvals) always returned empty and Approve/Reject had
// no real target to act on.
func (r *RegistrationRepository) queueApproval(ctx context.Context, tx pgx.Tx, entityType, entityID string) error {
	approvalID := GetIdToRecord("APRV")
	now := time.Now()
	_, err := tx.Exec(ctx, `
		INSERT INTO approvals (id, entity_type, entity_id, status, submitted_at, created_on, last_updated_on)
		VALUES ($1, $2, $3, 'pending', $4, $4, $4)`,
		approvalID, entityType, entityID, now)
	if err != nil {
		return fmt.Errorf("failed to queue approval: %w", err)
	}
	return nil
}

// GetVendorRegistration returns a partner's onboarding submission (or nil rows).
func (r *RegistrationRepository) GetVendorRegistration(ctx context.Context, userID string) (map[string]interface{}, string, error) {
	var raw []byte
	var status string
	query := `SELECT payload, status FROM vendor_registrations WHERE user_id = $1`
	if err := r.db.QueryRow(ctx, query, userID).Scan(&raw, &status); err != nil {
		return nil, "", err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, "", err
	}
	return payload, status, nil
}

// GetRegistration returns the registration only if owned by userID — an
// empty userID never matches, so ownership is mandatory, not opt-in.
func (r *RegistrationRepository) GetRegistration(ctx context.Context, id, userID string) (map[string]interface{}, error) {
	var raw []byte
	query := `SELECT payload FROM restaurant_registrations WHERE id = $1 AND user_id = $2`
	if err := r.db.QueryRow(ctx, query, id, userID).Scan(&raw); err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// UpdateRegistration merges the given fields into the stored payload and
// returns the updated document. Ownership-scoped like GetRegistration.
func (r *RegistrationRepository) UpdateRegistration(ctx context.Context, id, userID string, updates map[string]interface{}) (map[string]interface{}, error) {
	payload, err := r.GetRegistration(ctx, id, userID)
	if err != nil {
		return nil, errors.New("registration not found")
	}

	for k, v := range updates {
		payload[k] = v
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	query := `UPDATE restaurant_registrations
	          SET payload = $1::jsonb, last_updated_on = $2
	          WHERE id = $3 AND user_id = $4`
	if _, err := r.db.Exec(ctx, query, string(raw), time.Now(), id, userID); err != nil {
		return nil, err
	}
	return payload, nil
}

// GetRegistrationByUser resolves the caller's own restaurant registration
// without needing to already know its id — the vendor app previously had no
// way to look this up at all (registerRestaurant's response was only ever
// kept in memory for the current session, never persisted), so every
// dashboard/orders/ratings screen fell back to a hardcoded 'mock_restaurant_id'.
func (r *RegistrationRepository) GetRegistrationByUser(ctx context.Context, userID string) (map[string]interface{}, error) {
	var raw []byte
	query := `SELECT payload FROM restaurant_registrations WHERE user_id = $1 ORDER BY created_on DESC LIMIT 1`
	if err := r.db.QueryRow(ctx, query, userID).Scan(&raw); err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// IsRegistrationOwner reports whether userID owns restaurantID — used to gate
// the outlet endpoints, which are keyed by restaurant_id rather than the
// registration's own id/user_id columns directly.
func (r *RegistrationRepository) IsRegistrationOwner(ctx context.Context, restaurantID, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM restaurant_registrations WHERE id = $1 AND user_id = $2)`
	if err := r.db.QueryRow(ctx, query, restaurantID, userID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// GetVendorOutlets returns outlets in the Partner app Outlet JSON shape.
func (r *RegistrationRepository) GetVendorOutlets(ctx context.Context, restaurantID string) ([]map[string]interface{}, error) {
	query := `SELECT id, outlet_type, menu, cuisines, cost_for_two,
	                 avg_delivery_time, address, packaging_charge,
	                 operating_hours, is_active
	          FROM vendor_outlets
	          WHERE restaurant_id = $1
	          ORDER BY created_on ASC`

	rows, err := r.db.Query(ctx, query, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	outlets := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, outletType, menu, avgDeliveryTime  string
			cuisines                               []string
			costForTwo                             int
			addressRaw, packagingRaw, hoursRaw     []byte
			isActive                               bool
		)
		if err := rows.Scan(
			&id, &outletType, &menu, &cuisines, &costForTwo,
			&avgDeliveryTime, &addressRaw, &packagingRaw, &hoursRaw, &isActive,
		); err != nil {
			return nil, err
		}

		outlet := map[string]interface{}{
			"id":                id,
			"outlet_type":       outletType,
			"menu":              menu,
			"cuisines":          cuisines,
			"cost_for_two":      costForTwo,
			"avg_delivery_time": avgDeliveryTime,
			"is_active":         isActive,
			"address":           json.RawMessage(addressRaw),
			"packaging_charge":  json.RawMessage(packagingRaw),
			"operating_hours":   json.RawMessage(hoursRaw),
		}
		outlets = append(outlets, outlet)
	}
	return outlets, rows.Err()
}
