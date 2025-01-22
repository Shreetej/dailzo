package repository

import (
	"context"
	"dailzo/models"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user models.User) (string, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		print(err.Error())
		return " ", err
	}

	//get users id
	id := GetIdToRecord("USR")
	UserName := *user.FirstName + *user.LastName
	fmt.Println("CTX :", ctx)
	query := `INSERT INTO users (id, username, first_name, middle_name, last_name, email, mobileno, password) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err = r.db.QueryRow(ctx, query, id, UserName, user.FirstName, user.MiddleName, user.LastName, user.Email, user.MobileNo, string(hashedPassword)).Scan(&id)
	if err != nil {
		println("Error in query :", err.Error())
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
	query := `SELECT id, username, email, mobileno FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.MobileNo)
	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return &user, err
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, email, mobileno, password, favourite_restaurants, fevourite_foods FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.MobileNo, &user.Password, &user.FavouriteRestaurants, &user.FavouriteFoods)
	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Println("User not found for email:", email)
			return nil, errors.New("user not found")
		}
		fmt.Printf("Error executing query: %v\n", err)
		return nil, err
	}
	// if err == pgx.ErrNoRows {
	// 	return nil, errors.New("user not found")
	// }
	return &user, err
}

// UpdateUser updates a user's information
func (r *UserRepository) UpdateUser(ctx context.Context, user models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		print(err.Error())
		return err
	}
	fmt.Printf("Error executing query: %v\n", ctx)
	UserName := *user.FirstName + *user.LastName
	query := `
		UPDATE users
		SET first_name = $1, email = $2, mobileno = $3, password = $4, last_updated_on = $5, last_name = $6, middle_name = $7, username= $8
		WHERE email = $2
	`
	_, err = r.db.Exec(ctx, query, user.FirstName, user.Email, user.MobileNo, string(hashedPassword), time.Now(), user.LastName, user.MiddleName, UserName)
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
