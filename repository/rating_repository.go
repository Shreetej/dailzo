package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RatingRepository struct {
	db *pgxpool.Pool
}

func NewRatingRepository(db *pgxpool.Pool) *RatingRepository {
	return &RatingRepository{db: db}
}

// CreateRating inserts a new rating record into the database
func (r *RatingRepository) CreateRating(ctx context.Context, rating models.Rating) (string, error) {

	id := GetIdToRecord("RTNG")
	query := `INSERT INTO ratings 
		(id, rating, comment, user_id, entity_type, entity_id, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		rating.Rating,
		rating.Comment,
		rating.UserID,
		rating.EntityType,
		rating.EntityID,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&rating.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}
func (r *RatingRepository) GetRatingByID(ctx context.Context, id string) (models.Rating, error) {
	var rating models.Rating
	query := `SELECT id, user_id,  entity_id, rating, comment, created_on, last_updated_on, created_by, last_modified_by
	          FROM ratings WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&rating.ID,
		&rating.UserID,
		&rating.EntityID,
		&rating.Rating,
		&rating.Comment,
		&rating.CreatedOn,
		&rating.LastUpdatedOn,
		&rating.CreatedBy,
		&rating.LastModifiedBy,
	)

	if err != nil {
		return rating, err
	}

	return rating, nil
}

func (r *RatingRepository) GetRatings(ctx context.Context) ([]models.Rating, error) {
	var ratings []models.Rating
	query := `SELECT id, user_id, entity_id, rating, comment, created_on, last_updated_on, created_by, last_modified_by
	          FROM ratings`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var rating models.Rating
		if err := rows.Scan(
			&rating.ID,
			&rating.UserID,
			&rating.EntityID,
			&rating.Rating,
			&rating.Comment,
			&rating.CreatedOn,
			&rating.LastUpdatedOn,
			&rating.CreatedBy,
			&rating.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		ratings = append(ratings, rating)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ratings, nil
}

func (r *RatingRepository) UpdateRating(ctx context.Context, rating models.Rating) error {
	query := `UPDATE ratings
		SET user_id = $1,  entity_id = $2, order_id = $3, rating = $4, comment = $5, last_updated_on = $6, last_modified_by = $7
		WHERE id = $8`

	_, err := r.db.Exec(ctx, query,
		rating.UserID,
		rating.EntityID,
		rating.Rating,
		rating.Comment,
		rating.LastUpdatedOn,
		rating.LastModifiedBy,
		rating.ID,
	)

	return err
}

func (r *RatingRepository) DeleteRating(ctx context.Context, id string) error {
	query := `DELETE FROM ratings WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
