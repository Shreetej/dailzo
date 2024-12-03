package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AddressRepository struct {
	db *pgxpool.Pool
}

func NewAddressRepository(db *pgxpool.Pool) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) CreateAddress(ctx context.Context, address models.Address) (string, error) {
	// Generate unique ID for the address
	id := GetIdToRecord("ADDRS")

	// Prepare the query
	query := `INSERT INTO addresses 
		(id, address_line_1, address_line_2, address_line_3, zip_pin, benchmark, user_id, city, state, type, longitude, latitude, created_on, last_updated_on, created_by, last_modified_by, mobileno, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id`

	// Execute the query
	err := r.db.QueryRow(ctx, query,
		id,
		address.AddressLine1,
		address.AddressLine2,
		address.AddressLine3,
		address.ZIPPin,
		address.Benchmark,
		globals.GetLoogedInUserId(), // Assuming this returns logged-in user ID
		address.City,
		address.State,
		address.Type,
		address.Longitude,
		address.Latitude,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
		address.MobileNo,
		address.Name).Scan(&address.ID)

	if err != nil {
		println("Error in query :", err.Error())
		return "", err
	}

	return id, nil
}

func (r *AddressRepository) GetAddresses(ctx context.Context) (models.Address, error) {
	var address models.Address
	query := `SELECT id, address_line_1, address_line_2, address_line_3, zip_pin, benchmark, user_id, city, state, type, longitude, latitude, created_on, last_updated_on, created_by, last_modified_by, mobileno, name 
	          FROM addresses`

	err := r.db.QueryRow(ctx, query).Scan(
		&address.ID,
		&address.AddressLine1,
		&address.AddressLine2,
		&address.AddressLine3,
		&address.ZIPPin,
		&address.Benchmark,
		&address.UserID,
		&address.City,
		&address.State,
		&address.Type,
		&address.Longitude,
		&address.Latitude,
		&address.CreatedOn,
		&address.LastUpdatedOn,
		&address.CreatedBy,
		&address.LastModifiedBy,
		&address.MobileNo,
		&address.Name,
	)

	if err != nil {
		return address, err
	}

	return address, nil
}

func (r *AddressRepository) GetAddressByID(ctx context.Context, id string) (models.Address, error) {
	var address models.Address
	query := `SELECT id, address_line_1, address_line_2, address_line_3, zip_pin, benchmark, user_id, city, state, type, longitude, latitude, created_on, last_updated_on, created_by, last_modified_by, mobileno, name 
	          FROM addresses WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&address.ID,
		&address.AddressLine1,
		&address.AddressLine2,
		&address.AddressLine3,
		&address.ZIPPin,
		&address.Benchmark,
		&address.UserID,
		&address.City,
		&address.State,
		&address.Type,
		&address.Longitude,
		&address.Latitude,
		&address.CreatedOn,
		&address.LastUpdatedOn,
		&address.CreatedBy,
		&address.LastModifiedBy,
		&address.MobileNo,
		&address.Name,
	)

	if err != nil {
		return address, err
	}

	return address, nil
}

func (r *AddressRepository) UpdateAddress(ctx context.Context, address models.Address) error {
	query := `UPDATE addresses 
		SET address_line_1 = $1, address_line_2 = $2, address_line_3 = $3, zip_pin = $4, benchmark = $5, user_id = $6, city = $7, state = $8, type = $9, longitude = $10, latitude = $11, 
		last_updated_on = $12, last_modified_by = $13, mobileno = $14, name = $15 
		WHERE id = $16`

	_, err := r.db.Exec(ctx, query,
		address.AddressLine1,
		address.AddressLine2,
		address.AddressLine3,
		address.ZIPPin,
		address.Benchmark,
		address.UserID,
		address.City,
		address.State,
		address.Type,
		address.Longitude,
		address.Latitude,
		time.Now(),
		globals.GetLoogedInUserId(),
		address.MobileNo,
		address.Name,
		address.ID,
	)

	return err
}

func (r *AddressRepository) DeleteAddress(ctx context.Context, id string) error {
	query := `DELETE FROM addresses WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
