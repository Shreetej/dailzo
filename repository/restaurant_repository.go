package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RestaurantRepository struct {
	db *pgxpool.Pool
}

func NewRestaurantRepository(db *pgxpool.Pool) *RestaurantRepository {
	return &RestaurantRepository{db: db}
}

// CreateRestaurant inserts a new restaurant record into the database
func (r *RestaurantRepository) CreateRestaurant(ctx context.Context, restaurant models.Restaurant) (string, error) {

	id := GetIdToRecord("REST")
	query := `INSERT INTO restaurants 
		(id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		restaurant.Name,
		restaurant.Address,
		restaurant.PhoneNumber,
		restaurant.Email,
		restaurant.OpeningTime,
		restaurant.ClosingTime,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&restaurant.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}
