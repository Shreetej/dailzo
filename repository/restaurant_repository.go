package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"dailzo/utils"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RestaurantRepository struct {
	db      *pgxpool.Pool
	offRepo *OfferRepository
}

func NewRestaurantRepository(db *pgxpool.Pool) *RestaurantRepository {
	return &RestaurantRepository{db: db, offRepo: NewOfferRepository(db)}
}

// CreateRestaurant inserts a new restaurant record into the database
func (r *RestaurantRepository) CreateRestaurant(ctx context.Context, restaurant models.Restaurant) (string, error) {

	id := GetIdToRecord("REST")
	query := `INSERT INTO restaurants 
		(id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		id,
		restaurant.Name,
		restaurant.Address,
		restaurant.PhoneNumber,
		restaurant.Email,
		restaurant.OpeningTime,
		restaurant.ClosingTime,
		time.Now(),
		time.Now(),
		globals.GetLoogedInUserId(),
		globals.GetLoogedInUserId(),
	).Scan(&restaurant.ID)

	if err != nil {
		println("Error in query:", err.Error())
		return "", err
	}

	return id, nil
}
func (r *RestaurantRepository) GetRestaurantByID(ctx context.Context, id string) (models.Restaurant, error) {
	var restaurant models.Restaurant
	query := `SELECT id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by
	          FROM restaurants WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&restaurant.ID,
		&restaurant.Name,
		&restaurant.Address,
		&restaurant.PhoneNumber,
		&restaurant.Email,
		&restaurant.OpeningTime,
		&restaurant.ClosingTime,
		&restaurant.CreatedOn,
		&restaurant.LastUpdatedOn,
		&restaurant.CreatedBy,
		&restaurant.LastModifiedBy,
	)

	if err != nil {
		return restaurant, err
	}

	return restaurant, nil
}

func (r *RestaurantRepository) GetRestaurantsByIDs(ctx *fiber.Ctx, ids []string) ([]models.DisplayRestaurant, error) {
	var restaurants []models.DisplayRestaurant
	//var uLat, uLong = globals.GetSelectedAddLatLong()
	//fmt.Println("ids ", ids)
	idsPGArray := fmt.Sprintf("{%s}", strings.Join(ids, ","))
	fmt.Println("ids ", idsPGArray)

	query := `SELECT r.id, r.name, r.rating, r.address, a.longitude, a.latitude, r.phone_number, r.email, r.opening_time, r.closing_time 
			  FROM restaurants r JOIN addresses a ON r.address = a.id WHERE r.id = ANY($1)`

	rows, err := r.db.Query(ctx.Context(), query, idsPGArray)
	if err != nil {
		fmt.Println("err : 93", err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.DisplayRestaurant
		var restLat, restLong float64
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Rating,
			&restaurant.Address,
			&restLong,
			&restLat,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
		); err != nil {
			fmt.Println("err :  resto_repo_113", err.Error())
			return nil, err
		}
		restaurant.Distance = getDistance(restLat, restLong, ctx)
		restaurant.DeliveryTimings = fmt.Sprintf("%.2f Mins", (restaurant.Distance/10)*60)
		restaurant.IsFavorite = checkIfFev(restaurant.ID, ctx)
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("err : resto_repo_126", err.Error())
		return nil, err
	}
	//fmt.Println("restaurants ", restaurants)
	return restaurants, nil
}

