package config

import (
	"log"
	"os"

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
	JWTSecret    string
	SmtpHost     string
	SmtpPort     string
	SmtpEmail    string
	smtpPassword string
}

func LoadConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Using environment variables.")
	}

	return Config{
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "roger"),
		DBName:       getEnv("DB_NAME", "dailzo"),
		AppPort:      getEnv("APP_PORT", "3000"),
		JWTSecret:    getEnv("JWT_SECRET", "secret"),
		SmtpHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SmtpPort:     getEnv("SMTP_PORT", "587"),
		SmtpEmail:    getEnv("SMTP_EMAIL", "nitej.techno@gmail.com"),
		smtpPassword: getEnv("SMTP_PASSWORD", "Nitej@2024"),
	}
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
