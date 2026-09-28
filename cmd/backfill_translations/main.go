// Backfill translate-on-write over content that predates migration 011
// (seeded catalog). Idempotent: AutoTranslate upserts source='auto' and never
// clobbers a manual override. With the Noop translator this simply mirrors the
// base text; run it after wiring a real MT provider to translate the whole
// existing catalog in one pass.
//
//	go run ./cmd/backfill_translations
package main

import (
	"context"
	"fmt"
	"log"

	"dailzo/config"
	"dailzo/db"
	"dailzo/pkg/i18n"
	"dailzo/repository"
)

func main() {
	cfg := config.LoadConfig()
	db.ConnectDatabase(cfg)
	defer db.DB.Close()
	ctx := context.Background()

	trans := repository.NewTranslationRepository(db.DB, i18n.NewTranslator())

	// Categories
	catN := 0
	rows, err := db.DB.Query(ctx, `SELECT id, name, COALESCE(description,'') FROM categories`)
	if err != nil {
		log.Fatalf("query categories: %v", err)
	}
	for rows.Next() {
		var id, name, desc string
		if err := rows.Scan(&id, &name, &desc); err != nil {
			log.Fatalf("scan category: %v", err)
		}
		trans.AutoTranslate(ctx, "category", id, "name", name)
		if desc != "" {
			trans.AutoTranslate(ctx, "category", id, "description", desc)
		}
		catN++
	}
	rows.Close()

	// Products
	prodN := 0
	rows, err = db.DB.Query(ctx, `SELECT id, name, COALESCE(description,'') FROM food_products`)
	if err != nil {
		log.Fatalf("query products: %v", err)
	}
	for rows.Next() {
		var id, name, desc string
		if err := rows.Scan(&id, &name, &desc); err != nil {
			log.Fatalf("scan product: %v", err)
		}
		trans.AutoTranslate(ctx, "product", id, "name", name)
		if desc != "" {
			trans.AutoTranslate(ctx, "product", id, "description", desc)
		}
		prodN++
	}
	rows.Close()

	fmt.Printf("backfilled translations for %d categories and %d products\n", catN, prodN)
}