func (r *RestaurantRepository) GetDisplayRestaurants(ctx *fiber.Ctx, ids []string) ([]models.DisplayRestaurantWithOffers, error) {
	var restaurants []models.DisplayRestaurantWithOffers
	//var uLat, uLong = globals.GetSelectedAddLatLong()
	//fmt.Println("ids ", ids)
	idsPGArray := fmt.Sprintf("{%s}", strings.Join(ids, ","))
	fmt.Println("ids ", idsPGArray)

	query := `SELECT r.id, r.name, r.rating, r.address, a.longitude, a.latitude, r.phone_number, r.email, r.opening_time, r.closing_time 
			  FROM restaurants r JOIN addresses a ON r.address = a.id WHERE r.id = ANY($1)`

	rows, err := r.db.Query(ctx.Context(), query, idsPGArray)
	if err != nil {
		fmt.Println("err : 93", err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.DisplayRestaurantWithOffers
		var restLat, restLong float64
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Rating,
			&restaurant.Address,
			&restLong,
			&restLat,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
		); err != nil {
			fmt.Println("err :  resto_repo_162", err.Error())
			return nil, err
		}
		restaurant.Distance = getDistance(restLat, restLong, ctx)
		restaurant.DeliveryTimings = fmt.Sprintf("%.2f Mins", (restaurant.Distance/10)*60)
		restaurant.IsFavorite = checkIfFev(restaurant.ID, ctx)
		offers, err := r.offRepo.GetOffersByRestaurantID(ctx.Context(), restaurant.ID)
		if err != nil {
			fmt.Printf("Error fetching offers for restaurant %s: %v\n", restaurant.ID, err)
			restaurant.Offers = []models.DisplayOffer{}
		} else if len(offers) == 0 {
			restaurant.Offers = []models.DisplayOffer{}
		} else {
			restaurant.Offers = offers
		}
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("err : resto_repo_126", err.Error())
		return nil, err
	}
	//fmt.Println("restaurants ", restaurants)
	return restaurants, nil
}

// // GetOffersByRestaurant retrieves all offers applicable to a specific restaurant
// func (r *RestaurantRepository) GetDisplayRestaurants(ctx *fiber.Ctx, restaurantIDs []string) ([]models.DisplayRestaurant, error) {
// 	query := `
// 		SELECT
// 			r.id AS restaurant_id,
// 			r.name AS restaurant_name,
// 			r.address AS restaurant_address,
// 			r.phone_number,
// 			r.email,
// 			r.opening_time,
// 			r.closing_time,
// 			a.longitude,
// 			a.latitude,
// 			r.rating,
// 			o.id AS offer_id,
// 			o.name AS offer_name,
// 			o.description AS offer_description,
// 			o.discount_percent,
// 			o.max_discount_amount,
// 			o.start_date,
// 			o.end_date,
// 			o.is_active,
// 			oc.id AS condition_id,
// 			oc.condition_type,
// 			oc.value AS condition_value,
// 			oa.entity_type,
// 			oa.entity_id
// 		FROM
// 			restaurants r
// 		JOIN
// 			addresses a ON r.address = a.id
// 		LEFT JOIN
// 			offer_applicable_entities oa ON r.id = oa.entity_id AND oa.entity_type = 'restaurant'
// 		LEFT JOIN
// 			offers o ON oa.offer_id = o.id
// 		LEFT JOIN
// 			offer_conditions oc ON o.id = oc.offer_id
// 		WHERE
// 			r.id = ANY($1)
// 		ORDER BY
// 			r.id, o.id, oc.id;`

// 	// Convert restaurant IDs to a format for SQL query
// 	idsPGArray := fmt.Sprintf("{%s}", strings.Join(restaurantIDs, ","))

// 	rows, err := r.db.Query(ctx.Context(), query, idsPGArray)

// 	if err != nil {
// 		return nil, fmt.Errorf("query execution failed: %w", err)
// 	}
// 	defer rows.Close()

// 	restaurantsMap := make(map[string]*models.DisplayRestaurant)

// 	for rows.Next() {
// 		var (
// 			restaurantID      string
// 			restaurantName    string
// 			address           string
// 			phoneNumber       string
// 			email             string
// 			openingTime       time.Time
// 			closingTime       time.Time
// 			rating            float64
// 			offerID           sql.NullString
// 			offerName         sql.NullString
// 			offerDescription  sql.NullString
// 			discountPercent   sql.NullFloat64
// 			maxDiscountAmount sql.NullFloat64
// 			startDate         sql.NullString
// 			endDate           sql.NullString
// 			isActive          sql.NullBool
// 			conditionID       sql.NullString
// 			conditionType     sql.NullString
// 			conditionValue    sql.NullString
// 			entityType        sql.NullString
// 			entityID          sql.NullString
// 			restLat           float64
// 			restLong          float64
// 		)

