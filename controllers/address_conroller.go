package controllers

import (
	"dailzo/config"
	"dailzo/globals"
	"dailzo/models"
	"dailzo/repository"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type AddressController struct {
	repo *repository.AddressRepository
}

func NewAddressController(repo *repository.AddressRepository) *AddressController {
	return &AddressController{repo: repo}
}

func (c *AddressController) CreateAddress(ctx *fiber.Ctx) error {
	var addr models.Address
	if err := ctx.BodyParser(&addr); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	globals.UpdateUserID(ctx.Locals("user_id").(string))
	fmt.Print("User details:", ctx.Locals("user_id"))
	id, err := c.repo.CreateAddress(ctx.Context(), addr)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create user"})
	}

	// Log user creation
	log := config.SetupLogger()
	log.Info().Msgf("Address created with ID: %d", id)

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}
