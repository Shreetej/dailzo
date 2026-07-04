package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RegistrationRepository stores the Partner app's restaurant registration
// documents as JSONB and lists vendor outlets in the shape the app expects.
type RegistrationRepository struct {
	db *pgxpool.Pool
}

func NewRegistrationRepository(db *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

func (r *RegistrationRepository) CreateRegistration(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	id, _ := payload["id"].(string)
	if id == "" {
		id = GetIdToRecord("REST")
		payload["id"] = id
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO restaurant_registrations (id, payload, created_on, last_updated_on)
	          VALUES ($1, $2::jsonb, $3, $3)
	          ON CONFLICT (id) DO UPDATE
	          SET payload = EXCLUDED.payload, last_updated_on = EXCLUDED.last_updated_on`

	if _, err := r.db.Exec(ctx, query, id, string(raw), time.Now()); err != nil {
		return nil, err
	}
	return payload, nil
}

func (r *RegistrationRepository) GetRegistration(ctx context.Context, id string) (map[string]interface{}, error) {
	var raw []byte
	query := `SELECT payload FROM restaurant_registrations WHERE id = $1`
	if err := r.db.QueryRow(ctx, query, id).Scan(&raw); err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// UpdateRegistration merges the given fields into the stored payload and
// returns the updated document.
func (r *RegistrationRepository) UpdateRegistration(ctx context.Context, id string, updates map[string]interface{}) (map[string]interface{}, error) {
	payload, err := r.GetRegistration(ctx, id)
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
	          WHERE id = $3`
	if _, err := r.db.Exec(ctx, query, string(raw), time.Now(), id); err != nil {
		return nil, err
	}
	return payload, nil
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
