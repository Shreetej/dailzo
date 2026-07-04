package main

import (
	"dailzo/config"
	"dailzo/controllers"
	"dailzo/db"
	"dailzo/internal/api"
	"dailzo/internal/server"
	ws "dailzo/internal/websocket"
	"dailzo/middleware"
	"dailzo/pkg/response"
	"dailzo/repository"
	"dailzo/routes"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// Load config and setup logger
	cfg := config.LoadConfig()
	log := config.SetupLogger()

	// Connect to database
	db.ConnectDatabase(cfg)
	defer db.CloseDatabase()

	fmt.Print("User details:")

	// ─── Repositories ───────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db.DB)
	otpRepo := repository.NewOTPRepository(db.DB)
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
	offerRepo := repository.NewOfferRepository(db.DB)

	// New repositories for v1 API
	deliveryRepo := repository.NewDeliveryRepository(db.DB)
	groceryRepo := repository.NewGroceryRepository(db.DB)
	adminRepo := repository.NewAdminRepository(db.DB)

	// ─── Legacy Controllers ─────────────────────────────────────────────
	consentController := controllers.NewConsentController(consentRepo)
	emailController := controllers.NewEmailControllerWithConsent(consentController)
	offerController := controllers.NewOfferController(offerRepo)
	userController := controllers.NewUserController(userRepo, otpRepo)
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
	marketingRepo := repository.NewMarketingRepository(db.DB)
	marketingController := controllers.NewMarketingController(marketingRepo)
	registrationRepo := repository.NewRegistrationRepository(db.DB)
	registrationController := controllers.NewRegistrationController(registrationRepo)

	// ─── v1 API Server (implements OpenAPI ServerInterface) ─────────────
	apiServer := server.NewServer(
		userRepo, orderRepo, foodProductRepo,
		deliveryRepo, groceryRepo, adminRepo, otpRepo,
	)

	// ─── WebSocket Hub ──────────────────────────────────────────────────
	hub := ws.NewHub()
	go hub.Run()

	// ─── Fiber App ──────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return response.Error(c, code, "ERROR", err.Error())
		},
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// ─── Health Check ───────────────────────────────────────────────────
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "dailzo-backend",
		})
	})

	// ─── Legacy Routes (/api) ───────────────────────────────────────────
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
		marketingController,
	)

	// ─── v1 API Routes (/api/v1) ────────────────────────────────────────
	v1 := app.Group("/api/v1")

	// Auth endpoints must be reachable without a token; everything else on
	// v1 goes through JWT.
	publicV1Paths := map[string]bool{
		"/api/v1/auth/login":         true,
		"/api/v1/auth/signup":        true,
		"/api/v1/auth/send-otp":      true,
		"/api/v1/auth/verify-otp":    true,
		"/api/v1/is-user-registered": true,
	}
	jwtHandler := middleware.JWTMiddleware()
	v1Auth := func(c *fiber.Ctx) error {
		if publicV1Paths[c.Path()] {
			return c.Next()
		}
		return jwtHandler(c)
	}

	// Marketing & vendor-ops routes on v1 (same handlers as the legacy /api
	// copies registered in routes.SetupRoutes). Registered before the
	// generated handlers so they are not shadowed.
	v1.Get("/marketing/discounts", marketingController.GetDiscounts)
	v1.Post("/marketing/discounts", marketingController.CreateDiscount)
	v1.Get("/marketing/ads", marketingController.GetAdCampaigns)
	v1.Post("/marketing/ads", marketingController.CreateAdCampaign)
	v1.Post("/marketing/ads/:id/stop", marketingController.StopAdCampaign)
	v1.Get("/marketing/ads/packs", marketingController.GetAdPacks)
	v1.Post("/orders/:order_id/out-of-stock", marketingController.MarkOrderItemsOutOfStock)
	v1.Get("/orders/:order_id/out-of-stock", marketingController.GetOrderOutOfStockItems)
	v1.Post("/restaurant/:restaurant_id/outlets", marketingController.CreateVendorOutlet)

	// Restaurant registration flow used by the Partner app.
	v1.Post("/restaurant/register", registrationController.RegisterRestaurant)
	v1.Put("/restaurant/:restaurant_id/payment", registrationController.UpdatePaymentInfo)
	v1.Put("/restaurant/:restaurant_id/complete", registrationController.CompleteRegistration)
	v1.Get("/restaurant/:restaurant_id/outlets", registrationController.GetVendorOutlets)
	v1.Get("/restaurant/:restaurant_id", registrationController.GetRestaurantData)

	// Register OpenAPI-generated routes with JWT middleware
	api.RegisterHandlersWithOptions(v1, apiServer, api.FiberServerOptions{
		Middlewares: []api.MiddlewareFunc{
			api.MiddlewareFunc(v1Auth),
		},
	})

	// ─── WebSocket Route ────────────────────────────────────────────────
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/delivery/track/:orderId", websocket.New(ws.TrackDeliveryHandler(hub)))

	// ─── Graceful Shutdown ──────────────────────────────────────────────
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info().Msg("Shutting down DB connections")
		db.CloseDatabase()
		app.Shutdown()
		log.Info().Msg("DB shutdown successful")
	}()

	// ─── Start Server ───────────────────────────────────────────────────
	log.Info().Str("port", cfg.AppPort).Msg("Starting Dailzo backend")
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		panic(err)
	}
}