// 		err := rows.Scan(
// 			&restaurantID, &restaurantName, &address, &phoneNumber, &email,
// 			&openingTime, &closingTime, &restLong, &restLat, &rating,
// 			&offerID, &offerName, &offerDescription, &discountPercent,
// 			&maxDiscountAmount, &startDate, &endDate, &isActive,
// 			&conditionID, &conditionType, &conditionValue, &entityType, &entityID,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("row scan failed: %w", err)
// 		}

// 		// Check if the restaurant is already in the map
// 		restaurant, exists := restaurantsMap[restaurantID]
// 		if !exists {
// 			restaurant = &models.DisplayRestaurant{
// 				ID:              restaurantID,
// 				Name:            restaurantName,
// 				Address:         address,
// 				PhoneNumber:     phoneNumber,
// 				Email:           email,
// 				OpeningTime:     openingTime,
// 				ClosingTime:     closingTime,
// 				Distance:        getDistance(restLat, restLong, ctx),
// 				Rating:          rating,
// 				DeliveryTimings: fmt.Sprintf("%.2f Mins", (restaurant.Distance/10)*60),
// 				IsFavorite:      checkIfFev(restaurantID, ctx),
// 				Offer: models.DisplayOffer{
// 					OfferConditions:         []models.OfferCondition{},
// 					OfferApplicableEntities: []models.OfferApplicableEntity{},
// 				},
// 			}
// 			restaurantsMap[restaurantID] = restaurant
// 		}
// 		fmt.Println("restaurantsMap ", restaurantsMap)

// 		// Add offer details if present
// 		if offerID.Valid {
// 			restaurant.Offer = models.DisplayOffer{
// 				ID:                offerID.String,
// 				Name:              offerName.String,
// 				Description:       &offerDescription.String,
// 				DiscountPercent:   discountPercent.Float64,
// 				MaxDiscountAmount: maxDiscountAmount.Float64,
// 				StartDate:         startDate.String,
// 				EndDate:           endDate.String,
// 				IsActive:          isActive.Bool,
// 			}
// 		}

// 		// Add offer conditions if present
// 		if conditionID.Valid {
// 			restaurant.Offer.OfferConditions = append(restaurant.Offer.OfferConditions, models.OfferCondition{
// 				ID:            conditionID.String,
// 				ConditionType: conditionType.String,
// 				Value:         conditionValue.String,
// 			})
// 		}

// 		// Add applicable entities if present
// 		if entityType.Valid {
// 			restaurant.Offer.OfferApplicableEntities = append(restaurant.Offer.OfferApplicableEntities, models.OfferApplicableEntity{
// 				EntityType: entityType.String,
// 				EntityID:   entityID.String,
// 			})
// 		}
// 	}

// 	// Convert the map to a slice
// 	restaurants := make([]models.DisplayRestaurant, 0, len(restaurantsMap))
// 	for _, restaurant := range restaurantsMap {
// 		restaurants = append(restaurants, *restaurant)
// 	}

// 	return restaurants, nil
// }

