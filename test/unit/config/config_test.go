package config_test

import (
	"testing"
	"time"

	"github.com/saurabh/distributed-rate-limiter/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}
	if cfg.RuleCacheTTL != 30*time.Second {
		t.Errorf("RuleCacheTTL = %v, want 30s", cfg.RuleCacheTTL)
	}
	if cfg.Addr() != ":8080" {
		t.Errorf("Addr() = %q, want :8080", cfg.Addr())
	}
}

func TestLoadInvalidHTTPPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-port")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid HTTP_PORT")
	}
}
