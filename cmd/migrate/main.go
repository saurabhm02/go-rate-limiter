package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://ratelimit:ratelimit@localhost:5432/ratelimit?sslmode=disable"
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		log.Fatalf("create schema_migrations: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		log.Fatalf("glob migrations: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no migrations found in %s", migrationsDir)
	}
	sort.Strings(files)

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".up.sql")

		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
		).Scan(&applied); err != nil {
			log.Fatalf("check %s: %v", version, err)
		}
		if applied {
			log.Printf("skip %s (already applied)", version)
			continue
		}

		if err := applyMigration(ctx, pool, version, file); err != nil {
			log.Fatalf("apply %s: %v", version, err)
		}
		log.Printf("applied %s", version)
	}

	// Dev seed is opt-in and idempotent (ON CONFLICT DO NOTHING), so it is not
	// version-tracked — re-running it just refreshes the demo fixtures.
	if os.Getenv("SEED") == "dev" {
		seedPath := filepath.Join(migrationsDir, "seeds", "dev_seed.sql")
		seedSQL, err := os.ReadFile(seedPath)
		if err != nil {
			log.Fatalf("read seed: %v", err)
		}
		if _, err := pool.Exec(ctx, string(seedSQL)); err != nil {
			log.Fatalf("exec seed: %v", err)
		}
		log.Printf("applied %s", seedPath)
	}

	fmt.Println("migrations complete")
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version, file string) error {
	sqlBytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, version,
	); err != nil {
		return fmt.Errorf("record version: %w", err)
	}
	return tx.Commit(ctx)
}
