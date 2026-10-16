package main

import (
	"context"
	"dailzo/config"
	"dailzo/controllers"
	"dailzo/db"
	"dailzo/internal/api"
	"dailzo/internal/server"
	ws "dailzo/internal/websocket"
	"dailzo/middleware"
	"dailzo/models"
	"dailzo/payments"
	"dailzo/pkg/i18n"
	"dailzo/pkg/response"
	"dailzo/repository"
	"dailzo/routes"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
)

func main() {
	// Load config and setup logger
	cfg := config.LoadConfig()
	log := config.SetupLogger()

	// Fail-safe check on the JWT secret — do not run with a weak/default secret
	// in production. Empty is fatal; the known-weak default logs a loud warning.
	if cfg.JWTSecret == "" {
		log.Fatal().Msg("JWT_SECRET is not set — refusing to start")
	}
	if cfg.JWTSecret == "secret" || len(cfg.JWTSecret) < 16 {
		log.Warn().Msg("JWT_SECRET is weak/default — set a strong (32+ char) secret before production")
	}

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
	deliveryVendorRepo := repository.NewDeliveryVendorRepository(db.DB)
	groceryRepo := repository.NewGroceryRepository(db.DB)
	adminRepo := repository.NewAdminRepository(db.DB)

	// Vendor-facing complaints and outlet ratings. The Partner app has always
	// called these endpoints; they had no server-side implementation, so every
	// complaint and ratings screen in the app 404'd.
	vendorSupportRepo := repository.NewVendorSupportRepository(db.DB)
	vendorSupportController := controllers.NewVendorSupportController(vendorSupportRepo)

	// Server-driven i18n: translate-on-write + admin-configurable surfaces.
	translator := i18n.NewTranslator()
	translationRepo := repository.NewTranslationRepository(db.DB, translator)
	configRepo := repository.NewConfigRepository(db.DB, translationRepo)
	configController := controllers.NewConfigController(configRepo)

	// ─── Legacy Controllers ─────────────────────────────────────────────
	consentController := controllers.NewConsentController(consentRepo)
	emailController := controllers.NewEmailControllerWithConsent(consentController)
	offerController := controllers.NewOfferController(offerRepo)
	userController := controllers.NewUserController(userRepo, otpRepo)
	addressController := controllers.NewAddressController(addressRepo)
	foodProductController := controllers.NewFoodProductController(foodProductRepo, translationRepo)
	productVariantController := controllers.NewProductVariantController(productVariantRepo)
	paymentController := controllers.NewPaymentController(paymentRepo)
	orderController := controllers.NewOrderController(orderRepo)
	orderItemController := controllers.NewOrderItemController(orderItemRepo)
	payMethodController := controllers.NewPaymentMethodController(payMethodRepo)
	ratingController := controllers.NewRatingController(ratingRepo)
	refundController := controllers.NewRefundController(refundRepo)
	restaurantController := controllers.NewRestaurantController(restaurantRepo)
	marketingRepo := repository.NewMarketingRepository(db.DB)
	registrationRepo := repository.NewRegistrationRepository(db.DB)
	marketingController := controllers.NewMarketingController(marketingRepo, registrationRepo)
	registrationController := controllers.NewRegistrationController(registrationRepo)
	storefrontRepo := repository.NewStorefrontRepository(db.DB)
	// Fees and VAT are a server-owned policy an admin edits, not constants in
	// the mobile apps.
	billingRepo := repository.NewBillingRepository(db.DB)
	storefrontController := controllers.NewStorefrontController(storefrontRepo, translationRepo, vendorSupportRepo, billingRepo)

	// Vendor "Product Intelligence" analytics (top products/categories,
	// low sellers) aggregated from orders.items jsonb.
	analyticsRepo := repository.NewAnalyticsRepository(db.DB)
	analyticsController := controllers.NewAnalyticsController(analyticsRepo)

	// Delivery-partner intelligence (earnings, SLA, peak-time insights, payout).
	deliveryAnalyticsRepo := repository.NewDeliveryAnalyticsRepository(db.DB)
	deliveryAnalyticsController := controllers.NewDeliveryAnalyticsController(deliveryAnalyticsRepo)

	// Payments redesign (docs/PAYMENTS_REDESIGN.md): immediate-capture + COD,
	// gateway behind an interface (mock until HyperPay/PayTabs creds exist).
	paymentV2Repo := repository.NewPaymentV2Repository(db.DB)
	paymentGateways := map[string]payments.Gateway{
		models.GatewayMock: payments.NewMockGateway(),
	}
	paymentV2Controller := controllers.NewPaymentV2Controller(paymentV2Repo, paymentGateways, models.GatewayMock)
	adminPartnerController := controllers.NewAdminPartnerController(adminRepo)

	// ─── WebSocket Hub ──────────────────────────────────────────────────
	// Created before apiServer so the delivery location-ping handler can
	// broadcast through it (previously nothing ever fed the hub at all).
	hub := ws.NewHub()
	go hub.Run()

	// ─── v1 API Server (implements OpenAPI ServerInterface) ─────────────
	apiServer := server.NewServer(
		userRepo, orderRepo, foodProductRepo,
		deliveryRepo, deliveryVendorRepo, groceryRepo, adminRepo, otpRepo,
		hub,
	)

	// ─── Payment background jobs (docs/PAYMENTS_REDESIGN.md) ────────────
	// Reconciliation on the configured cadence + status-poll backstop for
	// payments stuck awaiting a webhook.
	jobCtx := context.Background()
	go payments.StartReconciliationScheduler(jobCtx, paymentV2Repo)
	go payments.StartStatusPollBackstop(jobCtx, paymentV2Repo, paymentGateways, 2*time.Minute, 10*time.Minute)

	// ─── Fiber App ──────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		BodyLimit:    10 * 1024 * 1024, // 10 MB
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
	app.Use(compress.New())

	// Access logging — no request/audit trail existed at all before this
	// (only ad-hoc application logs). Format omits headers/body so it
	// doesn't leak Authorization tokens or request payloads into logs.
	app.Use(logger.New(logger.Config{
		Format: "${time} ${status} ${latency} ${method} ${path} ${ip}\n",
	}))

	// CORS — restrict to configured origins in production (CORS_ORIGINS,
	// comma-separated); defaults to "*" for local dev. A wide-open origin
	// combined with credentialed (Authorization-header) requests is unsafe in
	// production, so refuse to start rather than silently serve that
	// combination — APP_ENV must be set explicitly to allow "*".
	//
	// "none" is the third answer, and the right one for this API today: its
	// clients are the two Flutter apps, which send no Origin header at all,
	// so CORS never applies to them and there is no real browser origin to
	// name. Naming a fake one to satisfy the check would be worse than saying
	// plainly that no browser may call this API cross-origin.
	corsOrigins := strings.TrimSpace(cfg.CORSOrigins)
	switch corsOrigins {
	case "none":
		log.Info().Msg("CORS_ORIGINS=none — no cross-origin browser access; native app clients are unaffected.")
	case "", "*":
		if cfg.AppEnv == "production" {
			log.Fatal().Msg("CORS_ORIGINS is unset/\"*\" with APP_ENV=production — refusing to start with a wide-open CORS policy. Set CORS_ORIGINS to your real frontend origin(s), or to \"none\" if only native apps call this API.")
		}
		app.Use(cors.New(cors.Config{
			AllowOrigins: "*",
			AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
			AllowHeaders: "Origin,Content-Type,Accept,Authorization",
			MaxAge:       300,
		}))
	default:
		// Fiber panics on a malformed origin, with a stack trace and no hint
		// about which value was wrong. Check first and say so plainly.
		if bad := invalidOrigins(corsOrigins); len(bad) > 0 {
			log.Fatal().Msgf("CORS_ORIGINS contains %d value(s) that are not valid origins: %s. "+
				"Each must be scheme://host[:port] with no trailing slash or path (e.g. https://app.example.com), "+
				"or use \"none\" when only native apps call this API.",
				len(bad), strings.Join(bad, ", "))
		}
		app.Use(cors.New(cors.Config{
			AllowOrigins: corsOrigins,
			AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
			AllowHeaders: "Origin,Content-Type,Accept,Authorization",
			MaxAge:       300,
		}))
	}

	// Rate limiting — throttles brute force / abuse. Generous global cap;
	// skips health check and websocket upgrades. Cap is overridable via
	// RATE_LIMIT_MAX (defaults to 300 requests/minute per IP).
	rateLimitMax := 300
	if v := os.Getenv("RATE_LIMIT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rateLimitMax = n
		}
	}
	app.Use(limiter.New(limiter.Config{
		Max:        rateLimitMax,
		Expiration: 1 * time.Minute,
		Next: func(c *fiber.Ctx) bool {
			p := c.Path()
			return p == "/health" || strings.HasPrefix(p, "/ws")
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests, "RATE_LIMITED",
				"Too many requests, please slow down.")
		},
	}))

	// ─── Health Check ───────────────────────────────────────────────────
	app.Get("/health", healthCheck)

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
		// The generated /admin/* routes (approvals, partners, complaints, KPIs)
		// were previously reachable by any authenticated user — this whole
		// route group only ever got v1Auth's plain JWT check, never
		// middleware.AdminOnly(), unlike /config/* and /users which chain it
		// explicitly. middleware.AuthenticateRequest (unlike JWTMiddleware)
		// doesn't call c.Next() itself, so the admin-role check below can run
		// between authentication and the real handler.
		if err := middleware.AuthenticateRequest(c); err != nil {
			return err
		}
		if strings.HasPrefix(c.Path(), "/api/v1/admin/") {
			userType, _ := c.Locals("user_type").(string)
			if userType != "admin" {
				return response.Forbidden(c, "Insufficient permissions")
			}
		}
		return c.Next()
	}

	// Marketing & vendor-ops routes on v1 (same handlers as the legacy /api
	// copies registered in routes.SetupRoutes). Registered before the
	// generated handlers so they are not shadowed.
	// Read-only public marketing displays (customer app shows these on Home
	// before/without a vendor session) stay public. Everything that mutates
	// vendor/order state, or reads a specific order's contents, requires auth.
	v1.Get("/marketing/discounts", marketingController.GetDiscounts)
	v1.Post("/marketing/discounts", jwtHandler, marketingController.CreateDiscount)
	v1.Get("/marketing/ads", marketingController.GetAdCampaigns)
	v1.Post("/marketing/ads", jwtHandler, marketingController.CreateAdCampaign)
	v1.Post("/marketing/ads/:id/stop", jwtHandler, marketingController.StopAdCampaign)
	v1.Get("/marketing/ads/packs", marketingController.GetAdPacks)
	v1.Post("/orders/:order_id/out-of-stock", jwtHandler, marketingController.MarkOrderItemsOutOfStock)
	v1.Get("/orders/:order_id/out-of-stock", jwtHandler, marketingController.GetOrderOutOfStockItems)
	v1.Post("/restaurant/:restaurant_id/outlets", jwtHandler, marketingController.CreateVendorOutlet)

	// ─── Storefront (customer app) ──────────────────────────────────────
	// Catalog browse is public; cart/orders/complaints/refunds require JWT
	// (they read the authenticated user via ctx.Locals("user_id")).
	v1.Get("/catalog/categories", storefrontController.GetCategories)
	v1.Get("/catalog/products", storefrontController.GetProducts)
	v1.Get("/catalog/restaurants", storefrontController.GetRestaurants)
	// Customisation groups for one product. Registered after the plain
	// /catalog/products route so it doesn't shadow it.
	v1.Get("/catalog/products/:id/options", storefrontController.GetProductOptions)
	// Billing policy + the priced cart. The apps render these rather than
	// computing fees or VAT themselves.
	v1.Get("/config/billing", storefrontController.GetBillingConfig)
	v1.Get("/admin/billing-config", jwtHandler, storefrontController.ListBillingConfigs)
	v1.Put("/admin/billing-config/:region", jwtHandler, storefrontController.UpdateBillingConfig)
	v1.Get("/cart/quote", jwtHandler, storefrontController.GetCartQuote)
	// Prices the lines the client is displaying, so the bill can't disagree
	// with the list above it when the local and server carts drift apart.
	v1.Post("/cart/quote", storefrontController.PostCartQuote)

	// Server-driven config: options (enum lists) + content sections + category
	// CRUD. GETs are public and localized via Accept-Language; mutations require
	// auth (admin). Content added here is auto-translated on write.
	v1.Get("/config/options", configController.GetOptions)
	v1.Get("/config/sections", configController.GetSections)
	v1.Post("/config/options", jwtHandler, middleware.AdminOnly(), configController.UpsertOption)
	v1.Delete("/config/options/:id", jwtHandler, middleware.AdminOnly(), configController.DeleteOption)
	v1.Post("/config/sections", jwtHandler, middleware.AdminOnly(), configController.UpsertSection)
	v1.Delete("/config/sections/:id", jwtHandler, middleware.AdminOnly(), configController.DeleteSection)
	v1.Post("/config/categories", jwtHandler, middleware.AdminOnly(), configController.UpsertCategory)
	v1.Delete("/config/categories/:id", jwtHandler, middleware.AdminOnly(), configController.DeleteCategory)
	v1.Put("/config/translations", jwtHandler, middleware.AdminOnly(), configController.SetTranslation)
	v1.Get("/cart", jwtHandler, storefrontController.GetCart)
	v1.Post("/cart/items", jwtHandler, storefrontController.AddCartItem)
	v1.Patch("/cart/items/:id", jwtHandler, storefrontController.UpdateCartItem)
	v1.Delete("/cart/items/:id", jwtHandler, storefrontController.DeleteCartItem)
	v1.Delete("/cart", jwtHandler, storefrontController.ClearCart)
	v1.Post("/orders", jwtHandler, storefrontController.CreateOrder)
	v1.Get("/my-orders", jwtHandler, storefrontController.GetMyOrders)
	v1.Get("/order/:id", jwtHandler, storefrontController.GetOrder)
	v1.Get("/vendor/orders", jwtHandler, storefrontController.GetVendorOrders)
	v1.Patch("/vendor/orders/:id/status", jwtHandler, storefrontController.UpdateVendorOrderStatus)
	v1.Post("/complaints", jwtHandler, storefrontController.CreateComplaint)
	v1.Get("/my-complaints", jwtHandler, storefrontController.GetComplaints)
	// Vendor product create/delete on v1 (maps to the food-product controller;
	// request-shape reconciliation with the vendor Product model is Phase 3).
	v1.Post("/products", jwtHandler, foodProductController.CreateFoodProduct)
	v1.Delete("/products/:id", jwtHandler, foodProductController.DeleteFoodProduct)
	v1.Post("/refunds", jwtHandler, storefrontController.CreateRefund)
	v1.Get("/my-refunds", jwtHandler, storefrontController.GetRefunds)

	// Vendor Product Intelligence analytics.
	v1.Get("/analytics/vendor/products/trending", jwtHandler, analyticsController.ProductTrends)
	v1.Get("/analytics/vendor/categories/trending", jwtHandler, analyticsController.CategoryTrends)
	v1.Get("/analytics/vendor/products/low-sellers", jwtHandler, analyticsController.LowSellers)
	v1.Get("/vendor/search-demand/unmet", jwtHandler, analyticsController.UnmetDemand)
	// /vendor/payout-summary intentionally stays on the platform-wide
	// analyticsController.PayoutSummary (aggregate-by-role market figure,
	// same family as the trending/low-sellers endpoints above) — it's not
	// currently called by the app (VendorPayoutNotifier has no callers).
	v1.Get("/vendor/payout-summary", jwtHandler, analyticsController.PayoutSummary)
	// Vendor store open/closed switch (restaurant + grocery).
	v1.Get("/vendor/store-status", jwtHandler, analyticsController.GetStoreStatus)
	v1.Put("/vendor/store-status", jwtHandler, analyticsController.SetStoreStatus)

	// GetDeliveryAvailable/PostDeliveryAvailableClaim, the grocery dashboard
	// handlers below, etc. aren't part of the OpenAPI-generated
	// api.ServerInterface apiServer is typed as, hence the concrete-type
	// assertion (same underlying *server.Server either way).
	deliveryQueue := apiServer.(*server.Server)

	// Grocery vendor dashboard. These handlers already existed in
	// internal/server/grocery_handlers.go but were never registered as
	// routes, so the vendor app's grocery dashboard/profile screens 404'd
	// on every one of these and silently fell back to client-side mock data.
	// payout-summary specifically used to be routed to the platform-wide
	// analyticsController.PayoutSummary (aggregate revenue across ALL
	// grocery vendors) instead of this vendor-scoped one — any grocery
	// vendor would see every other vendor's revenue summed into their own
	// "next payout" figure.
	v1.Post("/grocery/onboarding", jwtHandler, deliveryQueue.PostGroceryOnboarding)
	v1.Get("/grocery/profile", jwtHandler, deliveryQueue.GetGroceryProfile)
	v1.Get("/grocery/kpis", jwtHandler, deliveryQueue.GetGroceryKpis)
	v1.Get("/grocery/expiry-alerts", jwtHandler, deliveryQueue.GetGroceryExpiryAlerts)
	v1.Get("/grocery/stock-alerts", jwtHandler, deliveryQueue.GetGroceryStockAlerts)
	v1.Get("/grocery/settings", jwtHandler, deliveryQueue.GetGrocerySettings)
	v1.Get("/grocery/documents", jwtHandler, deliveryQueue.GetGroceryDocuments)
	v1.Get("/grocery/payout-account", jwtHandler, deliveryQueue.GetGroceryPayoutAccount)
	v1.Get("/grocery/payout-summary", jwtHandler, deliveryQueue.GetGroceryPayoutSummary)
	// Customer search-demand logging (feeds vendor unmet-demand). Public —
	// search demand is often pre-login/anonymous; user_id is captured when present.
	v1.Post("/search-log", analyticsController.LogSearch)

	// Delivery vendor (fleet/agency) — the caller's own profile (which
	// includes their invite_code once approved) and their own riders' review
	// queue. Ownership-scoped by the caller's user_id inside each handler,
	// not by a path parameter, so one fleet can't touch another's riders.
	v1.Get("/delivery-vendor/mine", jwtHandler, deliveryQueue.GetDeliveryVendorMine)
	v1.Get("/delivery-vendor/riders", jwtHandler, deliveryQueue.GetDeliveryVendorRiders)
	v1.Post("/delivery-vendor/riders/:id/approve", jwtHandler, deliveryQueue.PostDeliveryVendorRidersApprove)
	v1.Post("/delivery-vendor/riders/:id/reject", jwtHandler, deliveryQueue.PostDeliveryVendorRidersReject)
	v1.Get("/delivery-vendor/riders/active", jwtHandler, deliveryQueue.GetDeliveryVendorRidersActive)
	v1.Get("/delivery-vendor/riders/suspended", jwtHandler, deliveryQueue.GetDeliveryVendorRidersSuspended)
	v1.Post("/delivery-vendor/riders/:id/suspend", jwtHandler, deliveryQueue.PostDeliveryVendorRidersSuspend)
	v1.Post("/delivery-vendor/riders/:id/reinstate", jwtHandler, deliveryQueue.PostDeliveryVendorRidersReinstate)
	// Complaints on orders this fleet's own riders delivered — resolve
	// (service-level, no refund authority), escalate (hand off to platform
	// admin), or log an investigation update without closing it.
	// Fleet dashboard + payouts, aggregated over this fleet's own riders.
	v1.Get("/delivery-vendor/kpis", jwtHandler, deliveryQueue.GetDeliveryVendorKpis)
	v1.Get("/delivery-vendor/payout-summary", jwtHandler, deliveryQueue.GetDeliveryVendorPayoutSummary)
	v1.Get("/delivery-vendor/earnings", jwtHandler, deliveryQueue.GetDeliveryVendorRiderEarnings)
	v1.Get("/delivery-vendor/complaints", jwtHandler, deliveryQueue.GetDeliveryVendorComplaints)
	v1.Post("/delivery-vendor/complaints/:id/resolve", jwtHandler, deliveryQueue.PostDeliveryVendorComplaintsResolve)
	v1.Post("/delivery-vendor/complaints/:id/escalate", jwtHandler, deliveryQueue.PostDeliveryVendorComplaintsEscalate)
	v1.Post("/delivery-vendor/complaints/:id/update", jwtHandler, deliveryQueue.PostDeliveryVendorComplaintsUpdate)

	// Rider vehicle + KYC checklist, derived from the delivery_profiles
	// columns captured at onboarding (there is no separate vehicle or
	// document table). The Partner app's delivery profile screen has always
	// called both; neither had a route, so the screen 404'd on every open.
	v1.Get("/delivery/vehicle", jwtHandler, deliveryQueue.GetDeliveryVehicle)
	v1.Get("/delivery/documents", jwtHandler, deliveryQueue.GetDeliveryDocuments)

	// Vendor-facing complaints. Distinct from /my-complaints (the customer
	// reading their own) and /admin/complaints (platform-wide): these are
	// complaints filed against THIS vendor's orders, scoped by the caller's
	// user_id. "summary" is registered before ":id" so Fiber's
	// registration-order matching doesn't capture it as a complaint id.
	v1.Get("/complaints/summary", jwtHandler, vendorSupportController.GetComplaintSummary)
	v1.Get("/complaints", jwtHandler, vendorSupportController.GetComplaints)
	v1.Get("/complaints/:id", jwtHandler, vendorSupportController.GetComplaintByID)
	v1.Patch("/complaints/:id/status", jwtHandler, vendorSupportController.UpdateComplaintStatus)

	// Outlet ratings for the vendor's own outlets.
	v1.Get("/ratings/outlets", jwtHandler, vendorSupportController.GetOutletRatings)
	v1.Get("/ratings/outlets/:outletId", jwtHandler, vendorSupportController.GetOutletRating)
	v1.Get("/ratings/outlets/:outletId/reviews", jwtHandler, vendorSupportController.GetOutletReviews)

	// Stateless-JWT logout: the client discards its token, this confirms the
	// call so the app's logout path doesn't fail on a 404.
	v1.Post("/auth/logout", jwtHandler, deliveryQueue.PostAuthLogout)

	// Health check is also exposed under /api/v1 because both apps build
	// every request off a base URL that already ends in /api/v1 — a probe of
	// the root /health route below is unreachable for them.
	v1.Get("/health", healthCheck)

	// Delivery-partner intelligence dashboard.
	v1.Get("/delivery/kpis", jwtHandler, deliveryAnalyticsController.Kpis)
	v1.Get("/delivery/earnings/weekly", jwtHandler, deliveryAnalyticsController.WeeklyEarnings)
	v1.Get("/delivery/sla-kpis", jwtHandler, deliveryAnalyticsController.SlaKpis)
	v1.Get("/delivery/earnings-insights", jwtHandler, deliveryAnalyticsController.EarningsInsights)
	v1.Get("/delivery/recommendations", jwtHandler, deliveryAnalyticsController.Recommendations)
	v1.Get("/delivery/shifts", jwtHandler, deliveryAnalyticsController.Shifts)
	v1.Get("/delivery/payout-summary", jwtHandler, deliveryAnalyticsController.PayoutSummary)
	v1.Get("/delivery/online", jwtHandler, deliveryAnalyticsController.GetOnline)
	v1.Put("/delivery/online", jwtHandler, deliveryAnalyticsController.SetOnline)
	// Real delivery job queue: ready orders with no partner yet, and an atomic
	// claim. Delivery-partner-only now that GenerateJWT actually embeds
	// user_type (previously RoleMiddleware silently matched nothing).
	v1.Get("/delivery/available", jwtHandler, middleware.DeliveryOnly(), deliveryQueue.GetDeliveryAvailable)
	v1.Post("/delivery/available/:orderId/claim", jwtHandler, middleware.DeliveryOnly(), deliveryQueue.PostDeliveryAvailableClaim)
	// Live location ping -> feeds the websocket hub so /ws/delivery/track/:orderId
	// actually pushes something (see PostDeliveryLocation doc comment).
	v1.Post("/delivery/location", jwtHandler, middleware.DeliveryOnly(), deliveryQueue.PostDeliveryLocation)
	// Device push-token registration — also not part of the generated
	// ServerInterface, same reason as the delivery routes above.
	v1.Patch("/auth/fcm-token", jwtHandler, deliveryQueue.PatchAuthFCMToken)

	// Payments v2 (docs/PAYMENTS_REDESIGN.md): immediate-capture + COD lifecycle.
	// Webhook is PUBLIC (provider callback, signature-verified, no JWT).
	v1.Post("/payments/webhook/:gateway", paymentV2Controller.Webhook)
	v1.Post("/payments", jwtHandler, paymentV2Controller.CreatePayment)
	v1.Get("/payments", jwtHandler, paymentV2Controller.ListPayments)
	v1.Get("/payments/:id", jwtHandler, paymentV2Controller.GetPayment)
	v1.Post("/payments/:id/refund", jwtHandler, paymentV2Controller.Refund)
	v1.Post("/payments/:id/cod/collect", jwtHandler, paymentV2Controller.CODCollect)
	v1.Post("/payments/:id/cod/reconcile", jwtHandler, paymentV2Controller.CODReconcile)
	v1.Get("/admin/reconciliation/config", jwtHandler, middleware.AdminOnly(), paymentV2Controller.GetReconConfig)
	v1.Put("/admin/reconciliation/config", jwtHandler, middleware.AdminOnly(), paymentV2Controller.UpdateReconConfig)
	v1.Get("/admin/reconciliation/runs", jwtHandler, middleware.AdminOnly(), paymentV2Controller.ListReconRuns)
	v1.Post("/admin/reconciliation/run", jwtHandler, middleware.AdminOnly(), paymentV2Controller.RunReconNow)
	v1.Get("/orders/:orderId/invoices", jwtHandler, paymentV2Controller.GetOrderInvoices)

	// Admin partner management: resume/reinstate, commission rate, real
	// per-partner performance, platform revenue trend, and platform-wide
	// unmet search demand — previously either missing entirely (resume,
	// commission, performance, revenue trend) or 404ing because the client
	// called an admin path with no matching route (search-demand/unmet).
	v1.Post("/admin/partners/:id/resume", jwtHandler, middleware.AdminOnly(), adminPartnerController.PostResume)
	v1.Get("/admin/partners/:id/performance", jwtHandler, middleware.AdminOnly(), adminPartnerController.GetPerformance)
	v1.Get("/admin/partners/:id/commission", jwtHandler, middleware.AdminOnly(), adminPartnerController.GetCommission)
	v1.Put("/admin/partners/:id/commission", jwtHandler, middleware.AdminOnly(), adminPartnerController.PutCommission)
	v1.Get("/admin/reports/timeseries", jwtHandler, middleware.AdminOnly(), adminPartnerController.GetRevenueTimeseries)
	v1.Get("/admin/search-demand/unmet", jwtHandler, middleware.AdminOnly(), analyticsController.AdminUnmetDemand)
	v1.Post("/admin/search-demand/unmet/notify", jwtHandler, middleware.AdminOnly(), analyticsController.AdminNotifyUnmetDemand)

	// Restaurant registration flow used by the Partner app.
	// Unified vendor onboarding (restaurant/grocery/delivery) — JWT-scoped.
	v1.Post("/vendor/register", jwtHandler, registrationController.RegisterVendor)
	v1.Get("/vendor/registration", jwtHandler, registrationController.GetVendorRegistration)
	// All of these previously had no auth at all — any caller could read or
	// overwrite any restaurant's registration/payment data by guessing
	// restaurant_id. The vendor app is already logged in by this step in
	// onboarding (DioClient attaches the JWT to every request), so requiring
	// it here costs nothing legitimate; the controllers now also verify the
	// caller owns the restaurant_id they're touching.
	v1.Post("/restaurant/register", jwtHandler, registrationController.RegisterRestaurant)
	v1.Put("/restaurant/:restaurant_id/payment", jwtHandler, registrationController.UpdatePaymentInfo)
	v1.Put("/restaurant/:restaurant_id/complete", jwtHandler, registrationController.CompleteRegistration)
	v1.Get("/restaurant/:restaurant_id/outlets", jwtHandler, registrationController.GetVendorOutlets)
	// Must be registered before /restaurant/:restaurant_id below — Fiber
	// matches routes in registration order, so "mine" would otherwise be
	// captured as a :restaurant_id path param instead of hitting this route.
	v1.Get("/restaurant/mine", jwtHandler, registrationController.GetMyRestaurant)
	v1.Get("/restaurant/:restaurant_id", jwtHandler, registrationController.GetRestaurantData)

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
		log.Info().Msg("Shutting down: draining in-flight requests")
		// Drain the server first (finish in-flight requests) before closing DB.
		if err := app.ShutdownWithTimeout(15 * time.Second); err != nil {
			log.Error().Err(err).Msg("Server shutdown error")
		}
		db.CloseDatabase()
		log.Info().Msg("Shutdown complete")
	}()

	// ─── Start Server ───────────────────────────────────────────────────
	log.Info().Str("port", cfg.AppPort).Msg("Starting Dailzo backend")
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

// invalidOrigins returns the comma-separated entries of CORS_ORIGINS that
// Fiber would reject.
//
// Fiber validates origins itself and panics when one is malformed —
// "panic: [CORS] Invalid origin format in configuration: none" with a stack
// trace, which says nothing about where the value came from or what a valid
// one looks like. A misconfigured deployment deserves a sentence, not a
// goroutine dump.
func invalidOrigins(list string) []string {
	var bad []string
	for _, raw := range strings.Split(list, ",") {
		origin := strings.TrimSpace(raw)
		if origin == "" {
			continue
		}
		u, err := url.Parse(origin)
		// An origin is scheme + host only: no path, no trailing slash, no
		// query. "example.com" and "https://example.com/" are both rejected
		// by Fiber, so reject them here with an explanation.
		if err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" || u.RawQuery != "" {
			bad = append(bad, origin)
		}
	}
	return bad
}

// healthCheck serves both the root /health probe and its /api/v1 alias.
func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "dailzo-backend",
	})
}
