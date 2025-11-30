package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"fmt"
	"strings"
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
		productVariant.VariantName,           // Variant variant_name (e.g., "Small", "Medium", "Large")
		productVariant.AdditionalDescription, // Additional additional_description for the variant (e.g., "Small size, serves 1")
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
func (r *ProductVariantRepository) GetProductVariantByID(ctx context.Context, id string) (models.ProductVariant, error) {
	var productVariant models.ProductVariant
	query := `SELECT id, product_id, variant_name, additional_description, price, quantity_available, created_on, last_updated_on, created_by, last_modified_by
	          FROM product_variants WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&productVariant.ID,
		&productVariant.ProductID,
		&productVariant.VariantName,
		&productVariant.AdditionalDescription,
		&productVariant.Price,
		&productVariant.QuantityAvailable,
		&productVariant.CreatedOn,
		&productVariant.LastUpdatedOn,
		&productVariant.CreatedBy,
		&productVariant.LastModifiedBy,
	)

	if err != nil {
		return productVariant, err
	}

	return productVariant, nil
}

func (r *ProductVariantRepository) GetProductVariants(ctx context.Context) ([]models.ProductVariant, error) {
	var productVariants []models.ProductVariant
	query := `SELECT id, product_id, variant_name, additional_description, price, quantity_available, created_on, last_updated_on, created_by, last_modified_by
	          FROM product_variants`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var productVariant models.ProductVariant
		if err := rows.Scan(
			&productVariant.ID,
			&productVariant.ProductID,
			&productVariant.VariantName,
			&productVariant.AdditionalDescription,
			&productVariant.Price,
			&productVariant.QuantityAvailable,
			&productVariant.CreatedOn,
			&productVariant.LastUpdatedOn,
			&productVariant.CreatedBy,
			&productVariant.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		productVariants = append(productVariants, productVariant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return productVariants, nil
}

func (r *ProductVariantRepository) UpdateProductVariant(ctx context.Context, productVariant models.ProductVariant) error {
	query := `UPDATE product_variants
		SET product_id = $1, variant_name = $2, additional_description = $3, price = $4, quantity_available = $5, last_updated_on = $6, last_modified_by = $7
		WHERE id = $8`

	_, err := r.db.Exec(ctx, query,
		productVariant.ProductID,
		productVariant.VariantName,
		productVariant.AdditionalDescription,
		productVariant.Price,
		productVariant.QuantityAvailable,
		time.Now(),
		globals.GetLoogedInUserId(),
		productVariant.ID,
	)

	return err
}

func (r *ProductVariantRepository) DeleteProductVariant(ctx context.Context, id string) error {
	query := `DELETE FROM product_variants WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *ProductVariantRepository) GetProductVariantsByProductId(ctx context.Context, productIds []string) (map[string][]models.ProductVariant, error) {
	var productVariantsToReturn map[string][]models.ProductVariant
	idsPGArray := fmt.Sprintf("{%s}", strings.Join(productIds, ","))
	query := `SELECT id, product_id, variant_name, additional_description, price, quantity_available
	          FROM product_variants WHERE product_id = ANY($1)`
	rows, err := r.db.Query(ctx, query, idsPGArray)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var productVariant models.ProductVariant
		if err := rows.Scan(
			&productVariant.ID,
			&productVariant.ProductID,
			&productVariant.VariantName,
			&productVariant.AdditionalDescription,
			&productVariant.Price,
			&productVariant.QuantityAvailable,
		); err != nil {
			return nil, err
		}
		if productVariantsToReturn == nil {
			productVariantsToReturn = make(map[string][]models.ProductVariant)
		}
		productVariantsToReturn[productVariant.ProductID] = append(productVariantsToReturn[productVariant.ProductID], productVariant)

	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return productVariantsToReturn, nil
}
