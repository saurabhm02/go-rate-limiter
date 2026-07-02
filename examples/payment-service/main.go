package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/saurabh/distributed-rate-limiter/examples/ratelimitclient"
)

func main() {
	port := envInt("HTTP_PORT", 8081)
	limiterURL := env("RATE_LIMITER_URL", "http://localhost:8080")
	apiKey := env("API_KEY", "rl_demo_abc123xyz")

	client := ratelimitclient.New(limiterURL, apiKey)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		route := r.URL.Path
		dec, err := client.Check(r.Context(), route, 1)
		if err != nil {
			log.Printf("rate limit check failed: %v", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "rate_limiter_unavailable"})
			return
		}
		if !dec.Allowed {
			if dec.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.FormatInt(dec.RetryAfter, 10))
			}
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":   "rate_limit_exceeded",
				"route":   route,
				"message": "payment rate limit exceeded — try again later",
			})
			return
		}

		id := r.PathValue("id")
		writeJSON(w, http.StatusCreated, map[string]any{
			"payment_id": id,
			"status":     "created",
			"remaining":  dec.Remaining,
		})
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "payment-service"})
	})

	addr := ":" + strconv.Itoa(port)
	log.Printf("payment-service listening on %s (limiter=%s)", addr, limiterURL)
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := env(key, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
