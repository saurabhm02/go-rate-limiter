//go:build integration

package integration_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/saurabh/distributed-rate-limiter/internal/application"
	"github.com/saurabh/distributed-rate-limiter/internal/infrastructure/cache"
	redisinfra "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/redis"
	pgstore "github.com/saurabh/distributed-rate-limiter/internal/infrastructure/postgres"
	httptransport "github.com/saurabh/distributed-rate-limiter/internal/transport/http"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/handlers"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/middleware"
)

const cacheTTL = time.Second

// testEnv holds shared integration test infrastructure.
type testEnv struct {
	pool        *pgxpool.Pool
	redisClient *goredis.Client
	miniredis   *miniredis.Miniredis
}

func setupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("ratelimit"),
		postgres.WithUsername("ratelimit"),
		postgres.WithPassword("ratelimit"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	applyMigrations(t, ctx, dsn)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func setupMiniredis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return client, mr
}

func applyMigrations(t *testing.T, ctx context.Context, dsn string) {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	migrationsDir := filepath.Join(root, "migrations")
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("exec %s: %v", file, err)
		}
	}
	seed, err := os.ReadFile(filepath.Join(migrationsDir, "seeds", "dev_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(seed)); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

type serverOptions struct {
	withCheck bool
	redis     *goredis.Client
}

func newTestServer(t *testing.T, pool *pgxpool.Pool, opts serverOptions) *httptest.Server {
	t.Helper()

	tenantRepo := pgstore.NewTenantRepository(pool)
	apiKeyRepo := pgstore.NewAPIKeyRepository(pool)
	ruleRepo := cache.NewRuleCache(pgstore.NewRuleRepository(pool), time.Second)
	authService := application.NewAuthService(apiKeyRepo, tenantRepo)

	healthCheckers := []handlers.HealthChecker{
		pgstore.PoolPinger{PingFn: func(ctx context.Context) error { return pgstore.Ping(ctx, pool) }},
	}
	if opts.redis != nil {
		healthCheckers = append(healthCheckers, redisinfra.NewClientPinger(opts.redis))
	}

	deps := httptransport.Dependencies{
		HealthCheckers: healthCheckers,
		Auth:           middleware.NewAuthMiddleware(authService),
		Tenants:        tenantRepo,
		Rules:          ruleRepo,
	}

	if opts.withCheck {
		limiter, err := redisinfra.NewRateLimiter(opts.redis)
		if err != nil {
			t.Fatal(err)
		}
		deps.Check = application.NewCheckService(ruleRepo, application.NewRuleResolver(), limiter)
	}

	srv := httptest.NewServer(httptransport.NewMux(deps))
	t.Cleanup(srv.Close)
	return srv
}

func setupFullStack(t *testing.T) (*httptest.Server, testEnv) {
	t.Helper()
	pool := setupPostgres(t)
	redisClient, mr := setupMiniredis(t)
	srv := newTestServer(t, pool, serverOptions{withCheck: true, redis: redisClient})
	return srv, testEnv{pool: pool, redisClient: redisClient, miniredis: mr}
}
