package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FoodProductRepository struct {
	db *pgxpool.Pool
	rp *RestaurantRepository
}

func NewFoodProductRepository(db *pgxpool.Pool) *FoodProductRepository {
	return &FoodProductRepository{db: db, rp: NewRestaurantRepository(db)}
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

func (r *FoodProductRepository) GetFoodProductWithEntity(ctx *fiber.Ctx, entity string) ([]models.DisplayFoodCatagoryProducts, error) {
	var foodProductsToReturn []models.DisplayFoodCatagoryProducts
	restIdsSet := make(map[string]struct{}) // Use a set for unique restaurant IDs
	mapResIdToFoodProd := make(map[string][]models.DisplayFoodCatagoryProducts)
	mapCatToFoodProdToReturn := make(map[string]models.DisplayFoodCatagoryProducts)

	// Query for food products
	query := `SELECT name, description, category, restaurant, is_active
	          FROM food_products WHERE is_active = true AND (category ILIKE $1 OR name ILIKE $1)`
	rows, err := r.db.Query(ctx.Context(), query, entity)
	if err != nil {
		fmt.Println("Query Error:", err.Error())
		return nil, err
	}
	defer rows.Close()

	// Process food products
	for rows.Next() {
		var foodProduct models.DisplayFoodCatagoryProducts
		if err := rows.Scan(
			&foodProduct.Name,
			&foodProduct.Description,
			&foodProduct.Category,
			&foodProduct.RestaurantId,
			&foodProduct.IsActive,
		); err != nil {
			fmt.Println("Row Scan Error:", err)
			return nil, err
		}

		restaurantID := strings.TrimSpace(foodProduct.RestaurantId)
		mapResIdToFoodProd[restaurantID] = append(mapResIdToFoodProd[restaurantID], foodProduct)
		restIdsSet[restaurantID] = struct{}{} // Add to set for unique restaurant IDs
	}

	// Convert set to slice for querying restaurants
	var restIds []string
	for id := range restIdsSet {
		restIds = append(restIds, id)
	}

	// Fetch restaurants
	restaurants, err := r.rp.GetDisplayRestaurants(ctx, restIds)
	if err != nil {
		fmt.Println("Error Fetching Restaurants:", err)
		return nil, err
	}

	// Map restaurants to food products
	for _, restaurant := range restaurants {
		restaurantID := strings.TrimSpace(restaurant.ID)
		if foodProducts, ok := mapResIdToFoodProd[restaurantID]; ok {
			for _, foodProduct := range foodProducts {
				if existingProduct, exists := mapCatToFoodProdToReturn[foodProduct.Category]; exists {
					// Append restaurant to existing product
					existingProduct.Restaurants = append(existingProduct.Restaurants, restaurant)
					mapCatToFoodProdToReturn[foodProduct.Category] = existingProduct
				} else {
					// Create new category product with restaurant
					foodProduct.Restaurants = []models.DisplayRestaurantWithOffers{restaurant}
					mapCatToFoodProdToReturn[foodProduct.Category] = foodProduct
				}
			}
		}
	}

	// Convert map to slice
	for _, value := range mapCatToFoodProdToReturn {
		foodProductsToReturn = append(foodProductsToReturn, value)
	}

	return foodProductsToReturn, nil
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

// GetProductsWithFilters retrieves products with various filters
func (r *FoodProductRepository) GetProductsWithFilters(ctx context.Context, category, stock *string, promo *bool, expiry, role *string) ([]models.FoodProduct, error) {
	query := `SELECT id, name, description, price, category, image_url, is_active,
		stock_quantity, low_stock_threshold, expiry_date, is_promo, auto_discount_pct,
		outlet_id, created_on, last_updated_on
		FROM food_products WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	// Category filter
	if category != nil && *category != "" {
		argCount++
		query += fmt.Sprintf(" AND category ILIKE $%d", argCount)
		args = append(args, "%"+*category+"%")
	}

	// Stock filter
	if stock != nil && *stock != "" {
		switch *stock {
		case "low":
			query += " AND stock_quantity <= low_stock_threshold"
		case "out":
			query += " AND stock_quantity = 0"
		case "available":
			query += " AND stock_quantity > 0"
		}
	}

	// Promo filter
	if promo != nil && *promo {
		query += " AND is_promo = true"
	}

	// Expiry filter
	if expiry != nil && *expiry != "" {
		switch *expiry {
		case "soon":
			query += " AND expiry_date IS NOT NULL AND expiry_date <= CURRENT_DATE + INTERVAL '7 days' AND expiry_date >= CURRENT_DATE"
		case "expired":
			query += " AND expiry_date IS NOT NULL AND expiry_date < CURRENT_DATE"
		}
	}

	// Role filter (grocery vs restaurant)
	if role != nil && *role != "" {
		// This would filter based on the outlet type
		// For now, we assume outlet_id helps distinguish
	}

	query += " AND is_active = true ORDER BY name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []models.FoodProduct
	for rows.Next() {
		var p models.FoodProduct
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.Category,
			&p.ImageURL, &p.IsActive, &p.StockQuantity, &p.LowStockThreshold,
			&p.ExpiryDate, &p.IsPromo, &p.AutoDiscountPct, &p.OutletID,
			&p.CreatedOn, &p.LastUpdatedOn,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

// GetExpiryAlerts gets products expiring soon
func (r *FoodProductRepository) GetExpiryAlerts(ctx context.Context, outletID string) ([]models.FoodProduct, error) {
	query := `SELECT id, name, description, price, category, stock_quantity, expiry_date, outlet_id
		FROM food_products
		WHERE is_active = true
			AND expiry_date IS NOT NULL
			AND expiry_date <= CURRENT_DATE + INTERVAL '7 days'
			AND expiry_date >= CURRENT_DATE`

	args := []interface{}{}
	if outletID != "" {
		query += " AND outlet_id = $1"
		args = append(args, outletID)
	}

	query += " ORDER BY expiry_date ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query expiry alerts: %w", err)
	}
	defer rows.Close()

	var products []models.FoodProduct
	for rows.Next() {
		var p models.FoodProduct
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.Category,
			&p.StockQuantity, &p.ExpiryDate, &p.OutletID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

// GetAutoDiscounts gets products with auto-discount enabled
func (r *FoodProductRepository) GetAutoDiscounts(ctx context.Context, outletID string) ([]models.FoodProduct, error) {
	query := `SELECT id, name, description, price, category, stock_quantity,
		expiry_date, auto_discount_pct, outlet_id
		FROM food_products
		WHERE is_active = true
			AND auto_discount_enabled = true
			AND expiry_date IS NOT NULL
			AND expiry_date <= CURRENT_DATE + INTERVAL '3 days'`

	args := []interface{}{}
	if outletID != "" {
		query += " AND outlet_id = $1"
		args = append(args, outletID)
	}

	query += " ORDER BY expiry_date ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query auto discounts: %w", err)
	}
	defer rows.Close()

	var products []models.FoodProduct
	for rows.Next() {
		var p models.FoodProduct
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.Category,
			&p.StockQuantity, &p.ExpiryDate, &p.AutoDiscountPct, &p.OutletID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

// PatchProduct updates specific fields of a product
func (r *FoodProductRepository) PatchProduct(ctx context.Context, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	query := "UPDATE food_products SET "
	args := []interface{}{}
	argCount := 0

	for key, value := range updates {
		if argCount > 0 {
			query += ", "
		}
		argCount++
		query += fmt.Sprintf("%s = $%d", key, argCount)
		args = append(args, value)
	}

	// Always update last_updated_on
	argCount++
	query += fmt.Sprintf(", last_updated_on = $%d", argCount)
	args = append(args, time.Now())

	argCount++
	query += fmt.Sprintf(", last_modified_by = $%d", argCount)
	args = append(args, globals.GetLoogedInUserId())

	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}
