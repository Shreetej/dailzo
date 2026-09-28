package repository

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"dailzo/pkg/i18n"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TranslationRepository owns content_translations: the generic per-(entity,
// field, lang) store that backs translate-on-write and Accept-Language reads.
type TranslationRepository struct {
	db *pgxpool.Pool
	tr i18n.Translator
}

func NewTranslationRepository(db *pgxpool.Pool, tr i18n.Translator) *TranslationRepository {
	return &TranslationRepository{db: db, tr: tr}
}

// ctrID is the deterministic PK for a translation row so upserts are stable.
func ctrID(entityType, entityID, field, lang string) string {
	sum := md5.Sum([]byte(entityType + "|" + entityID + "|" + field + "|" + lang))
	return "ctr_" + hex.EncodeToString(sum[:])[:24]
}

// AutoTranslate fills every target language for one field with source='auto'.
// It never overwrites a manual translation (the ON CONFLICT clause updates only
// rows whose existing source is 'auto'). Best-effort: a provider error for one
// language is swallowed so a write is never blocked on translation.
func (r *TranslationRepository) AutoTranslate(ctx context.Context, entityType, entityID, field, baseText string) {
	if baseText == "" {
		return
	}
	for _, lang := range i18n.Targets {
		translated, err := r.tr.Translate(ctx, baseText, i18n.BaseLang, lang)
		if err != nil || translated == "" {
			continue
		}
		_, _ = r.db.Exec(ctx, `
			INSERT INTO content_translations (id, entity_type, entity_id, field, lang, value, source, updated_on)
			VALUES ($1,$2,$3,$4,$5,$6,'auto',CURRENT_TIMESTAMP)
			ON CONFLICT (entity_type, entity_id, field, lang)
			DO UPDATE SET value = EXCLUDED.value, updated_on = CURRENT_TIMESTAMP
			WHERE content_translations.source = 'auto'`,
			ctrID(entityType, entityID, field, lang), entityType, entityID, field, lang, translated)
	}
}

// SetManual stores a reviewed (human) translation that auto-translation will
// never clobber.
func (r *TranslationRepository) SetManual(ctx context.Context, entityType, entityID, field, lang, value string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO content_translations (id, entity_type, entity_id, field, lang, value, source, updated_on)
		VALUES ($1,$2,$3,$4,$5,$6,'manual',CURRENT_TIMESTAMP)
		ON CONFLICT (entity_type, entity_id, field, lang)
		DO UPDATE SET value = EXCLUDED.value, source = 'manual', updated_on = CURRENT_TIMESTAMP`,
		ctrID(entityType, entityID, field, lang), entityType, entityID, field, lang, value)
	return err
}

// Localize returns value keyed by entity_id for one (entity_type, field, lang).
// Empty map when lang is the base language (callers keep the base text).
func (r *TranslationRepository) Localize(ctx context.Context, entityType, field, lang string, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if lang == i18n.BaseLang || len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT entity_id, value FROM content_translations
		WHERE entity_type = $1 AND field = $2 AND lang = $3 AND entity_id = ANY($4)`,
		entityType, field, lang, ids)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, val string
		if err := rows.Scan(&id, &val); err != nil {
			return out, err
		}
		out[id] = val
	}
	return out, rows.Err()
}

// LocalizeMaps overwrites the given fields on each row-map with their
// translation for lang, in place. Rows are keyed by their "id" value. No-op
// when lang is the base language. Used to localize the []map[string]any that
// the catalog reads return without changing those queries.
func (r *TranslationRepository) LocalizeMaps(ctx context.Context, entityType, lang string, rows []map[string]any, fields ...string) {
	if lang == i18n.BaseLang || len(rows) == 0 {
		return
	}
	ids := make([]string, 0, len(rows))
	byID := map[string]map[string]any{}
	for _, row := range rows {
		id, ok := row["id"].(string)
		if !ok || id == "" {
			continue
		}
		ids = append(ids, id)
		byID[id] = row
	}
	for _, field := range fields {
		loc, err := r.Localize(ctx, entityType, field, lang, ids)
		if err != nil {
			continue
		}
		for id, val := range loc {
			if row := byID[id]; row != nil {
				row[field] = val
			}
		}
	}
}

// Delete removes all translations for an entity (call when the entity is deleted).
func (r *TranslationRepository) Delete(ctx context.Context, entityType, entityID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM content_translations WHERE entity_type=$1 AND entity_id=$2`,
		entityType, entityID)
	return err
}
