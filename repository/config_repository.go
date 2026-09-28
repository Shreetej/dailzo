package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigRepository powers the admin-configurable surfaces: content_sections
// (home/dashboard blocks) and option_items (enum/dropdown lists). Reads are
// localized via TranslationRepository by (entity_type, field, lang).
type ConfigRepository struct {
	db    *pgxpool.Pool
	trans *TranslationRepository
}

func NewConfigRepository(db *pgxpool.Pool, trans *TranslationRepository) *ConfigRepository {
	return &ConfigRepository{db: db, trans: trans}
}

// SetManualTranslation stores a reviewed human translation (override) that
// auto-translation will never clobber.
func (r *ConfigRepository) SetManualTranslation(ctx context.Context, entityType, entityID, field, lang, value string) error {
	return r.trans.SetManual(ctx, entityType, entityID, field, lang, value)
}

// ─── Options (cuisine, outlet_type, vehicle, shift, menu_type) ──────────────

type OptionItem struct {
	ID        string `json:"id"`
	SetKind   string `json:"set_kind"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

// ListOptions returns active items for a set, with labels localized to lang.
func (r *ConfigRepository) ListOptions(ctx context.Context, setKind, lang string) ([]OptionItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, set_kind, code, label, sort_order, is_active
		FROM option_items WHERE set_kind=$1 AND is_active=true
		ORDER BY sort_order, label`, setKind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []OptionItem
	var ids []string
	for rows.Next() {
		var it OptionItem
		if err := rows.Scan(&it.ID, &it.SetKind, &it.Code, &it.Label, &it.SortOrder, &it.IsActive); err != nil {
			return nil, err
		}
		items = append(items, it)
		ids = append(ids, it.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	loc, _ := r.trans.Localize(ctx, "option", "label", lang, ids)
	for i := range items {
		if v, ok := loc[items[i].ID]; ok {
			items[i].Label = v
		}
	}
	return items, nil
}

// UpsertOption creates or updates an option and auto-translates its label.
func (r *ConfigRepository) UpsertOption(ctx context.Context, it OptionItem) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO option_items (id, set_kind, code, label, sort_order, is_active)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (set_kind, code) DO UPDATE
		SET label=EXCLUDED.label, sort_order=EXCLUDED.sort_order, is_active=EXCLUDED.is_active`,
		it.ID, it.SetKind, it.Code, it.Label, it.SortOrder, it.IsActive)
	if err != nil {
		return err
	}
	r.trans.AutoTranslate(ctx, "option", it.ID, "label", it.Label)
	return nil
}

func (r *ConfigRepository) DeleteOption(ctx context.Context, id string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM option_items WHERE id=$1`, id); err != nil {
		return err
	}
	return r.trans.Delete(ctx, "option", id)
}

// ─── Content sections (home/dashboard blocks) ───────────────────────────────

type ContentSection struct {
	ID         string          `json:"id"`
	Surface    string          `json:"surface"`
	SectionKey string          `json:"section_key"`
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Subtitle   string          `json:"subtitle"`
	Payload    json.RawMessage `json:"payload"`
	City       *string         `json:"city"`
	SortOrder  int             `json:"sort_order"`
	IsActive   bool            `json:"is_active"`
}

