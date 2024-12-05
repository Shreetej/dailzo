package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateOrder inserts a new order into the database
func (r *OrderRepository) CreateOrder(ctx context.Context, order models.Order) (string, error) {

	id := GetIdToRecord("ORDR") // Assuming you have a function to generate IDs
	query := `INSERT INTO orders 
		(id, user_id, restaurant_id, status, total_amount, order_date, delivery_person_id, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	// Execute the query and return the generated id
	err := r.db.QueryRow(ctx, query,
		id,
		order.UserID,
		order.RestaurantID,
		order.Status,
		order.TotalAmount,
		time.Now(),
		order.DeliveryPersonID,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(), // Assuming a function for getting logged-in user ID
		globals.GetLoogedInUserId(),
	).Scan(&order.ID)
	println("order.Status, :", order.Status)

	if err != nil {
		println("Error in query:", err)
		println("Error in query:", err.Error())
		return "", err
	}
	println("Error in query:", id)

	return id, nil
}

// GetOrderByID retrieves an order by its ID
func (r *OrderRepository) GetOrderByID(ctx context.Context, id string) (*models.Order, error) {
	query := `SELECT id, user_id, restaurant_id, status, total_amount, order_date, delivery_person_id, created_on, last_updated_on, created_by, last_modified_by
		FROM orders
		WHERE id = $1`

	var order models.Order
	err := r.db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.RestaurantID,
		&order.Status,
		&order.TotalAmount,
		&order.OrderDate,
		&order.DeliveryPersonID,
		&order.CreatedOn,
		&order.LastUpdatedOn,
		&order.CreatedBy,
		&order.LastModifiedBy,
	)

	if err != nil {
		println("Error in query:", err.Error())
		return nil, err
	}

	return &order, nil
}

// GetOrderByID retrieves an order by its ID
func (r *OrderRepository) GetOrders(ctx context.Context) ([]models.Order, error) {
	query := `SELECT id, user_id, restaurant_id, status, total_amount, order_date, delivery_person_id, created_on, last_updated_on, created_by, last_modified_by
		FROM orders`

	rows, err := r.db.Query(ctx, query)
	var orders []models.Order

	if err == pgx.ErrNoRows {
		return nil, errors.New("no orders found")
	}
	defer rows.Close()
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.RestaurantID,
			&order.Status,
			&order.TotalAmount,
			&order.OrderDate,
			&order.DeliveryPersonID,
			&order.CreatedOn,
			&order.LastUpdatedOn,
			&order.CreatedBy,
			&order.LastModifiedBy,
		); err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	if err != nil {
		println("Error in query:", err.Error())
		return nil, err
	}

	return orders, nil
}

// UpdateOrder updates an existing order in the database
// func (r *OrderRepository) UpdateOrder(ctx context.Context, order models.Order) error {
// 	query := `UPDATE orders
// 		SET user_id = $1, restaurant_id = $2, status = $3, total_amount = $4,  delivery_person_id = $5, last_updated_on = $6, last_modified_by = $7
// 		WHERE id = $8`

// 	_, err := r.db.Exec(ctx, query,
// 		order.UserID,
// 		order.RestaurantID,
// 		order.Status,
// 		order.TotalAmount,
// 		order.DeliveryPersonID,
// 		time.Now(),
// 		globals.GetLoogedInUserId(),
// 		order.ID,
// 	)
// 	println("Error in query:", err.Error())
// 	if err != nil {
// 		println("Error in query:", err.Error())
// 		return err
// 	}

// 	return nil
// }

func (r *OrderRepository) UpdateOrder(ctx context.Context, order models.Order) error {
	// Validate that the required fields are provided
	if order.ID == "" {
		return fmt.Errorf("order ID cannot be empty")
	}
	if order.Status == "" {
		return fmt.Errorf("order status cannot be empty")
	}

	// Define the SQL update query
	query := `UPDATE orders 
		SET user_id = $1, restaurant_id = $2, status = $3, total_amount = $4, 
			delivery_person_id = $5, last_updated_on = $6, last_modified_by = $7
		WHERE id = $8`

	// Execute the update query
	result, err := r.db.Exec(ctx, query,
		order.UserID,
		order.RestaurantID,
		order.Status,
		order.TotalAmount,
		order.DeliveryPersonID,
		time.Now(),
		globals.GetLoogedInUserId(),
		order.ID,
	)

	// Check for errors in the query execution
	if err != nil {
		// Log the error for debugging
		println("Error executing query:", err.Error())
		return fmt.Errorf("failed to update order: %w", err)
	}

	// Check if any rows were affected by the update
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		// If no rows were affected, it means the order with the provided ID was not found
		return fmt.Errorf("no order found with ID %s", order.ID)
	}

	// Successful update
	return nil
}

// DeleteOrder deletes an order by its ID
func (r *OrderRepository) DeleteOrder(ctx context.Context, id string) error {
	query := `DELETE FROM orders WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)

	if err != nil {
		println("Error in query:", err.Error())
		return err
	}

	return nil
}
