package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FoodProductRepository struct {
	db *pgxpool.Pool
}

func NewFoodProductRepository(db *pgxpool.Pool) *FoodProductRepository {
	return &FoodProductRepository{db: db}
}

func (r *FoodProductRepository) CreateFoodProduct(ctx context.Context, foodProduct models.FoodProduct) (string, error) {

	id := GetIdToRecord("FPROD")
	query := `INSERT INTO food_products 
    (id, name, description, category, type, price, image_url, is_active, created_on, last_updated_on, created_by, last_modified_by) 
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
    RETURNING id`

	// Assuming 'db' is your database connection and 'ctx' is the context
	err := r.db.QueryRow(ctx, query,
		id,
		foodProduct.Name,
		foodProduct.Description,
		foodProduct.Category,
		foodProduct.Type,
		foodProduct.Price,
		foodProduct.ImageURL,
		foodProduct.IsActive,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&foodProduct.ID)

	if err != nil {
		println("Error in query :", err.Error())
		return " ", err
	}

	return id, nil
}
