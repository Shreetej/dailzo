package main

import (
	"dailzo/config"
	"dailzo/controllers"
	"dailzo/db"
	"dailzo/repository"
	"dailzo/routes"
	"fmt"
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

	fmt.Print("User details:")

	// // Initialize repositories and controllers
	// userRepo := repository.NewUserRepository(db.DB)
	// userController := controllers.NewUserController(userRepo)

	// addressRepo := repository.NewAddressRepository(db.DB)
	// addressController := controllers.NewAddressController(addressRepo)

	// foodProductRepo := repository.NewFoodProductRepository(db.DB)
	// foodProductController := controllers.NewFoodProductController(foodProductRepo)

	// productVariantRepo := repository.NewProductVariantRepository(db.DB)
	// productVariantController := controllers.NewProductVariantController(productVariantRepo)

	// Repositories
	userRepo := repository.NewUserRepository(db.DB)
	addressRepo := repository.NewAddressRepository(db.DB)
	foodProductRepo := repository.NewFoodProductRepository(db.DB)
	productVariantRepo := repository.NewProductVariantRepository(db.DB)
	paymentRepo := repository.NewPaymentRepository(db.DB)
	orderRepo := repository.NewOrderRepository(db.DB)
	orderItemRepo := repository.NewOrderItemRepository(db.DB)
	payMethodRepo := repository.NewPaymentMethodRepository(db.DB)
	ratingRepo := repository.NewRatingRepository(db.DB)
	refundRepo := repository.NewRefundRepository(db.DB)
	restaurantRepo := repository.NewRestaurantRepository(db.DB)

	consentRepo := repository.NewConsentRepository(db.DB)
	consentController := controllers.NewConsentController(consentRepo)
	emailController := controllers.NewEmailControllerWithConsent(consentController)

	// Initialize OfferRepository
	offerRepo := repository.NewOfferRepository(db.DB)

	// Initialize OfferController
	offerController := controllers.NewOfferController(offerRepo)

	// Controllers
	userController := controllers.NewUserController(userRepo)
	addressController := controllers.NewAddressController(addressRepo)
	foodProductController := controllers.NewFoodProductController(foodProductRepo)
	productVariantController := controllers.NewProductVariantController(productVariantRepo)
	paymentController := controllers.NewPaymentController(paymentRepo)
	orderController := controllers.NewOrderController(orderRepo)
	orderItemController := controllers.NewOrderItemController(orderItemRepo)
	payMethodController := controllers.NewPaymentMethodController(payMethodRepo)
	ratingController := controllers.NewRatingController(ratingRepo)
	refundController := controllers.NewRefundController(refundRepo)
	restaurantController := controllers.NewRestaurantController(restaurantRepo)

	//emailController := controllers.NewEmailController()

	// Initialize Fiber app
	app := fiber.New()

	// Setup routes
	routes.SetupRoutes(
		app,
		userController,
		addressController,
		foodProductController,
		productVariantController,
		paymentController,
		orderController,
		orderItemController,
		payMethodController,
		ratingController,
		refundController,
		restaurantController,
		emailController,
		offerController,
	)

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
