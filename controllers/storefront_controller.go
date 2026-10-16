package controllers

import (
	"strconv"

	"dailzo/models"
	"dailzo/pkg/i18n"
	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// StorefrontController serves customer-facing storefront endpoints: catalog,
// server-side cart, checkout, order history, complaints and refunds.
type StorefrontController struct {
	repo    *repository.StorefrontRepository
	trans   *repository.TranslationRepository
	vendor  *repository.VendorSupportRepository
	billing *repository.BillingRepository
}

func NewStorefrontController(
	repo *repository.StorefrontRepository,
	trans *repository.TranslationRepository,
	vendor *repository.VendorSupportRepository,
	billing *repository.BillingRepository,
) *StorefrontController {
	return &StorefrontController{repo: repo, trans: trans, vendor: vendor, billing: billing}
}

// userID pulls the authenticated user id set by the JWT middleware.
func userID(ctx *fiber.Ctx) string {
	if v, ok := ctx.Locals("user_id").(string); ok {
		return v
	}
	return ""
}

// ─── Catalog (public) ─────────────────────────────────────────────────────────

// GET /catalog/categories?kind=food|grocery
func (c *StorefrontController) GetCategories(ctx *fiber.Ctx) error {
	kind := ctx.Query("kind", "food")
	cats, err := c.repo.GetCategories(ctx.Context(), kind)
	if err != nil {
		return response.InternalError(ctx, "could not fetch categories")
	}
	c.trans.LocalizeMaps(ctx.Context(), "category", i18n.ParseLang(ctx.Get("Accept-Language")), cats, "name", "description")
	return response.Success(ctx, cats)
}

// GET /catalog/products?category_id=&outlet_id=&restaurant_id=&q=
func (c *StorefrontController) GetProducts(ctx *fiber.Ctx) error {
	products, err := c.repo.GetProducts(ctx.Context(),
		ctx.Query("category_id"), ctx.Query("outlet_id"),
		ctx.Query("restaurant_id"), ctx.Query("q"))
	if err != nil {
		return response.InternalError(ctx, "could not fetch products")
	}
	c.trans.LocalizeMaps(ctx.Context(), "product", i18n.ParseLang(ctx.Get("Accept-Language")), products, "name", "description")
	return response.Success(ctx, products)
}

// GET /catalog/restaurants
func (c *StorefrontController) GetRestaurants(ctx *fiber.Ctx) error {
	restaurants, err := c.repo.GetRestaurants(ctx.Context())
	if err != nil {
		return response.InternalError(ctx, "could not fetch restaurants")
	}
	return response.Success(ctx, restaurants)
}

// ─── Cart (JWT) ────────────────────────────────────────────────────────────────

// GET /cart?type=food|grocery
func (c *StorefrontController) GetCart(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	items, err := c.repo.GetCart(ctx.Context(), uid, ctx.Query("type", "food"))
	if err != nil {
		return response.InternalError(ctx, "could not fetch cart")
	}
	return response.Success(ctx, items)
}

// POST /cart/items  { cart_type, product_id, variant_id?, quantity }
func (c *StorefrontController) AddCartItem(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	var body struct {
		CartType  string `json:"cart_type"`
		ProductID string `json:"product_id"`
		VariantID string `json:"variant_id"`
		Quantity  int    `json:"quantity"`
		// Chosen product_options ids. Only the ids travel: every price is
		// resolved server-side at checkout so a client can't set its own.
		SelectedOptionIDs []string `json:"selected_option_ids"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if body.ProductID == "" {
		return response.BadRequest(ctx, "product_id is required")
	}
	if body.CartType == "" {
		body.CartType = "food"
	}
	id, err := c.repo.AddCartItem(ctx.Context(), uid, body.CartType, body.ProductID,
		body.VariantID, body.Quantity, body.SelectedOptionIDs)
	if err != nil {
		return response.InternalError(ctx, "could not add to cart")
	}
	return response.Created(ctx, fiber.Map{"id": id})
}

// PATCH /cart/items/:id  { quantity }
func (c *StorefrontController) UpdateCartItem(ctx *fiber.Ctx) error {
	if userID(ctx) == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if err := c.repo.SetCartItemQty(ctx.Context(), ctx.Params("id"), body.Quantity); err != nil {
		return response.InternalError(ctx, "could not update cart item")
	}
	return response.SuccessWithMessage(ctx, "updated", nil)
}

// DELETE /cart/items/:id
func (c *StorefrontController) DeleteCartItem(ctx *fiber.Ctx) error {
	if userID(ctx) == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	if err := c.repo.DeleteCartItem(ctx.Context(), ctx.Params("id")); err != nil {
		return response.InternalError(ctx, "could not remove cart item")
	}
	return response.SuccessWithMessage(ctx, "removed", nil)
}

// DELETE /cart?type=food|grocery
func (c *StorefrontController) ClearCart(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	if err := c.repo.ClearCart(ctx.Context(), uid, ctx.Query("type", "food")); err != nil {
		return response.InternalError(ctx, "could not clear cart")
	}
	return response.SuccessWithMessage(ctx, "cleared", nil)
}

// ─── Orders (JWT) ────────────────────────────────────────────────────────────

// POST /orders  { cart_type, delivery_address, customer_name, payment_method, tip, delivery_fee, tax_amount, restaurant_id }
func (c *StorefrontController) CreateOrder(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	var meta map[string]any
	if err := ctx.BodyParser(&meta); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	order, err := c.repo.Checkout(ctx.Context(), uid, meta)
	if err != nil {
		if err.Error() == "cart is empty" {
			return response.BadRequest(ctx, "cart is empty")
		}
		return response.InternalError(ctx, "could not place order")
	}
	return response.Created(ctx, order)
}

// GET /my-orders
func (c *StorefrontController) GetMyOrders(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	orders, err := c.repo.GetMyOrders(ctx.Context(), uid)
	if err != nil {
		return response.InternalError(ctx, "could not fetch orders")
	}
	return response.Success(ctx, orders)
}

// GET /order/:id
func (c *StorefrontController) GetOrder(ctx *fiber.Ctx) error {
	order, err := c.repo.GetOrder(ctx.Context(), ctx.Params("id"))
	if err != nil {
		return response.NotFound(ctx, "order not found")
	}
	return response.Success(ctx, order)
}

// GET /vendor/orders?status=  — vendor view of incoming customer orders.
func (c *StorefrontController) GetVendorOrders(ctx *fiber.Ctx) error {
	// outlet_id (falling back to restaurant_id, same query param either app
	// might send) scopes the list to one outlet instead of every order in the
	// system — see the trust-boundary note on GetAllOrders.
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}

	outletID := ctx.Query("outlet_id")
	if outletID == "" {
		outletID = ctx.Query("restaurant_id")
	}

	// Falling back to "no filter" used to mean "every order on the
	// platform": any authenticated vendor who omitted the query parameter —
	// as the grocery dashboard always does — got every other vendor's orders.
	// The caller's own businesses and outlets are now the floor, and an
	// explicit outlet_id may only narrow within them.
	scope, err := c.vendor.GetVendorScope(ctx.Context(), uid)
	if err != nil {
		return response.InternalError(ctx, "could not resolve vendor")
	}
	if outletID != "" && !scope.Owns(outletID) {
		return response.Forbidden(ctx, "outlet does not belong to this account")
	}

	orders, err := c.repo.GetVendorOrders(ctx.Context(), ctx.Query("status"), outletID, scope)
	if err != nil {
		return response.InternalError(ctx, "could not fetch orders")
	}
	return response.Success(ctx, orders)
}

// PATCH /vendor/orders/:id/status  { status }
func (c *StorefrontController) UpdateVendorOrderStatus(ctx *fiber.Ctx) error {
	var body struct {
		Status string `json:"status"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if body.Status == "" {
		return response.BadRequest(ctx, "status is required")
	}
	if !models.IsValidOrderStatus(body.Status) {
		return response.BadRequest(ctx, "invalid status value")
	}
	if err := c.repo.UpdateOrderStatus(ctx.Context(), ctx.Params("id"), body.Status); err != nil {
		return response.InternalError(ctx, "could not update order status")
	}
	return response.SuccessWithMessage(ctx, "updated", nil)
}

// ─── Complaints (JWT) ────────────────────────────────────────────────────────

// POST /complaints  { order_id?, complaint_type, description, priority? }
func (c *StorefrontController) CreateComplaint(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	var body map[string]any
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if strVal(body["description"]) == "" {
		return response.BadRequest(ctx, "description is required")
	}
	created, err := c.repo.CreateComplaint(ctx.Context(), uid, body)
	if err != nil {
		return response.InternalError(ctx, "could not file complaint")
	}
	return response.Created(ctx, created)
}

// GET /my-complaints
func (c *StorefrontController) GetComplaints(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	list, err := c.repo.GetComplaints(ctx.Context(), uid)
	if err != nil {
		return response.InternalError(ctx, "could not fetch complaints")
	}
	return response.Success(ctx, list)
}

// ─── Refunds (JWT) ───────────────────────────────────────────────────────────

// POST /refunds  { payment_id?, amount, reason }
func (c *StorefrontController) CreateRefund(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	var body map[string]any
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	created, err := c.repo.CreateRefund(ctx.Context(), uid, body)
	if err != nil {
		return response.InternalError(ctx, "could not request refund")
	}
	return response.Created(ctx, created)
}

// GET /my-refunds
func (c *StorefrontController) GetRefunds(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}
	list, err := c.repo.GetRefunds(ctx.Context(), uid)
	if err != nil {
		return response.InternalError(ctx, "could not fetch refunds")
	}
	return response.Success(ctx, list)
}

// strVal mirrors the repository helper for controller-side validation.
func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GET /catalog/products/:id/options — customisation groups for one product.
// Public: the customisation sheet opens before a user is necessarily signed in.
func (c *StorefrontController) GetProductOptions(ctx *fiber.Ctx) error {
	productID := ctx.Params("id")
	if productID == "" {
		return response.BadRequest(ctx, "product id is required")
	}

	groups, err := c.repo.GetProductOptionGroups(ctx.Context(), productID)
	if err != nil {
		return response.InternalError(ctx, "could not fetch product options")
	}

	// An empty list is a valid answer: a product with nothing to customise.
	return response.Success(ctx, groups)
}

// GET /cart/quote?cart_type=food&region=SA&tip=0&discount=0
//
// The fully priced bill for the signed-in user's cart. The customer app shows
// exactly these lines instead of computing fees and VAT itself.
func (c *StorefrontController) GetCartQuote(ctx *fiber.Ctx) error {
	uid := userID(ctx)
	if uid == "" {
		return response.Unauthorized(ctx, "authentication required")
	}

	cartType := ctx.Query("cart_type")
	if cartType == "" {
		cartType = "food"
	}
	tip, _ := strconv.ParseFloat(ctx.Query("tip", "0"), 64)
	discount, _ := strconv.ParseFloat(ctx.Query("discount", "0"), 64)

	quote, err := c.repo.QuoteCart(ctx.Context(), uid, cartType, ctx.Query("region"), tip, discount)
	if err != nil {
		return response.InternalError(ctx, "could not price cart")
	}
	return response.Success(ctx, quote)
}

// GET /config/billing?region=SA — the market's fee schedule.
//
// Public: the app shows delivery/minimum-order messaging before sign-in.
func (c *StorefrontController) GetBillingConfig(ctx *fiber.Ctx) error {
	cfg, err := c.billing.GetBillingConfig(ctx.Context(), ctx.Query("region"))
	if err != nil {
		return response.InternalError(ctx, "could not load billing config")
	}
	return response.Success(ctx, cfg)
}

// GET /admin/billing-config — every market's fee schedule, for the admin editor.
func (c *StorefrontController) ListBillingConfigs(ctx *fiber.Ctx) error {
	configs, err := c.billing.ListBillingConfigs(ctx.Context())
	if err != nil {
		return response.InternalError(ctx, "could not list billing configs")
	}
	return response.Success(ctx, configs)
}

// PUT /admin/billing-config/:region — change what customers are charged.
//
// Admin-only (enforced by the /admin/ prefix check in the v1 auth middleware).
// Values are validated in the repository, since they decide real money.
func (c *StorefrontController) UpdateBillingConfig(ctx *fiber.Ctx) error {
	region := ctx.Params("region")
	if region == "" {
		return response.BadRequest(ctx, "region is required")
	}

	var body repository.BillingConfig
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	body.RegionCode = region

	if err := c.billing.UpsertBillingConfig(ctx.Context(), body, userID(ctx)); err != nil {
		return response.BadRequest(ctx, err.Error())
	}

	saved, err := c.billing.GetBillingConfig(ctx.Context(), region)
	if err != nil {
		return response.InternalError(ctx, "saved, but could not read it back")
	}
	return response.SuccessWithMessage(ctx, "billing config updated", saved)
}

// POST /cart/quote — price the lines the client is showing.
//
// Public: the cart is usable before sign-in, and this reveals no private data
// (it prices products the catalog already exposes). Every price still comes
// from the database and the billing policy; the body only says which products
// and how many.
func (c *StorefrontController) PostCartQuote(ctx *fiber.Ctx) error {
	var body struct {
		CartType string                 `json:"cart_type"`
		Region   string                 `json:"region"`
		Tip      float64                `json:"tip"`
		Discount float64                `json:"discount"`
		Lines    []repository.QuoteLine `json:"lines"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return response.BadRequest(ctx, "invalid input")
	}
	if body.CartType == "" {
		body.CartType = "food"
	}

	quote, err := c.repo.QuoteLines(ctx.Context(), body.CartType, body.Region,
		body.Tip, body.Discount, body.Lines)
	if err != nil {
		return response.InternalError(ctx, "could not price cart")
	}
	return response.Success(ctx, quote)
}
