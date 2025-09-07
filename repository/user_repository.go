package repository

import (
	"context"
	"dailzo/globals"
	"dailzo/models"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	db   *pgxpool.Pool
	addr *AddressRepository
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db, addr: NewAddressRepository(db)}
}

func (r *UserRepository) CreateUser(ctx context.Context, user models.User) (string, error) {
	// Hash the password with optimized cost factor for better performance
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 6) // Optimized for performance
	if err != nil {
		return " ", err
	}

	//get users id
	id := GetIdToRecord("USR")
	UserName := ""
	if user.FirstName != nil && user.LastName != nil {
		UserName = *user.FirstName + *user.LastName
	}
	query := `INSERT INTO users (id, username, first_name, middle_name, last_name, email, mobileno, password, user_type, address_id, profile_image_url, bio, date_of_birth, gender, created_on, last_updated_on, created_by, last_modified_by, favourite_restaurants, favourite_foods) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20) RETURNING id`
	err = r.db.QueryRow(ctx, query,
		id,
		UserName,
		user.FirstName,
		user.MiddleName,
		user.LastName,
		user.Email,
		user.MobileNo,
		string(hashedPassword),
		user.UserType,
		user.AddressID,
		user.ProfileImageURL,
		user.Bio,
		user.DateOfBirth,
		user.Gender,
		time.Now(),
		time.Now(),
		user.CreatedBy,
		user.LastModifiedBy,
		user.FavouriteRestaurants,
		user.FavouriteFoods,
	).Scan(&id)
	if err != nil {
		return " ", err
	}
	return id, nil
}

func (r *UserRepository) GetUsers(ctx context.Context) ([]models.DisplayUser, error) {
	// var user models.User
	query := `SELECT id, username, email, mobileno, created_on, last_updated_on FROM users`
	rows, err := r.db.Query(ctx, query)
	if err == pgx.ErrNoRows {
		return nil, errors.New("no users found")
	}
	defer rows.Close()
	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.DisplayUser])
	fmt.Println("Users : ", users)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, mobileno, first_name, middle_name, last_name, password, user_type, address_id, profile_image_url, bio, date_of_birth, gender, created_on, last_updated_on, created_by, last_modified_by, favourite_restaurants, favourite_foods FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.MobileNo,
		&user.FirstName,
		&user.MiddleName,
		&user.LastName,
		&user.Password,
		&user.UserType,
		&user.AddressID,
		&user.ProfileImageURL,
		&user.Bio,
		&user.DateOfBirth,
		&user.Gender,
		&user.CreatedOn,
		&user.LastUpdatedOn,
		&user.CreatedBy,
		&user.LastModifiedBy,
		&user.FavouriteRestaurants,
		&user.FavouriteFoods,
	)
	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return &user, err
}

// GetUserByEmailForLogin - Optimized for login, fetches only essential fields
func (r *UserRepository) GetUserByEmailForLogin(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	// Only select fields needed for login and response
	query := `SELECT id, username, email, mobileno, first_name, last_name, password FROM users WHERE email = $1 LIMIT 1`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.MobileNo,
		&user.FirstName,
		&user.LastName,
		&user.Password,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByEmail(ctx *fiber.Ctx, email string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, mobileno, first_name, middle_name, last_name, password, user_type, address_id, profile_image_url, bio, date_of_birth, gender, created_on, last_updated_on, created_by, last_modified_by, favourite_restaurants, favourite_foods FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx.Context(), query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.MobileNo,
		&user.FirstName,
		&user.MiddleName,
		&user.LastName,
		&user.Password,
		&user.UserType,
		&user.AddressID,
		&user.ProfileImageURL,
		&user.Bio,
		&user.DateOfBirth,
		&user.Gender,
		&user.CreatedOn,
		&user.LastUpdatedOn,
		&user.CreatedBy,
		&user.LastModifiedBy,
		&user.FavouriteRestaurants,
		&user.FavouriteFoods,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user's information
func (r *UserRepository) UpdateUser(ctx context.Context, user models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 6) // Optimized for performance
	if err != nil {
		return err
	}
	fmt.Printf("Error executing query: %v\n", ctx)
	UserName := ""
	if user.FirstName != nil && user.LastName != nil {
		UserName = *user.FirstName + *user.LastName
	}
	query := `
		UPDATE users
		SET first_name = $1, middle_name = $2, last_name = $3, email = $4, mobileno = $5, password = $6, user_type = $7, address_id = $8, profile_image_url = $9, bio = $10, date_of_birth = $11, gender = $12, last_updated_on = $13, last_modified_by = $14, username = $15, favourite_restaurants = $16, favourite_foods = $17
		WHERE id = $18
	`
	_, err = r.db.Exec(ctx, query,
		user.FirstName,
		user.MiddleName,
		user.LastName,
		user.Email,
		user.MobileNo,
		string(hashedPassword),
		user.UserType,
		user.AddressID,
		user.ProfileImageURL,
		user.Bio,
		user.DateOfBirth,
		user.Gender,
		time.Now(),
		user.LastModifiedBy,
		UserName,
		user.FavouriteRestaurants,
		user.FavouriteFoods,
		user.ID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("user not found")
		}
		fmt.Printf("Error executing query: %v\n", err)
		return err
	}
	return err
}

