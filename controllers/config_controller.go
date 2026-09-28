package controllers

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"dailzo/pkg/i18n"
	"dailzo/pkg/response"
	"dailzo/repository"

	"github.com/gofiber/fiber/v2"
)

// ConfigController serves the admin-configurable surfaces: enum/option lists
// and home/dashboard content sections. Public GETs localize via
// Accept-Language; mutations are admin-only (mounted behind the JWT group).
type ConfigController struct {
	repo *repository.ConfigRepository
}

func NewConfigController(repo *repository.ConfigRepository) *ConfigController {
	return &ConfigController{repo: repo}
}

// lang resolves the response language from the Accept-Language header.
func lang(ctx *fiber.Ctx) string { return i18n.ParseLang(ctx.Get("Accept-Language")) }

// slugID builds a stable id from parts (used when the client omits one).
func slugID(prefix string, parts ...string) string {
	sum := md5.Sum([]byte(strings.Join(parts, "|")))
	return prefix + "_" + hex.EncodeToString(sum[:])[:16]
}

// ─── Options ────────────────────────────────────────────────────────────────

// GET /config/options?kind=cuisine  (public, localized)
func (c *ConfigController) GetOptions(ctx *fiber.Ctx) error {
	kind := ctx.Query("kind")
	if kind == "" {
		return response.BadRequest(ctx, "kind is required")
	}
	items, err := c.repo.ListOptions(ctx.Context(), kind, lang(ctx))
	if err != nil {
		return response.InternalError(ctx, "could not fetch options")
	}
	if items == nil {
		items = []repository.OptionItem{}
	}
	return response.Success(ctx, items)
}

// POST /config/options  (admin) — create/update one option, auto-translates label
func (c *ConfigController) UpsertOption(ctx *fiber.Ctx) error {
	var in repository.OptionItem
	if err := ctx.BodyParser(&in); err != nil {
		return response.BadRequest(ctx, "invalid body")
	}
	if in.SetKind == "" || in.Code == "" || in.Label == "" {
		return response.BadRequest(ctx, "set_kind, code and label are required")
	}
	if in.ID == "" {
		in.ID = slugID("opt", in.SetKind, in.Code)
	}
	if err := c.repo.UpsertOption(ctx.Context(), in); err != nil {
		return response.InternalError(ctx, "could not save option")
	}
	return response.Success(ctx, in)
}

// DELETE /config/options/:id  (admin)
func (c *ConfigController) DeleteOption(ctx *fiber.Ctx) error {
	if err := c.repo.DeleteOption(ctx.Context(), ctx.Params("id")); err != nil {
		return response.InternalError(ctx, "could not delete option")
	}
	return response.Success(ctx, fiber.Map{"deleted": true})
}

// ─── Content sections ───────────────────────────────────────────────────────

// GET /config/sections?surface=customer_home&city=  (public, localized)
func (c *ConfigController) GetSections(ctx *fiber.Ctx) error {
	surface := ctx.Query("surface")
	if surface == "" {
		return response.BadRequest(ctx, "surface is required")
	}
	secs, err := c.repo.ListSections(ctx.Context(), surface, ctx.Query("city"), lang(ctx))
	if err != nil {
		return response.InternalError(ctx, "could not fetch sections")
	}
	if secs == nil {
		secs = []repository.ContentSection{}
	}
	return response.Success(ctx, secs)
}

// POST /config/sections  (admin) — create/update one section, auto-translates title/subtitle
func (c *ConfigController) UpsertSection(ctx *fiber.Ctx) error {
	var in repository.ContentSection
	if err := ctx.BodyParser(&in); err != nil {
		return response.BadRequest(ctx, "invalid body")
	}
	if in.Surface == "" || in.SectionKey == "" || in.Type == "" {
		return response.BadRequest(ctx, "surface, section_key and type are required")
	}
	if in.ID == "" {
		in.ID = slugID("sec", in.Surface, in.SectionKey)
	}
	if err := c.repo.UpsertSection(ctx.Context(), in); err != nil {
		return response.InternalError(ctx, "could not save section")
	}
	return response.Success(ctx, in)
}

// DELETE /config/sections/:id  (admin)
func (c *ConfigController) DeleteSection(ctx *fiber.Ctx) error {
	if err := c.repo.DeleteSection(ctx.Context(), ctx.Params("id")); err != nil {
		return response.InternalError(ctx, "could not delete section")
	}
	return response.Success(ctx, fiber.Map{"deleted": true})
}

// ─── Manual translation override (admin/vendor) ─────────────────────────────

// PUT /config/translations  (admin) — set a reviewed translation that
// auto-translation will never overwrite. Body: {entity_type, entity_id, field, lang, value}.
func (c *ConfigController) SetTranslation(ctx *fiber.Ctx) error {
	var in struct {
		EntityType string `json:"entity_type"`
		EntityID   string `json:"entity_id"`
		Field      string `json:"field"`
		Lang       string `json:"lang"`
		Value      string `json:"value"`
	}
	if err := ctx.BodyParser(&in); err != nil {
		return response.BadRequest(ctx, "invalid body")
	}
	if in.EntityType == "" || in.EntityID == "" || in.Field == "" || in.Lang == "" {
		return response.BadRequest(ctx, "entity_type, entity_id, field and lang are required")
	}
	if err := c.repo.SetManualTranslation(ctx.Context(), in.EntityType, in.EntityID, in.Field, in.Lang, in.Value); err != nil {
		return response.InternalError(ctx, "could not save translation")
	}
	return response.Success(ctx, in)
}

// ─── Categories (admin CRUD; public localized read lives on the storefront) ──

// POST /config/categories  (admin) — create/update, auto-translates name+description
func (c *ConfigController) UpsertCategory(ctx *fiber.Ctx) error {
	var in repository.Category
	if err := ctx.BodyParser(&in); err != nil {
		return response.BadRequest(ctx, "invalid body")
	}
	if in.Name == "" || in.Kind == "" {
		return response.BadRequest(ctx, "name and kind are required")
	}
	if in.ID == "" {
		in.ID = slugID("cat", in.Kind, in.Name)
	}
	if err := c.repo.UpsertCategory(ctx.Context(), in); err != nil {
		return response.InternalError(ctx, "could not save category")
	}
	return response.Success(ctx, in)
}

// DELETE /config/categories/:id  (admin) — soft delete
func (c *ConfigController) DeleteCategory(ctx *fiber.Ctx) error {
	if err := c.repo.DeleteCategory(ctx.Context(), ctx.Params("id")); err != nil {
		return response.InternalError(ctx, "could not delete category")
	}
	return response.Success(ctx, fiber.Map{"deleted": true})
}
