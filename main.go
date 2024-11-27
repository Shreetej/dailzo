package main

import (
	"dailzo/config"
	"dailzo/controllers"
	"dailzo/db"
	"dailzo/repository"
	"dailzo/routes"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Load config and setup logger
	cfg := config.LoadConfig()
	log := config.SetupLogger()

	// Connect to database
	db.ConnectDatabase(cfg)
	defer db.CloseDatabase()

	// Initialize repositories and controllers
	userRepo := repository.NewUserRepository(db.DB)
	userController := controllers.NewUserController(userRepo)

	// Initialize Fiber app
	app := fiber.New()

	// Setup routes
	routes.SetupRoutes(app, userController)

	// Graceful shutdown handling
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info().Msg("Shutting down DB connections")
		db.CloseDatabase()
		app.Shutdown()
		log.Info().Msg("DB shutdown sucessful")
	}()

	// Start the server
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		panic(err)
	}
	log.Info().Str("App started at port :", cfg.DBPort)
}
