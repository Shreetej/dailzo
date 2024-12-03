package controllers

import (
	"dailzo/config"
	"dailzo/models"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type AddressController struct {
	repo *repository.AddressRepository
}

func NewAddressController(repo *repository.AddressRepository) *AddressController {
	return &AddressController{repo: repo}
}

func (c *AddressController) CreateAddress(ctx *fiber.Ctx) error {
	var address models.Address
	if err := ctx.BodyParser(&address); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}
	id, err := c.repo.CreateAddress(ctx.Context(), address)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create address"})
	}
	log := config.SetupLogger()
	log.Info().Msgf("Address created with ID: %d", id)
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (c *AddressController) GetAddress(ctx *fiber.Ctx) error {
	addressID := ctx.Params("id")
	address, err := c.repo.GetAddressByID(ctx.Context(), addressID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Address not found"})
	}
	return ctx.JSON(address)
}

func (c *AddressController) GetAddresses(ctx *fiber.Ctx) error {
	addresses, err := c.repo.GetAddresses(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No addresses found"})
	}
	return ctx.JSON(addresses)
}

func (c *AddressController) UpdateAddress(ctx *fiber.Ctx) error {
	var address models.Address
	if err := ctx.BodyParser(&address); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid input"})
	}

	if err := c.repo.UpdateAddress(ctx.Context(), address); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update address"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Address updated successfully"})
}

func (c *AddressController) DeleteAddress(ctx *fiber.Ctx) error {
	addressID := ctx.Params("id")
	if err := c.repo.DeleteAddress(ctx.Context(), addressID); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete address"})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Address deleted successfully"})
}
