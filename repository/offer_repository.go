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

type OfferRepository struct {
	db *pgxpool.Pool
}

func NewOfferRepository(db *pgxpool.Pool) *OfferRepository {
	return &OfferRepository{db: db}
}

func (r *OfferRepository) CreateOffer(ctx context.Context, offer models.Offer) (string, error) {
	id := GetIdToRecord("OFFER")
	query := `INSERT INTO offers 
		(id, name, description, discount_percent, max_discount_amount, start_date, end_date, is_active, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		offer.Name,
		offer.Description,
		offer.DiscountPercent,
		offer.MaxDiscountAmount,
		offer.StartDate,
		offer.EndDate,
		offer.IsActive,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&offer.ID)

	if err != nil {
		return "", err
	}

	return id, nil
}

func (r *OfferRepository) GetOffers(ctx context.Context) ([]models.Offer, error) {
	fmt.Print("in offers")
	var offers []models.Offer
	query := `SELECT id, name, description, discount_percent, max_discount_amount, start_date, end_date, is_active, created_on, last_updated_on 
	          FROM offers`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, errors.New("no offers found")
	}
	defer rows.Close()
	for rows.Next() {
		var offer models.Offer
		if err = rows.Scan(
			&offer.ID,
			&offer.Name,
			&offer.Description,
			&offer.DiscountPercent,
			&offer.MaxDiscountAmount,
			&offer.StartDate,
			&offer.EndDate,
			&offer.IsActive,
			&offer.CreatedOn,
			&offer.LastUpdatedOn,
		); err != nil {
			fmt.Print(err.Error())

			return nil, err
		}
		offers = append(offers, offer)
	}

	//offers, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Offer])

	if err != nil {
		fmt.Println("In outer try: get Offers", err.Error())

		return nil, err
	}

	return offers, nil
}

func (r *OfferRepository) UpdateOffer(ctx context.Context, offer models.Offer) error {
	query := `UPDATE offers 
		SET name = $1, description = $2, discount_percent = $3, max_discount_amount = $4, start_date = $5, end_date = $6, is_active = $7, last_updated_on = $8, last_modified_by = $9 
		WHERE id = $10`

	_, err := r.db.Exec(ctx, query,
		offer.Name,
		offer.Description,
		offer.DiscountPercent,
		offer.MaxDiscountAmount,
		offer.StartDate,
		offer.EndDate,
		offer.IsActive,
		time.Now(),
		globals.GetLoogedInUserId(),
		offer.ID,
	)

	return err
}

func (r *OfferRepository) DeleteOffer(ctx context.Context, id string) error {
	query := `DELETE FROM offers WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *OfferRepository) CreateCondition(ctx context.Context, condition models.OfferCondition) (string, error) {
	id := GetIdToRecord("COND")
	query := `INSERT INTO offer_conditions 
		(id, offer_id, condition_type, value, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		condition.OfferID,
		condition.ConditionType,
		condition.Value,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&condition.ID)

	if err != nil {
		fmt.Println("In outer try: conditions", err.Error())

		return "", err
	}

	return id, nil
}

// func (r *OfferRepository) GetConditionsByOfferID(ctx context.Context, offerID string) ([]models.OfferCondition, error) {
// 	conditions := []models.OfferCondition{}
// 	query := `SELECT id, offer_id, condition_type, value, created_on, last_updated_on
// 	          FROM offer_conditions WHERE offer_id = $1`

// 	rows, err := r.db.Query(ctx, query, offerID)
// 	if err != nil {
// 		fmt.Println("In outer try: get conditions", err.Error())
// 		return nil, errors.New("no offers found")
// 	}

// 	defer rows.Close()

// 	//conditions, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.OfferCondition])
// 	for rows.Next() {
// 		var offer_condition models.OfferCondition
// 		if err = rows.Scan(
// 			&offer_condition.ID,
// 			&offer_condition.OfferID,
// 			&offer_condition.ConditionType,
// 			&offer_condition.Value,
// 			&offer_condition.CreatedOn,
// 			&offer_condition.LastUpdatedOn,
// 		); err != nil {
// 			fmt.Print(err.Error())

// 			return nil, err
// 		}
// 		conditions = append(conditions, offer_condition)
// 	if err != nil {
// 		fmt.Println("In outer try: get conditions", err.Error())

// 		return nil, err
// 	}

// 	return conditions, nil
// }

func (r *OfferRepository) CreateApplicableEntity(ctx context.Context, entity models.OfferApplicableEntity) (string, error) {
	id := GetIdToRecord("APP_ENT")
	query := `INSERT INTO offer_applicable_entities 
		(id, offer_id, entity_type, entity_id, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		entity.OfferID,
		entity.EntityType,
		entity.EntityID,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&entity.ID)

	if err != nil {
		return "", err
	}

	return id, nil
}

func (r *OfferRepository) GetEntitiesByOfferID(ctx context.Context, offerID string) ([]models.OfferApplicableEntity, error) {
	query := `SELECT id, offer_id, entity_type, entity_id, created_on, last_updated_on 
	          FROM offer_applicable_entities WHERE offer_id = $1`

	rows, err := r.db.Query(ctx, query, offerID)
	if err == pgx.ErrNoRows {
		return nil, errors.New("no entities found for the offer")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.OfferApplicableEntity])
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *OfferRepository) GetConditionsByOfferID(ctx context.Context, offerID string) ([]models.OfferCondition, error) {
	query := `SELECT id, offer_id, condition_type, value, created_on, last_updated_on 
			  FROM offer_conditions WHERE offer_id = $1`

	rows, err := r.db.Query(ctx, query, offerID)
	if err == pgx.ErrNoRows {
		return nil, errors.New("no conditions found for the offer")
	}
	defer rows.Close()

	conditions, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.OfferCondition])
	if err != nil {
		return nil, err
	}

	return conditions, nil
}

func (r *OfferRepository) GetOffersByRestaurantID(ctx context.Context, restaurantID string) ([]models.DisplayOffer, error) {
	query := `SELECT o.id, o.name, o.description, o.discount_percent, o.max_discount_amount, o.start_date, o.end_date, o.is_active, o.created_on, o.last_updated_on 
			  FROM offers o
			  JOIN offer_applicable_entities ae ON o.id = ae.offer_id
			  WHERE ae.entity_type = 'restaurant' AND ae.entity_id = $1`

	rows, err := r.db.Query(ctx, query, restaurantID)
	if err == pgx.ErrNoRows {
		return nil, errors.New("no offers found for the given restaurant IDs")
	}
	defer rows.Close()

	offers, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.DisplayOffer])
	if err != nil {
		return nil, err
	}

	for i, offer := range offers {
		conditions, err := r.GetConditionsByOfferID(ctx, offer.ID)
		if err != nil {
			return nil, err
		}
		offers[i].OfferConditions = conditions
		entities, err := r.GetEntitiesByOfferID(ctx, offer.ID)
		if err != nil {
			return nil, err
		}
		offers[i].OfferApplicableEntities = entities
	}

	return offers, nil
}
