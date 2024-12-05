package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"fmt"
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
	println("User in query :", globals.GetLoogedInUserId())
	if err != nil {
		println("Error in query :", err.Error())
		return " ", err
	}

	return id, nil
}

func (r *FoodProductRepository) GetFoodProductByID(ctx context.Context, id string) (models.FoodProduct, error) {
	var foodProduct models.FoodProduct
	query := `SELECT id, name, description, price, category, created_by, last_modified_by 
	          FROM food_products WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&foodProduct.ID,
		&foodProduct.Name,
		&foodProduct.Description,
		&foodProduct.Price,
		&foodProduct.Category,
		&foodProduct.CreatedBy,
		&foodProduct.LastModifiedBy,
	)

	if err != nil {
		return foodProduct, err
	}

	return foodProduct, nil
}

func (r *FoodProductRepository) GetFoodProducts(ctx context.Context) ([]models.FoodProduct, error) {
	var foodProducts []models.FoodProduct
	query := `SELECT id, name, description, price, category, created_by, last_modified_by 
	          FROM food_products`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		println("err :", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var foodProduct models.FoodProduct
		if err := rows.Scan(
			&foodProduct.ID,
			&foodProduct.Name,
			&foodProduct.Description,
			&foodProduct.Price,
			&foodProduct.Category,
			&foodProduct.CreatedBy,
			&foodProduct.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		foodProducts = append(foodProducts, foodProduct)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return foodProducts, nil
}

func (r *FoodProductRepository) UpdateFoodProduct(ctx context.Context, foodProduct models.FoodProduct) error {
	query := `UPDATE food_products
		SET name = $1, description = $2,  price = $3, category = $4, last_updated_on = $5, last_modified_by = $6, type = $7,image_url = $8, is_active = $9
		WHERE id = $10`
	println("foodProduct.Type :", foodProduct.Type)

	result, err := r.db.Exec(ctx, query,
		foodProduct.Name,
		foodProduct.Description,
		foodProduct.Price,
		foodProduct.Category,
		time.Now(),
		globals.GetLoogedInUserId(),
		foodProduct.Type,
		foodProduct.ImageURL,
		foodProduct.IsActive,
		foodProduct.ID,
	)
	fmt.Println("foodProduct.Type :", foodProduct.ID)

	if err != nil {
		println("err :", err)
		return err
	}
	rowsAffected := result.RowsAffected()
	fmt.Printf("Rows affected: %d\n", rowsAffected)
	if rowsAffected == 0 {
		fmt.Println("No rows updated. Check the WHERE clause or input data.")
	}
	return err
}

func (r *FoodProductRepository) DeleteFoodProduct(ctx context.Context, id string) error {
	query := `DELETE FROM food_products WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
