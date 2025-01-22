package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"dailzo/utils"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
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
func (r *RestaurantRepository) GetRestaurantByID(ctx context.Context, id string) (models.Restaurant, error) {
	var restaurant models.Restaurant
	query := `SELECT id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by
	          FROM restaurants WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&restaurant.ID,
		&restaurant.Name,
		&restaurant.Address,
		&restaurant.PhoneNumber,
		&restaurant.Email,
		&restaurant.OpeningTime,
		&restaurant.ClosingTime,
		&restaurant.CreatedOn,
		&restaurant.LastUpdatedOn,
		&restaurant.CreatedBy,
		&restaurant.LastModifiedBy,
	)

	if err != nil {
		return restaurant, err
	}

	return restaurant, nil
}

func (r *RestaurantRepository) GetRestaurants(ctx *fiber.Ctx) ([]models.DisplayRestaurant, error) {
	var restaurants []models.DisplayRestaurant
	var uLat, uLong = globals.GetSelectedAddLatLong()
	println(globals.GetLoogedInUserId())
	//username := sess.Get("username")
	query := `SELECT r.id, r.name, r.rating, r.address, a.longitude, a.latitude, r.phone_number, r.email, r.opening_time, r.closing_time, r.created_on, r.last_updated_on, r.created_by, r.last_modified_by
			  FROM restaurants r JOIN addresses a ON r.address = a.id`

	rows, err := r.db.Query(ctx.Context(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.DisplayRestaurant
		var restLat, restLong float64
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Rating,
			&restaurant.Address,
			&restLong,
			&restLat,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
			&restaurant.CreatedOn,
			&restaurant.LastUpdatedOn,
			&restaurant.CreatedBy,
			&restaurant.LastModifiedBy,
		); err != nil {
			fmt.Println("restaurant : ", restaurant)
			fmt.Println("err : ", err)
			return nil, err
		}
		restaurant.Distance = utils.GetDistance(restLat, restLong, uLat, uLong)
		//restaurant.DeliveryTimings = "30"
		restaurant.IsFavorite = checkIfFev(restaurant.ID, ctx)
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {

		return nil, err
	}

	return restaurants, nil
}

func checkIfFev(resuarentId string, ctx *fiber.Ctx) bool {
	sess, err := globals.Store.Get(ctx)
	if err != nil {
		return false
	}
	favouriteRestaurants := sess.Get("favouriteRestaurants")

	if favouriteRestaurants == nil {
		fmt.Println("favouriteRestaurants is nil")
		return false
	}

	favouriteRestaurantsStr, ok := favouriteRestaurants.(string)
	if !ok {
		fmt.Println("favouriteRestaurants is not a string")
		return false
	}
	return strings.Contains(favouriteRestaurantsStr, resuarentId)
}
func (r *RestaurantRepository) GetRestaurantsByNearLocations(ctx context.Context, name string) ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	query := `SELECT id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by
	          FROM restaurants WHERE name ILIKE $1`

	rows, err := r.db.Query(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.Restaurant
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Address,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
			&restaurant.CreatedOn,
			&restaurant.LastUpdatedOn,
			&restaurant.CreatedBy,
			&restaurant.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return restaurants, nil
}

func (r *RestaurantRepository) GetRestaurantsByName(ctx context.Context, name string) ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	query := `SELECT id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by
	          FROM restaurants WHERE name ILIKE $1`

	rows, err := r.db.Query(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.Restaurant
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Address,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
			&restaurant.CreatedOn,
			&restaurant.LastUpdatedOn,
			&restaurant.CreatedBy,
			&restaurant.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return restaurants, nil
}

func (r *RestaurantRepository) UpdateRestaurant(ctx context.Context, restaurant models.Restaurant) error {
	query := `UPDATE restaurants
		SET name = $1, address = $2, phone_number = $3, email = $4, opening_time = $5, closing_time = $6, 
		last_updated_on = $7, last_modified_by = $8
		WHERE id = $9`

	_, err := r.db.Exec(ctx, query,
		restaurant.Name,
		restaurant.Address,
		restaurant.PhoneNumber,
		restaurant.Email,
		restaurant.OpeningTime,
		restaurant.ClosingTime,
		time.Now(),
		globals.GetLoogedInUserId(),
		restaurant.ID,
	)

	return err
}

func (r *RestaurantRepository) DeleteRestaurant(ctx context.Context, id string) error {
	query := `DELETE FROM restaurants WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
