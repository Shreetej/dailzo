package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderItemRepository struct {
	db *pgxpool.Pool
}

func NewOrderItemRepository(db *pgxpool.Pool) *OrderItemRepository {
	return &OrderItemRepository{db: db}
}

// CreateOrderItem inserts a new order item into the database
func (r *OrderItemRepository) CreateOrderItem(ctx context.Context, orderItem models.OrderItem) (string, error) {

	id := GetIdToRecord("ORDITM") // Assuming you have a function to generate IDs
	query := `INSERT INTO order_items 
		(id, order_id, product_variant_id, quantity, price, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	// Execute the query and return the generated id
	err := r.db.QueryRow(ctx, query,
		id,
		orderItem.OrderID,
		orderItem.ProductVariantID,
		orderItem.Quantity,
		orderItem.Price,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(), // Assuming a function for getting logged-in user ID
		globals.GetLoogedInUserId(),
	).Scan(&orderItem.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}

// GetOrderItemByID retrieves an order item by its ID
func (r *OrderItemRepository) GetOrderItems(ctx context.Context) ([]models.OrderItem, error) {
	query := `SELECT id, order_id, product_variant_id, quantity, price, created_on, last_updated_on, created_by, last_modified_by
		FROM order_items`
	var orderItems []models.OrderItem

	rows, err := r.db.Query(ctx, query)
	if err == pgx.ErrNoRows {
		return nil, errors.New("no address found")
	}
	defer rows.Close()
	for rows.Next() {
		var orderItem models.OrderItem
		if err = rows.Scan(
			&orderItem.ID,
			&orderItem.OrderID,
			&orderItem.ProductVariantID,
			&orderItem.Quantity,
			&orderItem.Price,
			&orderItem.CreatedOn,
			&orderItem.LastUpdatedOn,
			&orderItem.CreatedBy,
			&orderItem.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		orderItems = append(orderItems, orderItem)
	}

	if err != nil {
		println("Error in query:", err.Error())
		return nil, err
	}

	return orderItems, nil
}

func (r *OrderItemRepository) GetOrderItemByID(ctx context.Context, id string) (*models.OrderItem, error) {
	query := `SELECT id, order_id, product_variant_id, quantity, price, created_on, last_updated_on, created_by, last_modified_by
		FROM order_items
		WHERE id = $1`

	var orderItem models.OrderItem
	err := r.db.QueryRow(ctx, query, id).Scan(
		&orderItem.ID,
		&orderItem.OrderID,
		&orderItem.ProductVariantID,
		&orderItem.Quantity,
		&orderItem.Price,
		&orderItem.CreatedOn,
		&orderItem.LastUpdatedOn,
		&orderItem.CreatedBy,
		&orderItem.LastModifiedBy,
	)

	if err != nil {
		println("Error in query:", err.Error())
		return nil, err
	}

	return &orderItem, nil
}

// UpdateOrderItem updates an existing order item in the database
func (r *OrderItemRepository) UpdateOrderItem(ctx context.Context, orderItem models.OrderItem) error {
	query := `UPDATE order_items 
		SET order_id = $1, product_variant_id = $2, quantity = $3, price = $4, last_updated_on = $5, last_modified_by = $6
		WHERE id = $7`

	_, err := r.db.Exec(ctx, query,
		orderItem.OrderID,
		orderItem.ProductVariantID,
		orderItem.Quantity,
		orderItem.Price,
		time.Now(),
		globals.GetLoogedInUserId(),
		orderItem.ID,
	)

	if err != nil {
		println("Error in query:", err.Error())
		return err
	}

	return nil
}

// DeleteOrderItem deletes an order item by its ID
func (r *OrderItemRepository) DeleteOrderItem(ctx context.Context, id string) error {
	query := `DELETE FROM order_items WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)

	if err != nil {
		println("Error in query:", err.Error())
		return err
	}

	return nil
}
