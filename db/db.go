package db

import (
	"context"
	"dailzo/config"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

// ConnectDatabase initializes a connection to the PostgreSQL database using pgx v5.
func ConnectDatabase(cfg config.Config) {
	// Step 1: Connect to the default 'postgres' database
	dsnPostgres := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)

	configPostgres, err := pgxpool.ParseConfig(dsnPostgres)
	if err != nil {
		log.Fatalf("Unable to parse default database configuration: %v\n", err)
	}

	poolPostgres, err := pgxpool.NewWithConfig(context.Background(), configPostgres)
	if err != nil {
		log.Fatalf("Unable to connect to default database: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 2: Create the target database if it does not exist
	createDBSQL := fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)
	_, err = poolPostgres.Exec(ctx, createDBSQL)
	if err != nil && !isDuplicateDatabaseError(err) {
		log.Fatalf("Unable to create database: %v\n", err)
	}
	poolPostgres.Close()

	// Step 3: Connect to the target database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse database configuration: %v\n", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	if err = DB.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping the database: %v\n", err)
	}

	log.Println("Connected to the database successfully!")
	// Load the .sql file
	sqlFile := "DatabaseScripts&ERD/DailzoScript.sql"
	sqlBytes, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Fatalf("Failed to read SQL file: %v\n", err)
	}
	sqlContent := string(sqlBytes)

	// Execute the SQL commands to create tables if not exist
	_, err = DB.Exec(ctx, sqlContent)
	if err != nil {
		log.Fatalf("Failed to execute SQL: %v\n", err)
	}

	fmt.Println("All tables created successfully!")
}

// isDuplicateDatabaseError checks if the error is due to the database already existing
func isDuplicateDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	// Check for SQLSTATE 42P04 (duplicate database)
	return (err.Error() == "ERROR: database \"dailzo\" already exists (SQLSTATE 42P04)") ||
		(len(err.Error()) > 0 && (err.Error()[0:6] == "ERROR:" && err.Error()[len(err.Error())-7:] == "42P04)"))
}

// CloseDatabase closes the connection pool.
func CloseDatabase() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed.")
	}
}
