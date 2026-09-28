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
		LoggedInUserID(ctx), // Assuming this returns logged-in user ID
		address.City,
		address.State,
		address.Type,
		address.Longitude,
		address.Latitude,
		time.Now(),
		time.Now(),
		LoggedInUserID(ctx),
		LoggedInUserID(ctx),
		address.MobileNo,
		address.Name).Scan(&address.ID)

	if err != nil {
		println("Error in query :", err.Error())
		return "", err
	}

	return id, nil
}

// GetAddresses returns only the caller's own addresses. Previously this had
// no WHERE clause at all — any authenticated user could list every address
// (including phone numbers and lat/long) belonging to every other user.
func (r *AddressRepository) GetAddresses(ctx context.Context) ([]models.DisplayAddress, error) {
	query := `SELECT id, address_line_1, address_line_2, address_line_3, zip_pin, city, state, type,
	                 longitude, latitude, created_on, last_updated_on, mobileno, name
	          FROM addresses WHERE user_id = $1`
	rows, err := r.db.Query(ctx, query, LoggedInUserID(ctx))

	if err == pgx.ErrNoRows {
		return nil, errors.New("no address found")
	}
	defer rows.Close()
	addresses, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.DisplayAddress])
	if err != nil {
		return nil, err
	}
	return addresses, nil
}

// GetAddressByID is scoped to the caller — previously any authenticated user
// could fetch any other user's address by guessing/enumerating its id.
func (r *AddressRepository) GetAddressByID(ctx context.Context, id string) (models.Address, error) {
	var address models.Address
	query := `SELECT id, address_line_1, address_line_2, address_line_3, zip_pin, benchmark, user_id, city, state, type, longitude, latitude, mobileno, name
	          FROM addresses WHERE id = $1 AND user_id = $2`

	err := r.db.QueryRow(ctx, query, id, LoggedInUserID(ctx)).Scan(
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
		&address.MobileNo,
		&address.Name,
	)

	if err != nil {
		return address, err
	}

	return address, nil
}

func (r *AddressRepository) GetAddressByUserID(ctx context.Context, usrId string) (models.Address, error) {
	var address models.Address
	query := `SELECT id, address_line_1, address_line_2, address_line_3, zip_pin, benchmark, user_id, city, state, type, longitude, latitude, mobileno, name 
	          FROM addresses WHERE user_id = $1 LIMIT 1`

	err := r.db.QueryRow(ctx, query, usrId).Scan(
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
		&address.MobileNo,
		&address.Name,
	)

	if err != nil {
		fmt.Println("Addresses : ", address)
		fmt.Println("err : 106", err)
		return address, err
	}

	return address, nil
}

// UpdateAddress is scoped to the caller (WHERE id = ... AND user_id = ...) —
// previously any authenticated user could update any address by id, and the
// query even let the caller reassign an address's user_id via the request
// body, i.e. hijack it onto another account. user_id is no longer settable
// from the update payload at all.
func (r *AddressRepository) UpdateAddress(ctx context.Context, address models.Address) error {
	query := `UPDATE addresses
		SET address_line_1 = $1, address_line_2 = $2, address_line_3 = $3, zip_pin = $4, benchmark = $5, city = $6, state = $7, type = $8, longitude = $9, latitude = $10,
		last_updated_on = $11, last_modified_by = $12, mobileno = $13, name = $14
		WHERE id = $15 AND user_id = $16`

	tag, err := r.db.Exec(ctx, query,
		address.AddressLine1,
		address.AddressLine2,
		address.AddressLine3,
		address.ZIPPin,
		address.Benchmark,
		address.City,
		address.State,
		address.Type,
		address.Longitude,
		address.Latitude,
		time.Now(),
		LoggedInUserID(ctx),
		address.MobileNo,
		address.Name,
		address.ID,
		LoggedInUserID(ctx),
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("address not found")
	}
	return nil
}

// DeleteAddress is scoped to the caller — previously any authenticated user
// could delete any other user's address by id.
func (r *AddressRepository) DeleteAddress(ctx context.Context, id string) error {
	query := `DELETE FROM addresses WHERE id = $1 AND user_id = $2`
	tag, err := r.db.Exec(ctx, query, id, LoggedInUserID(ctx))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("address not found")
	}
	return nil
}
