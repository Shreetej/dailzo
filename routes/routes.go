package routes

import (
	"dailzo/controllers"
	"dailzo/middleware"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, userController *controllers.UserController, addressController *controllers.AddressController, foodProductController *controllers.FoodProductController, productVariantController *controllers.ProductVariantController) {
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

	api.Post("/foodproduct", middleware.JWTMiddleware(), foodProductController.CreateFoodProduct)
	fmt.Print("User details:")

	api.Post("/productvariant", middleware.JWTMiddleware(), productVariantController.CreateProductVariant)

}
