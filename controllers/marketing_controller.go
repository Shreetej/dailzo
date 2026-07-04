package controllers

import (
	"dailzo/models"
	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

type MarketingController struct {
	repo *repository.MarketingRepository
}

func NewMarketingController(repo *repository.MarketingRepository) *MarketingController {
	return &MarketingController{repo: repo}
}

// GET /marketing/discounts
func (c *MarketingController) GetDiscounts(ctx *fiber.Ctx) error {
	discounts, err := c.repo.GetDiscounts(ctx.Context())
	if err != nil {
		return response.InternalError(ctx, "could not fetch discounts")
	}
	return response.Success(ctx, discounts)
}

// POST /marketing/discounts
func (c *MarketingController) CreateDiscount(ctx *fiber.Ctx) error {
	var discount models.DiscountCampaign
	if err := ctx.BodyParser(&discount); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if discount.Code == "" || discount.Type == "" {
		return response.BadRequest(ctx, "code and type are required")
	}

	created, err := c.repo.CreateDiscount(ctx.Context(), discount)
	if err != nil {
		return response.InternalError(ctx, "could not create discount")
	}
	return response.Created(ctx, created)
}

// GET /marketing/ads
func (c *MarketingController) GetAdCampaigns(ctx *fiber.Ctx) error {
	ads, err := c.repo.GetAdCampaigns(ctx.Context())
	if err != nil {
		return response.InternalError(ctx, "could not fetch ad campaigns")
	}
	return response.Success(ctx, ads)
}

// POST /marketing/ads
func (c *MarketingController) CreateAdCampaign(ctx *fiber.Ctx) error {
	var campaign models.AdCampaign
	if err := ctx.BodyParser(&campaign); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if campaign.Name == "" || campaign.Kind == "" {
		return response.BadRequest(ctx, "name and kind are required")
	}

	created, err := c.repo.CreateAdCampaign(ctx.Context(), campaign)
	if err != nil {
		return response.InternalError(ctx, "could not create ad campaign")
	}
	return response.Created(ctx, created)
}

// POST /marketing/ads/:id/stop
func (c *MarketingController) StopAdCampaign(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return response.BadRequest(ctx, "campaign id is required")
	}
	if err := c.repo.StopAdCampaign(ctx.Context(), id); err != nil {
		return response.NotFound(ctx, "ad campaign not found")
	}
	return response.SuccessWithMessage(ctx, "Ad campaign stopped", nil)
}

// GET /marketing/ads/packs
func (c *MarketingController) GetAdPacks(ctx *fiber.Ctx) error {
	packs, err := c.repo.GetAdPacks(ctx.Context())
	if err != nil {
		return response.InternalError(ctx, "could not fetch ad packs")
	}
	return response.Success(ctx, packs)
}

// POST /orders/:order_id/out-of-stock
func (c *MarketingController) MarkOrderItemsOutOfStock(ctx *fiber.Ctx) error {
	orderID := ctx.Params("order_id")
	if orderID == "" {
		return response.BadRequest(ctx, "order id is required")
	}

	var body struct {
		ProductIDs []string `json:"product_ids"`
	}
	if err := ctx.BodyParser(&body); err != nil || len(body.ProductIDs) == 0 {
		return response.BadRequest(ctx, "product_ids is required")
	}

	if err := c.repo.MarkOrderItemsOutOfStock(ctx.Context(), orderID, body.ProductIDs); err != nil {
		return response.InternalError(ctx, "could not mark items out of stock")
	}
	return response.SuccessWithMessage(ctx, "Items marked out of stock; customer informed", nil)
}

// GET /orders/:order_id/out-of-stock
func (c *MarketingController) GetOrderOutOfStockItems(ctx *fiber.Ctx) error {
	orderID := ctx.Params("order_id")
	productIDs, err := c.repo.GetOrderOutOfStockItems(ctx.Context(), orderID)
	if err != nil {
		return response.InternalError(ctx, "could not fetch out-of-stock items")
	}
	return response.Success(ctx, fiber.Map{"order_id": orderID, "product_ids": productIDs})
}

// POST /restaurant/:restaurant_id/outlets
func (c *MarketingController) CreateVendorOutlet(ctx *fiber.Ctx) error {
	restaurantID := ctx.Params("restaurant_id")
	if restaurantID == "" {
		return response.BadRequest(ctx, "restaurant id is required")
	}

	var outlet models.VendorOutlet
	if err := ctx.BodyParser(&outlet); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	outlet.RestaurantID = restaurantID
	outlet.IsActive = true

	created, err := c.repo.CreateVendorOutlet(ctx.Context(), outlet)
	if err != nil {
		return response.InternalError(ctx, "could not create outlet")
	}
	return response.Created(ctx, created)
}