// ListSections returns active sections for a surface (optionally city-scoped:
// rows with a NULL city always apply; a non-empty city also returns that
// city's rows), with title/subtitle localized to lang.
func (r *ConfigRepository) ListSections(ctx context.Context, surface, city, lang string) ([]ContentSection, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, surface, section_key, type, COALESCE(title,''), COALESCE(subtitle,''),
		       COALESCE(payload,'{}'::jsonb), city, sort_order, is_active
		FROM content_sections
		WHERE surface=$1 AND is_active=true AND (city IS NULL OR city=$2)
		ORDER BY sort_order, section_key`, surface, city)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var secs []ContentSection
	var ids []string
	for rows.Next() {
		var s ContentSection
		var payload []byte
		if err := rows.Scan(&s.ID, &s.Surface, &s.SectionKey, &s.Type, &s.Title, &s.Subtitle,
			&payload, &s.City, &s.SortOrder, &s.IsActive); err != nil {
			return nil, err
		}
		s.Payload = json.RawMessage(payload)
		secs = append(secs, s)
		ids = append(ids, s.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	titles, _ := r.trans.Localize(ctx, "section", "title", lang, ids)
	subs, _ := r.trans.Localize(ctx, "section", "subtitle", lang, ids)
	for i := range secs {
		if v, ok := titles[secs[i].ID]; ok {
			secs[i].Title = v
		}
		if v, ok := subs[secs[i].ID]; ok {
			secs[i].Subtitle = v
		}
	}
	return secs, nil
}

// UpsertSection creates or updates a section and auto-translates title+subtitle.
func (r *ConfigRepository) UpsertSection(ctx context.Context, s ContentSection) error {
	payload := s.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO content_sections (id, surface, section_key, type, title, subtitle, payload, city, sort_order, is_active, last_updated_on)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,CURRENT_TIMESTAMP)
		ON CONFLICT (surface, section_key) DO UPDATE
		SET type=EXCLUDED.type, title=EXCLUDED.title, subtitle=EXCLUDED.subtitle,
		    payload=EXCLUDED.payload, city=EXCLUDED.city, sort_order=EXCLUDED.sort_order,
		    is_active=EXCLUDED.is_active, last_updated_on=CURRENT_TIMESTAMP`,
		s.ID, s.Surface, s.SectionKey, s.Type, s.Title, s.Subtitle, []byte(payload), s.City, s.SortOrder, s.IsActive)
	if err != nil {
		return err
	}
	r.trans.AutoTranslate(ctx, "section", s.ID, "title", s.Title)
	r.trans.AutoTranslate(ctx, "section", s.ID, "subtitle", s.Subtitle)
	return nil
}

func (r *ConfigRepository) DeleteSection(ctx context.Context, id string) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM content_sections WHERE id=$1`, id); err != nil {
		return err
	}
	return r.trans.Delete(ctx, "section", id)
}

// ─── Catalog categories (admin CRUD; menu sections & grocery aisles) ─────────

type Category struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Kind         string  `json:"kind"`         // 'food' | 'grocery'
	ParentID     *string `json:"parent_id"`
	ImageURL     *string `json:"image_url"`
	Description  *string `json:"description"`
	SortOrder    int     `json:"sort_order"`
	RestaurantID *string `json:"restaurant_id"`
	OutletID     *string `json:"outlet_id"`
	IsActive     bool    `json:"is_active"`
	// Department groups top-level grocery categories into storefront sections
	// (e.g. "Grocery & Kitchen", "Snacks & Drinks", "Beauty & Personal Care").
	// Unset for food categories and for grocery subcategories.
	Department *string `json:"department"`
}

// UpsertCategory creates or updates a category and auto-translates name+description.
func (r *ConfigRepository) UpsertCategory(ctx context.Context, c Category) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO categories (id, name, kind, parent_id, image_url, description, sort_order,
		                        restaurant_id, outlet_id, is_active, department, last_updated_on)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE
		SET name=EXCLUDED.name, kind=EXCLUDED.kind, parent_id=EXCLUDED.parent_id,
		    image_url=EXCLUDED.image_url, description=EXCLUDED.description,
		    sort_order=EXCLUDED.sort_order, restaurant_id=EXCLUDED.restaurant_id,
		    outlet_id=EXCLUDED.outlet_id, is_active=EXCLUDED.is_active,
		    department=EXCLUDED.department,
		    last_updated_on=CURRENT_TIMESTAMP`,
		c.ID, c.Name, c.Kind, c.ParentID, c.ImageURL, c.Description, c.SortOrder,
		c.RestaurantID, c.OutletID, c.IsActive, c.Department)
	if err != nil {
		return err
	}
	r.trans.AutoTranslate(ctx, "category", c.ID, "name", c.Name)
	if c.Description != nil {
		r.trans.AutoTranslate(ctx, "category", c.ID, "description", *c.Description)
	}
	return nil
}

// DeleteCategory soft-deletes (is_active=false) so existing products keep their
// FK; hard delete would cascade. Translations are left in place for restore.
func (r *ConfigRepository) DeleteCategory(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE categories SET is_active=false, last_updated_on=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}
