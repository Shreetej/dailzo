// Migration applier: runs .sql files through the app's own pgx pool, so
// migrations use the exact same connection settings as the server. Tracks
// which files have already been applied in a schema_migrations table, so:
//   - re-running is a no-op instead of relying on every migration's own SQL
//     being carefully written IF NOT EXISTS/idempotent (they are today, but
//     that was a convention to remember, not something enforced)
//   - different environments (dev/staging/prod) can be checked against each
//     other for drift by comparing this table
//
// Usage:
//
//	go run ./cmd/migrate -all                                  # apply every not-yet-applied migration in order
//	go run ./cmd/migrate -file migrations/016_otp_hardening.sql # apply one file
//	go run ./cmd/migrate -status                                # list applied vs pending
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"dailzo/config"
	"dailzo/db"
)

const migrationsDir = "migrations"

func ensureTrackingTable(ctx context.Context) {
	_, err := db.DB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.schema_migrations (
			version     VARCHAR(255) PRIMARY KEY,
			applied_on  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		log.Fatalf("could not create schema_migrations tracking table: %v", err)
	}
}

func appliedVersions(ctx context.Context) map[string]bool {
	rows, err := db.DB.Query(ctx, `SELECT version FROM public.schema_migrations`)
	if err != nil {
		log.Fatalf("could not read schema_migrations: %v", err)
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			log.Fatalf("scan schema_migrations: %v", err)
		}
		applied[v] = true
	}
	return applied
}

func allMigrationFiles() []string {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("read %s: %v", migrationsDir, err)
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // filenames are numerically prefixed (001_, 002_...)
	return files
}

func applyOne(ctx context.Context, path, version string) {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}
	tx, err := db.DB.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx for %s: %v", version, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		log.Fatalf("apply %s: %v", path, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO public.schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, version); err != nil {
		log.Fatalf("record %s as applied: %v", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit %s: %v", version, err)
	}
	fmt.Printf("applied %s\n", path)
}

func main() {
	file := flag.String("file", "", "path to a single .sql migration file to apply")
	all := flag.Bool("all", false, "apply every migration in migrations/ not yet recorded as applied")
	status := flag.Bool("status", false, "list applied vs pending migrations and exit")
	flag.Parse()

	if *file == "" && !*all && !*status {
		log.Fatal("usage: go run ./cmd/migrate (-all | -status | -file <path.sql>)")
	}

	cfg := config.LoadConfig()
	db.ConnectDatabase(cfg)
	defer db.DB.Close()

	ctx := context.Background()
	ensureTrackingTable(ctx)
	applied := appliedVersions(ctx)

	switch {
	case *status:
		for _, f := range allMigrationFiles() {
			state := "pending"
			if applied[f] {
				state = "applied"
			}
			fmt.Printf("%-45s %s\n", f, state)
		}
	case *all:
		for _, f := range allMigrationFiles() {
			if applied[f] {
				continue
			}
			applyOne(ctx, filepath.Join(migrationsDir, f), f)
		}
	default:
		version := filepath.Base(*file)
		if applied[version] {
			fmt.Printf("%s already applied, skipping\n", version)
			return
		}
		applyOne(ctx, *file, version)
	}
}
