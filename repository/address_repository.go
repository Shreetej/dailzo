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

	id := GetIdToRecord("ADDRS")
	query := `INSERT INTO address 
		(id, address_line_1, address_line_2, address_line_3, zip_pin, benchmark, user_id, city, state, type, longitude, latitude, created_on, last_updated_on, created_by, last_modified_by, mobileno, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id`

	// Assuming 'db' is your database connection and 'ctx' is the context
	err := r.db.QueryRow(ctx, query,
		id,
		address.AddressLine1,
		address.AddressLine2,
		address.AddressLine3,
		address.ZIPPin,
		address.Benchmark,
		globals.GetLoogedInUserId(),
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
		return " ", err
	}

	return id, nil
}
