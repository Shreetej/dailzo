package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

type Config struct {
	// DatabaseURL is the whole connection string when the platform supplies
	// one (Railway, Render, Fly and Heroku all inject DATABASE_URL for an
	// attached Postgres). It wins over the DB_* parts below, which stay for
	// local development.
	DatabaseURL  string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	AppPort      string
	AppEnv       string
	CORSOrigins  string
	JWTSecret    string
	SmtpHost     string
	SmtpPort     string
	SmtpEmail    string
	smtpPassword string
}

var (
	loadOnce sync.Once
	loaded   Config
)

// LoadConfig reads the environment once (main.go and utils/jwt.go's init()
// both call it independently) and caches the result — otherwise every call
// re-parses .env and can print the "No .env file found" notice more than once.
func LoadConfig() Config {
	loadOnce.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found. Using environment variables.")
		}

		databaseURL := firstEnv("DATABASE_URL", "POSTGRES_URL", "DATABASE_PRIVATE_URL")

		dbPassword := getEnv("DB_PASSWORD", "roger")
		// Only meaningful when the DB_* parts are actually being used; a
		// platform-supplied DATABASE_URL carries its own credentials.
		if dbPassword == "roger" && databaseURL == "" {
			log.Println("WARNING: DB_PASSWORD is not set — falling back to the well-known dev default. Set DB_PASSWORD before production.")
		}

		// Hosts inject the port to listen on as PORT. APP_PORT stays as the
		// local override; without this the container bound 3000 while the
		// platform routed to something else.
		appPort := firstEnv("PORT", "APP_PORT")
		if appPort == "" {
			appPort = "3000"
		}

		loaded = Config{
			DatabaseURL: databaseURL,
			DBHost:      getEnv("DB_HOST", "localhost"),
			DBPort:      getEnv("DB_PORT", "5432"),
			DBUser:      getEnv("DB_USER", "postgres"),
			DBPassword:  dbPassword,
			DBName:      getEnv("DB_NAME", "dailzo"),
			// Managed Postgres refuses plaintext connections; a local one
			// usually has no TLS at all.
			DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
			AppPort:     appPort,
			AppEnv:      getEnv("APP_ENV", "development"),
			CORSOrigins: getEnv("CORS_ORIGINS", ""),
			// Secrets have NO hardcoded fallback — they must come from the env/.env.
			// (An empty JWT secret is rejected at startup in main.go.)
			JWTSecret:    getEnv("JWT_SECRET", ""),
			SmtpHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SmtpPort:     getEnv("SMTP_PORT", "587"),
			SmtpEmail:    getEnv("SMTP_EMAIL", ""),
			smtpPassword: getEnv("SMTP_PASSWORD", ""),
		}
	})
	return loaded
}

func SetupLogger() zerolog.Logger {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("app", "Dailzo").Logger()
	return log
}

// DSN is the single place a connection string is built, so the server and
// cmd/migrate can never disagree about which database they are talking to.
func (c Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(c.DBUser), url.QueryEscape(c.DBPassword),
		c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

// RedactedDSN is the DSN with the password removed, for logs and errors.
func (c Config) RedactedDSN() string {
	u, err := url.Parse(c.DSN())
	if err != nil {
		return "(unparseable connection string)"
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), "xxxxx")
		}
	}
	return u.Redacted()
}

// firstEnv returns the first of these variables that is set and non-empty.
func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			return v
		}
	}
	return ""
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
