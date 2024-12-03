package routes

import (
	"dailzo/controllers"
	"dailzo/middleware"

	"github.com/gofiber/fiber/v2"
)

// func SetupRoutes(app *fiber.App, userController *controllers.UserController, addressController *controllers.AddressController, foodProductController *controllers.FoodProductController, productVariantController *controllers.ProductVariantController) {
// 	api := app.Group("/api")

// 	// Public routes
// 	api.Post("/users", userController.CreateUser)
// 	api.Post("/login", userController.Login)

// 	// Protected routes (JWT required)
// 	api.Get("/users/:id", middleware.JWTMiddleware(), userController.GetUserById)
// 	api.Put("/users", middleware.JWTMiddleware(), userController.UpdateUser)
// 	api.Delete("/users/:id", middleware.JWTMiddleware(), userController.DeleteUser)
// 	api.Get("/users", middleware.JWTMiddleware(), userController.GetUsers)
// 	api.Post("/address", middleware.JWTMiddleware(), addressController.CreateAddress)

// 	api.Post("/foodproduct", middleware.JWTMiddleware(), foodProductController.CreateFoodProduct)
// 	fmt.Print("User details:")

// 	api.Post("/productvariant", middleware.JWTMiddleware(), productVariantController.CreateProductVariant)

// }

