package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductVariantRepository struct {
	db *pgxpool.Pool
}

func NewProductVariantRepository(db *pgxpool.Pool) *ProductVariantRepository {
	return &ProductVariantRepository{db: db}
}

func (r *ProductVariantRepository) CreateProductVariant(ctx context.Context, productVariant models.ProductVariant) (string, error) {

	id := GetIdToRecord("PRODV")
	query := `INSERT INTO product_variants 
				(id, product_id, variant_name, additional_description, price, quantity_available, is_active, created_on, last_updated_on, created_by, last_modified_by) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id`

	// Assuming 'db' is your database connection and 'ctx' is the context
	// Assuming 'db' is your database connection and 'ctx' is the context
	err := r.db.QueryRow(ctx, query,
		id,
		productVariant.ProductID,             // Product ID (link to the food_product table)
		productVariant.VariantName,           // Variant name (e.g., "Small", "Medium", "Large")
		productVariant.AdditionalDescription, // Additional description for the variant (e.g., "Small size, serves 1")
		productVariant.Price,                 // Price for the variant
		productVariant.QuantityAvailable,     // Quantity available for this variant
		productVariant.IsActive,              // Whether the variant is active
		time.Now(),                           // Timestamp for when the variant was created
		time.Now(),                           // Timestamp for the last update
		globals.GetLoogedInUserId(),          // User ID who created the variant
		globals.GetLoogedInUserId(),          // User ID who last modified the variant
	).Scan(&id)

	if err != nil {
		println("Error in query :", err.Error())
		return " ", err
	}

	return id, nil
}
