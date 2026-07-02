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

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	migrationsDir := filepath.Join("migrations")
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		log.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)

	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("read %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			log.Fatalf("exec %s: %v", file, err)
		}
		log.Printf("applied %s", filepath.Base(file))
	}

	seedPath := filepath.Join(migrationsDir, "seeds", "dev_seed.sql")
	seedBytes, err := os.ReadFile(seedPath)
	if err != nil {
		log.Fatalf("read seed: %v", err)
	}
	if _, err := pool.Exec(ctx, string(seedBytes)); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Printf("seed already applied")
		} else {
			log.Fatalf("exec seed: %v", err)
		}
	} else {
		log.Printf("applied %s", seedPath)
	}

	fmt.Println("migrations complete")
}