func SetupRoutes(
	app *fiber.App,
	userController *controllers.UserController,
	addressController *controllers.AddressController,
	foodProductController *controllers.FoodProductController,
	productVariantController *controllers.ProductVariantController,
	paymentController *controllers.PaymentController,
	orderController *controllers.OrderController,
	orderItemController *controllers.OrderItemController,
	paymentMethodController *controllers.PaymentMethodController,
	ratingController *controllers.RatingController,
	refundController *controllers.RefundController,
	restaurantController *controllers.RestaurantController,
) {
	api := app.Group("/api")

	// Public routes
	api.Post("/users", userController.CreateUser)
	api.Post("/login", userController.Login)

	// Protected routes (JWT required)
	api.Get("/users/:id", middleware.JWTMiddleware(), userController.GetUserById)
	api.Put("/users", middleware.JWTMiddleware(), userController.UpdateUser)
	api.Delete("/users/:id", middleware.JWTMiddleware(), userController.DeleteUser)
	api.Get("/users", middleware.JWTMiddleware(), userController.GetUsers)

	api.Post("/address", middleware.JWTMiddleware(), addressController.CreateAddress)
	api.Get("/address/:id", middleware.JWTMiddleware(), addressController.GetAddressById)
	api.Put("/address/:id", middleware.JWTMiddleware(), addressController.UpdateAddress)
	api.Delete("/address/:id", middleware.JWTMiddleware(), addressController.DeleteAddress)

	api.Post("/foodproduct", middleware.JWTMiddleware(), foodProductController.CreateFoodProduct)
	api.Get("/foodproduct/:id", middleware.JWTMiddleware(), foodProductController.GetFoodProductById)
	api.Put("/foodproduct/:id", middleware.JWTMiddleware(), foodProductController.UpdateFoodProduct)
	api.Delete("/foodproduct/:id", middleware.JWTMiddleware(), foodProductController.DeleteFoodProduct)
	api.Get("/foodproducts", middleware.JWTMiddleware(), foodProductController.GetFoodProducts)

	api.Post("/productvariant", middleware.JWTMiddleware(), productVariantController.CreateProductVariant)
	api.Get("/productvariant/:id", middleware.JWTMiddleware(), productVariantController.GetProductVariantById)
	api.Put("/productvariant/:id", middleware.JWTMiddleware(), productVariantController.UpdateProductVariant)
	api.Delete("/productvariant/:id", middleware.JWTMiddleware(), productVariantController.DeleteProductVariant)
	api.Get("/productvariants", middleware.JWTMiddleware(), productVariantController.GetProductVariants)

	// Payment routes
	api.Post("/payment", middleware.JWTMiddleware(), paymentController.CreatePayment)
	api.Get("/payment/:id", middleware.JWTMiddleware(), paymentController.GetPayment)
	api.Put("/payment/:id", middleware.JWTMiddleware(), paymentController.UpdatePayment)
	api.Delete("/payment/:id", middleware.JWTMiddleware(), paymentController.DeletePayment)
	api.Get("/payments", middleware.JWTMiddleware(), paymentController.GetPayments)
	// api.Get("/payments/user/:userId", middleware.JWTMiddleware(), paymentController.GetPaymentsByUserId)
	// api.Get("/payments/order/:orderId", middleware.JWTMiddleware(), paymentController.GetPaymentsByOrderId)
	// api.Get("/payments/status/:status", middleware.JWTMiddleware(), paymentController.GetPaymentsByStatus)
	// api.Get("/payments/count", middleware.JWTMiddleware(), paymentController.CountPayments)

	// Order routes
	api.Post("/order", middleware.JWTMiddleware(), orderController.CreateOrder)
	api.Get("/order/:id", middleware.JWTMiddleware(), orderController.GetOrder)
	api.Put("/order/:id", middleware.JWTMiddleware(), orderController.UpdateOrder)
	api.Delete("/order/:id", middleware.JWTMiddleware(), orderController.DeleteOrder)
	api.Get("/orders", middleware.JWTMiddleware(), orderController.GetOrders)

	// OrderItem routes
	api.Post("/orderitem", middleware.JWTMiddleware(), orderItemController.CreateOrderItem)
	api.Get("/orderitem/:id", middleware.JWTMiddleware(), orderItemController.GetOrderItem)
	api.Put("/orderitem/:id", middleware.JWTMiddleware(), orderItemController.UpdateOrderItem)
	api.Delete("/orderitem/:id", middleware.JWTMiddleware(), orderItemController.DeleteOrderItem)
	api.Get("/orderitems", middleware.JWTMiddleware(), orderItemController.GetOrderItems)

	// PaymentMethod routes
	api.Post("/paymethod", middleware.JWTMiddleware(), paymentMethodController.CreatePaymentMethod)
	api.Get("/paymethod/:id", middleware.JWTMiddleware(), paymentMethodController.GetPaymentMethod)
	api.Put("/paymethod/:id", middleware.JWTMiddleware(), paymentMethodController.UpdatePaymentMethod)
	api.Delete("/paymethod/:id", middleware.JWTMiddleware(), paymentMethodController.DeletePaymentMethod)
	api.Get("/paymethods", middleware.JWTMiddleware(), paymentMethodController.GetPaymentMethods)

	// Rating routes
	api.Post("/rating", middleware.JWTMiddleware(), ratingController.CreateRating)
	api.Get("/rating/:id", middleware.JWTMiddleware(), ratingController.GetRating)
	api.Put("/rating/:id", middleware.JWTMiddleware(), ratingController.UpdateRating)
	api.Delete("/rating/:id", middleware.JWTMiddleware(), ratingController.DeleteRating)
	api.Get("/ratings", middleware.JWTMiddleware(), ratingController.GetRatings)
	// api.Get("/ratings/restaurant/:restaurantId", middleware.JWTMiddleware(), ratingController.GetRatingsByRestaurant)
	// api.Get("/ratings/user/:userId", middleware.JWTMiddleware(), ratingController.GetRatingsByUser)

	// Refund routes
	api.Post("/refund", middleware.JWTMiddleware(), refundController.CreateRefund)
	api.Get("/refund/:id", middleware.JWTMiddleware(), refundController.GetRefund)
	api.Put("/refund/:id", middleware.JWTMiddleware(), refundController.UpdateRefund)
	api.Delete("/refund/:id", middleware.JWTMiddleware(), refundController.DeleteRefund)
	api.Get("/refunds", middleware.JWTMiddleware(), refundController.GetRefunds)
	//api.Get("/refunds/order/:orderId", middleware.JWTMiddleware(), refundController.GetRefundsByOrderId)

	// Restaurant routes
	api.Post("/restaurant", middleware.JWTMiddleware(), restaurantController.CreateRestaurant)
	api.Get("/restaurant/:id", middleware.JWTMiddleware(), restaurantController.GetRestaurant)
	api.Put("/restaurant/:id", middleware.JWTMiddleware(), restaurantController.UpdateRestaurant)
	api.Delete("/restaurant/:id", middleware.JWTMiddleware(), restaurantController.DeleteRestaurant)
	api.Get("/restaurants", middleware.JWTMiddleware(), restaurantController.GetRestaurants)
	// api.Get("/restaurants/city/:city", middleware.JWTMiddleware(), restaurantController.GetRestaurantsByCity)
	// api.Get("/restaurants/cuisine/:cuisine", middleware.JWTMiddleware(), restaurantController.GetRestaurantsByCuisine)
}
