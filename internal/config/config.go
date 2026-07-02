package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort        int
	DatabaseURL     string
	RedisURL        string
	LogLevel        string
	RuleCacheTTL    time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPPort:        8080,
		DatabaseURL:     "postgres://ratelimit:ratelimit@localhost:5432/ratelimit?sslmode=disable",
		RedisURL:        "redis://localhost:6379/0",
		LogLevel:        "info",
		RuleCacheTTL:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HTTP_PORT %q: %w", port, err)
		}
		cfg.HTTPPort = p
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	if v := os.Getenv("REDIS_URL"); v != "" {
		cfg.RedisURL = v
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("RULE_CACHE_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid RULE_CACHE_TTL %q: %w", v, err)
		}
		cfg.RuleCacheTTL = d
	}

	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid SHUTDOWN_TIMEOUT %q: %w", v, err)
		}
		cfg.ShutdownTimeout = d
	}

	return cfg, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}