// DeleteUser deletes a user by ID
func (r *UserRepository) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *UserRepository) UpdateFavoriteRestaurant(ctx *fiber.Ctx, newFavoriteRestaurant string) error {
	var currentFavorites sql.NullString
	sess, err := globals.Store.Get(ctx)
	if err != nil {
		return err
	}
	userID := sess.Get("userID")
	fmt.Println(userID)
	// Retrieve the current favoriteRestaurant value
	query := `SELECT favourite_restaurants FROM users WHERE id = $1`

	err = r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Println("User not found for email:")
			return errors.New("user not found")
		}
		fmt.Printf("Error executing query: %v\n", err)
		return err
	}
	// Append the new favorite restaurant to the existing value
	if currentFavorites.Valid && currentFavorites.String != "" {
		currentFavorites.String = currentFavorites.String + "," + newFavoriteRestaurant
	} else {
		currentFavorites.String = newFavoriteRestaurant
	}

	// Update the favoriteRestaurant field with the new value
	updateQuery := `UPDATE users SET favourite_restaurants = $1 WHERE id = $2`
	_, err = r.db.Exec(ctx.Context(), updateQuery, currentFavorites.String, userID)
	if err != nil {
		return err
	}
	sess.Set("favouriteRestaurants", currentFavorites.String)
	if err := sess.Save(); err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) UpdateFavoriteFoods(ctx *fiber.Ctx, newFavoriteFood string) error {
	var currentFavorites sql.NullString
	sess, err := globals.Store.Get(ctx)
	if err != nil {
		return err
	}
	userID := sess.Get("userID")
	// Retrieve the current favoriteFoods value
	query := `SELECT favourite_foods FROM users WHERE id = $1`
	err = r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Println("User not found for email:")
			return errors.New("user not found")
		}
		fmt.Printf("Error executing query: %v\n", err)
		return err
	}

	// Append the new favorite food to the existing value
	if currentFavorites.Valid && currentFavorites.String != "" {
		currentFavorites.String = currentFavorites.String + "," + newFavoriteFood
	} else {
		currentFavorites.String = newFavoriteFood
	}

	// Update the favoriteFoods field with the new value
	updateQuery := `UPDATE users SET favourite_foods = $1 WHERE id = $2`
	_, err = r.db.Exec(ctx.Context(), updateQuery, currentFavorites, userID)
	if err != nil {
		return err
	}
	sess.Set("favouriteFoods", currentFavorites.String)
	if err := sess.Save(); err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) RemoveFavoriteFood(ctx *fiber.Ctx, foodToRemove string) error {
	var currentFavorites string
	sess, err := globals.Store.Get(ctx)
	if err != nil {
		return err
	}
	userID := sess.Get("userID")

	// Retrieve the current favoriteFoods value
	query := `SELECT favourite_foods FROM users WHERE id = $1`
	err = r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Println("User not found for ID:", userID)
			return errors.New("user not found")
		}
		fmt.Printf("Error executing query: %v\n", err)
		return err
	}

	// Remove the specified food from the list
	favorites := strings.Split(currentFavorites, ",")
	for i, favorite := range favorites {
		if strings.TrimSpace(favorite) == foodToRemove {
			favorites = append(favorites[:i], favorites[i+1:]...)
			break
		}
	}
	newFavorites := strings.Join(favorites, ",")

	// Update the favoriteFoods field with the new value
	updateQuery := `UPDATE users SET favourite_foods = $1 WHERE id = $2`
	_, err = r.db.Exec(ctx.Context(), updateQuery, newFavorites, userID)
	if err != nil {
		return err
	}

	sess.Set("favouriteFoods", newFavorites)
	if err := sess.Save(); err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) RemoveFavoriteRestaurant(ctx *fiber.Ctx, restaurantToRemove string) error {
	var currentFavorites string
	sess, err := globals.Store.Get(ctx)
	if err != nil {
		return err
	}
	userID := sess.Get("userID")

	// Retrieve the current favoriteRestaurants value
	query := `SELECT favourite_restaurants FROM users WHERE id = $1`
	err = r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Println("User not found for ID:", userID)
			return errors.New("user not found")
		}
		fmt.Printf("Error executing query: %v\n", err)
		return err
	}

	// Remove the specified restaurant from the list
	favorites := strings.Split(currentFavorites, ",")
	for i, favorite := range favorites {
		if strings.TrimSpace(favorite) == restaurantToRemove {
			favorites = append(favorites[:i], favorites[i+1:]...)
			break
		}
	}
	newFavorites := strings.Join(favorites, ",")

	// Update the favoriteRestaurants field with the new value
	updateQuery := `UPDATE users SET favourite_restaurants = $1 WHERE id = $2`
	_, err = r.db.Exec(ctx.Context(), updateQuery, newFavorites, userID)
	if err != nil {
		return err
	}

	sess.Set("favouriteRestaurants", currentFavorites)
	if err := sess.Save(); err != nil {
		return err
	}

	return nil
}
