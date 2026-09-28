package config

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

type Config struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
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

		dbPassword := getEnv("DB_PASSWORD", "roger")
		if dbPassword == "roger" {
			log.Println("WARNING: DB_PASSWORD is not set — falling back to the well-known dev default. Set DB_PASSWORD before production.")
		}

		loaded = Config{
			DBHost:      getEnv("DB_HOST", "localhost"),
			DBPort:      getEnv("DB_PORT", "5432"),
			DBUser:      getEnv("DB_USER", "postgres"),
			DBPassword:  dbPassword,
			DBName:      getEnv("DB_NAME", "dailzo"),
			AppPort:     getEnv("APP_PORT", "3000"),
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

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
