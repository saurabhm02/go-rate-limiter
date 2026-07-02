package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/saurabh/distributed-rate-limiter/examples/ratelimitclient"
)

func main() {
	port := envInt("HTTP_PORT", 8083)
	limiterURL := env("RATE_LIMITER_URL", "http://localhost:8080")
	apiKey := env("API_KEY", "rl_demo_abc123xyz")
	upstream := env("UPSTREAM_URL", "http://localhost:8081")

	upstreamURL, err := url.Parse(upstream)
	if err != nil {
		log.Fatalf("invalid UPSTREAM_URL: %v", err)
	}

	client := ratelimitclient.New(limiterURL, apiKey)
	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"status":"ok","service":"gateway-middleware"}`)
			return
		}

		dec, err := client.Check(r.Context(), r.URL.Path, 1)
		if err != nil {
			log.Printf("rate limit check failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":"rate_limiter_unavailable"}`)
			return
		}
		if !dec.Allowed {
			if dec.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.FormatInt(dec.RetryAfter, 10))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":"rate_limit_exceeded","enforced_by":"gateway-middleware"}`)
			return
		}

		proxy.ServeHTTP(w, r)
	})

	addr := ":" + strconv.Itoa(port)
	log.Printf("gateway-middleware listening on %s (upstream=%s limiter=%s)", addr, upstream, limiterURL)
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
