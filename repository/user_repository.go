package repository

import (
	"context"
	"crypto/rand"
	"dailzo/models"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// ErrEmailTaken is returned when a signup collides with an existing account
// (unique violation on users.email), so callers can return a clean 409.
var ErrEmailTaken = errors.New("email already registered")

func (r *UserRepository) CreateUser(ctx context.Context, user models.User) (string, error) {
	passwordToHash := user.Password
	if passwordToHash == "" {
		// Phone/OTP-created accounts (see PostAuthSendOtp) have no password.
		// Hash a random value instead of "" so there is no guessable credential
		// that would authenticate this row via the email/password login path.
		randomSeed := make([]byte, 32)
		if _, err := rand.Read(randomSeed); err != nil {
			return "", err
		}
		passwordToHash = base64.RawStdEncoding.EncodeToString(randomSeed)
	}
	// Hash the password with optimized cost factor for better performance
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordToHash), 12) // bcrypt cost 12 (secure)
	if err != nil {
		return " ", err
	}

	//get users id
	id := GetIdToRecord("USR")
	// username has a UNIQUE constraint. Derive a human-ish base from the name
	// (or the email local-part) and append a slice of the unique id so two
	// vendors with the same display name never collide.
	base := user.Email
	if at := strings.IndexByte(base, '@'); at > 0 {
		base = base[:at]
	}
	if user.FirstName != nil && *user.FirstName != "" && user.LastName != nil {
		base = *user.FirstName + *user.LastName
	}
	suffix := id
	if len(id) > 6 {
		suffix = id[len(id)-6:]
	}
	UserName := base + "-" + suffix

	// email has a UNIQUE constraint that (unlike NOT NULL) still applies to an
	// empty string, so two phone-only signups with Email == "" would collide
	// on the second one. Pass real SQL NULL instead — Postgres never treats
	// NULL as equal to NULL, so any number of phone-only accounts can coexist,
	// and a later WHERE email = '' lookup (e.g. from a malformed login
	// request) can never match one of them.
	var emailParam interface{}
	if user.Email != "" {
		emailParam = user.Email
	}

	query := `INSERT INTO users (id, username, first_name, middle_name, last_name, email, mobileno, password, user_type, address_id, profile_image_url, bio, date_of_birth, gender, created_on, last_updated_on, created_by, last_modified_by, favourite_restaurants, favourite_foods) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20) RETURNING id`
	err = r.db.QueryRow(ctx, query,
		id,
		UserName,
		user.FirstName,
		user.MiddleName,
		user.LastName,
		emailParam,
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Unique violation (email or username already exists).
			return "", ErrEmailTaken
		}
		fmt.Printf("Error executing query: %v\n", err)
		return " ", err
	}
	return id, nil
}

func (r *UserRepository) GetUsers(ctx context.Context) ([]models.DisplayUser, error) {
	// var user models.User
	query := `SELECT id, username, COALESCE(email, '') AS email, mobileno, created_on, last_updated_on FROM users`
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
	query := `SELECT id, username, COALESCE(email, '') AS email, mobileno, first_name, middle_name, last_name, password, user_type, address_id, profile_image_url, bio, date_of_birth, gender, created_on, last_updated_on, created_by, last_modified_by, favourite_restaurants, favourite_foods FROM users WHERE id = $1`
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
	query := `SELECT id, username, COALESCE(email, '') AS email, mobileno, first_name, last_name, password, user_type FROM users WHERE email = $1 LIMIT 1`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.MobileNo,
		&user.FirstName,
		&user.LastName,
		&user.Password,
		&user.UserType,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByMobile looks up a user by phone number for the OTP login/signup
// flow. If more than one row shares a number (pre-existing dev data predating
// the mobile-OTP flow), the earliest-created account wins deterministically.
func (r *UserRepository) GetUserByMobile(ctx context.Context, mobile string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, COALESCE(email, '') AS email, mobileno, first_name, last_name, user_type
	          FROM users WHERE mobileno = $1 ORDER BY created_on ASC LIMIT 1`
	err := r.db.QueryRow(ctx, query, mobile).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.MobileNo,
		&user.FirstName,
		&user.LastName,
		&user.UserType,
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
	query := `SELECT id, username, COALESCE(email, '') AS email, mobileno, first_name, middle_name, last_name, password, user_type, address_id, profile_image_url, bio, date_of_birth, gender, created_on, last_updated_on, created_by, last_modified_by, favourite_restaurants, favourite_foods FROM users WHERE email = $1`
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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12) // bcrypt cost 12 (secure)
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

// UpdateProfileFields updates only the caller-editable profile fields (name,
// mobile, date of birth, gender) for PATCH /auth/profile. Unlike UpdateUser
// (which unconditionally re-bcrypts whatever is in user.Password — correct
// for the legacy PUT /users "replace with a new plaintext password" contract,
// but silently double-hashes and corrupts the existing hash when the caller
// only meant to edit a profile field), this never touches the password
// column at all.
func (r *UserRepository) UpdateProfileFields(ctx context.Context, user models.User) error {
	query := `
		UPDATE users
		SET first_name = $1, middle_name = $2, last_name = $3, mobileno = $4, date_of_birth = $5, gender = $6, last_updated_on = $7
		WHERE id = $8
	`
	_, err := r.db.Exec(ctx, query,
		user.FirstName,
		user.MiddleName,
		user.LastName,
		user.MobileNo,
		user.DateOfBirth,
		user.Gender,
		time.Now(),
		user.ID,
	)
	return err
}

// SetFCMToken records the caller's current device push token, overwriting
// whatever was there (a user only ever has one active device token at a time
// in this simple model — multi-device would need its own table).
func (r *UserRepository) SetFCMToken(ctx context.Context, userID, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET fcm_token = $1, last_updated_on = $2 WHERE id = $3`,
		token, time.Now(), userID)
	return err
}