func (r *RestaurantRepository) GetRestaurants(ctx *fiber.Ctx) ([]models.DisplayRestaurant, error) {
	var restaurants []models.DisplayRestaurant
	println(globals.GetLoogedInUserId())
	query := `SELECT r.id, r.name, r.rating, r.address, a.longitude, a.latitude, r.phone_number, r.email, r.opening_time, r.closing_time 
			  FROM restaurants r JOIN addresses a ON r.address = a.id`

	rows, err := r.db.Query(ctx.Context(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.DisplayRestaurant
		var restLat, restLong float64
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Rating,
			&restaurant.Address,
			&restLong,
			&restLat,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
		); err != nil {
			fmt.Println("restaurant : ", restaurant)
			fmt.Println("err : 167", err)
			return nil, err
		}
		restaurant.Distance = getDistance(restLat, restLong, ctx)
		restaurant.DeliveryTimings = fmt.Sprintf("%.2f Mins", (restaurant.Distance/10)*60)
		restaurant.IsFavorite = checkIfFev(restaurant.ID, ctx)
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {

		return nil, err
	}

	return restaurants, nil
}

func checkIfFev(resuarentId string, ctx *fiber.Ctx) bool {
	sess, err := globals.Store.Get(ctx)
	if err != nil {
		return false
	}
	favouriteRestaurants := sess.Get("favouriteRestaurants")
	fmt.Println("favouriteRestaurants ", favouriteRestaurants)
	if favouriteRestaurants == nil {
		fmt.Println("favouriteRestaurants is nil")
		return false
	}

	favouriteRestaurantsStr, ok := favouriteRestaurants.(string)
	if !ok {
		fmt.Println("favouriteRestaurants is not a string")
		return false
	}
	return strings.Contains(favouriteRestaurantsStr, resuarentId)
}

func getDistance(restLat, restLong float64, ctx *fiber.Ctx) float64 {
	sess, err := globals.Store.Get(ctx)
	if err != nil {
		return 0.0
	}
	addressLatitude := sess.Get("latitude")
	addressLongitude := sess.Get("longitude")
	if addressLatitude == nil || addressLongitude == nil {
		fmt.Println("addressLatitude is nil OR addressLongitude is nil")
		return 0.0
	}

	addressLatitudeFloat, ok := addressLatitude.(float64)
	if !ok {
		fmt.Println("addressLatitudefloat is not a float64")
		return 0.0
	}
	addressLongitudeFloat, ok := addressLongitude.(float64)
	if !ok {
		fmt.Println("addressLongitudeFloat is not a float64")
		return 0.0
	}
	return utils.GetDistance(restLat, restLong, addressLatitudeFloat, addressLongitudeFloat)
}

func (r *RestaurantRepository) GetRestaurantsByNearLocations(ctx context.Context, name string) ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	query := `SELECT id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by
	          FROM restaurants WHERE name ILIKE $1`

	rows, err := r.db.Query(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.Restaurant
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Address,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
			&restaurant.CreatedOn,
			&restaurant.LastUpdatedOn,
			&restaurant.CreatedBy,
			&restaurant.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return restaurants, nil
}

func (r *RestaurantRepository) GetRestaurantsByName(ctx context.Context, name string) ([]models.Restaurant, error) {
	var restaurants []models.Restaurant
	query := `SELECT id, name, address, phone_number, email, opening_time, closing_time, created_on, last_updated_on, created_by, last_modified_by
	          FROM restaurants WHERE name ILIKE $1`

	rows, err := r.db.Query(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var restaurant models.Restaurant
		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Address,
			&restaurant.PhoneNumber,
			&restaurant.Email,
			&restaurant.OpeningTime,
			&restaurant.ClosingTime,
			&restaurant.CreatedOn,
			&restaurant.LastUpdatedOn,
			&restaurant.CreatedBy,
			&restaurant.LastModifiedBy,
		); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return restaurants, nil
}

func (r *RestaurantRepository) UpdateRestaurant(ctx context.Context, restaurant models.Restaurant) error {
	query := `UPDATE restaurants
		SET name = $1, address = $2, phone_number = $3, email = $4, opening_time = $5, closing_time = $6, 
		last_updated_on = $7, last_modified_by = $8
		WHERE id = $9`

	_, err := r.db.Exec(ctx, query,
		restaurant.Name,
		restaurant.Address,
		restaurant.PhoneNumber,
		restaurant.Email,
		restaurant.OpeningTime,
		restaurant.ClosingTime,
		time.Now(),
		globals.GetLoogedInUserId(),
		restaurant.ID,
	)

	return err
}

func (r *RestaurantRepository) DeleteRestaurant(ctx context.Context, id string) error {
	query := `DELETE FROM restaurants WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
