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
