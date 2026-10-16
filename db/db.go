package db

import (
	"context"
	"dailzo/config"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func envInt(key string, fallback int32) int32 {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return int32(n)
		}
	}
	return fallback
}

var DB *pgxpool.Pool

// ConnectDatabase initializes a connection to the PostgreSQL database using pgx v5.
func ConnectDatabase(cfg config.Config) {
	// One DSN builder for the server and cmd/migrate — and it prefers a
	// platform-supplied DATABASE_URL. Without that, a container with no DB_*
	// variables silently fell back to localhost:5432 and crash-looped against
	// a database that was never there.
	dsn := cfg.DSN()

	// Configure database connection settings
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse database configuration (%s): %v\n", cfg.RedactedDSN(), err)
	}

	// Connection pool sizing — overridable via DB_MAX_CONNS/DB_MIN_CONNS since
	// the right size depends on deployment (single dev instance vs. a
	// production box under real concurrent load); 10/2 are dev-scale defaults.
	config.MaxConns = envInt("DB_MAX_CONNS", 10)
	config.MinConns = envInt("DB_MIN_CONNS", 2)
	config.MaxConnLifetime = time.Hour        // Maximum lifetime of a connection
	config.MaxConnIdleTime = 30 * time.Minute // Maximum idle time for a connection

	// Create a new connection pool
	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	// Test the connection with a 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = DB.Ping(ctx); err != nil {
		// Name the target: the failure used to print only the pgx error, so a
		// deployment that had fallen back to localhost looked identical to a
		// deployment whose real database was down.
		log.Fatalf("Unable to reach the database at %s: %v\n"+
			"Set DATABASE_URL (or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME) for this environment.\n",
			cfg.RedactedDSN(), err)
	}

	log.Println("Connected to the database successfully!")

	fmt.Println("All tables created successfully!")

}

// CloseDatabase closes the connection pool.
func CloseDatabase() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed.")
	}
}