// DeleteUser deletes a user by ID
func (r *UserRepository) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *UserRepository) UpdateFavoriteRestaurant(ctx *fiber.Ctx, newFavoriteRestaurant string) error {
	userID := LoggedInUserID(ctx.Context())
	if userID == "" {
		return errors.New("unauthorized")
	}
	var currentFavorites sql.NullString
	// Retrieve the current favoriteRestaurant value
	query := `SELECT favourite_restaurants FROM users WHERE id = $1`

	err := r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("user not found")
		}
		return err
	}
	// Append the new favorite restaurant to the existing value, unless it's
	// already there (repeat taps shouldn't duplicate the entry).
	existing := strings.Split(currentFavorites.String, ",")
	for _, id := range existing {
		if strings.TrimSpace(id) == newFavoriteRestaurant {
			return nil
		}
	}
	if currentFavorites.Valid && currentFavorites.String != "" {
		currentFavorites.String = currentFavorites.String + "," + newFavoriteRestaurant
	} else {
		currentFavorites.String = newFavoriteRestaurant
	}

	// Update the favoriteRestaurant field with the new value
	updateQuery := `UPDATE users SET favourite_restaurants = $1 WHERE id = $2`
	_, err = r.db.Exec(ctx.Context(), updateQuery, currentFavorites.String, userID)
	return err
}

func (r *UserRepository) UpdateFavoriteFoods(ctx *fiber.Ctx, newFavoriteFood string) error {
	userID := LoggedInUserID(ctx.Context())
	if userID == "" {
		return errors.New("unauthorized")
	}
	var currentFavorites sql.NullString
	// Retrieve the current favoriteFoods value
	query := `SELECT favourite_foods FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("user not found")
		}
		return err
	}

	existing := strings.Split(currentFavorites.String, ",")
	for _, id := range existing {
		if strings.TrimSpace(id) == newFavoriteFood {
			return nil
		}
	}
	if currentFavorites.Valid && currentFavorites.String != "" {
		currentFavorites.String = currentFavorites.String + "," + newFavoriteFood
	} else {
		currentFavorites.String = newFavoriteFood
	}

	// Update the favoriteFoods field with the new value
	updateQuery := `UPDATE users SET favourite_foods = $1 WHERE id = $2`
	_, err = r.db.Exec(ctx.Context(), updateQuery, currentFavorites.String, userID)
	return err
}

func (r *UserRepository) RemoveFavoriteFood(ctx *fiber.Ctx, foodToRemove string) error {
	userID := LoggedInUserID(ctx.Context())
	if userID == "" {
		return errors.New("unauthorized")
	}
	var currentFavorites string
	// Retrieve the current favoriteFoods value
	query := `SELECT favourite_foods FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("user not found")
		}
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
	return err
}

func (r *UserRepository) RemoveFavoriteRestaurant(ctx *fiber.Ctx, restaurantToRemove string) error {
	userID := LoggedInUserID(ctx.Context())
	if userID == "" {
		return errors.New("unauthorized")
	}
	var currentFavorites string
	// Retrieve the current favoriteRestaurants value
	query := `SELECT favourite_restaurants FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx.Context(), query, userID).Scan(&currentFavorites)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("user not found")
		}
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
	return err
}
