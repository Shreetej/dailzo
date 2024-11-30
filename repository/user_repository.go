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
	query := `INSERT INTO users (id, name, email, mobileno, password) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err = r.db.QueryRow(ctx, query, id, user.Name, user.Email, user.MobileNo, string(hashedPassword)).Scan(&id)
	if err != nil {
		println("Error in query :", err.Error())
		return " ", err
	}
	return id, nil
}

func (r *UserRepository) GetUsers(ctx context.Context) ([]models.DisplayUser, error) {
	// var user models.User
	query := `SELECT id, name, email, mobileno, created_at, updated_at FROM users`
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

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	var user models.User
	query := `SELECT id, name, email, mobileno, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email, &user.MobileNo, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return &user, err
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `SELECT id, name, email, mobileno, password, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Name, &user.Email, &user.MobileNo, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return &user, err
}

// UpdateUser updates a user's information
func (r *UserRepository) UpdateUser(ctx context.Context, user models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		print(err.Error())
		return err
	}
	query := `
		UPDATE users
		SET name = $1, email = $2, mobileno = $3, password = $4, updated_at = $5
		WHERE id = $6
	`
	_, err = r.db.Exec(ctx, query, user.Name, user.Email, user.MobileNo, string(hashedPassword), time.Now(), user.ID)
	return err
}

// DeleteUser deletes a user by ID
func (r *UserRepository) DeleteUser(ctx context.Context, userID int) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
