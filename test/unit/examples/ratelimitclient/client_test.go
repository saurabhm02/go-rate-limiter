package ratelimitclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saurabh/distributed-rate-limiter/examples/ratelimitclient"
)

func TestClientCheckAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/check" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Fatalf("missing api key")
		}
		w.Header().Set("X-RateLimit-Limit", "10")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allowed":   true,
			"limit":     10,
			"remaining": 9,
		})
	}))
	t.Cleanup(srv.Close)

	client := ratelimitclient.New(srv.URL, "test-key")
	dec, err := client.Check(context.Background(), "/api/payments/1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !dec.Allowed || dec.Remaining != 9 {
		t.Fatalf("dec=%+v", dec)
	}
}

func TestClientCheckDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allowed":     false,
			"retry_after": 5,
		})
	}))
	t.Cleanup(srv.Close)

	client := ratelimitclient.New(srv.URL, "test-key")
	dec, err := client.Check(context.Background(), "/api/payments/1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Allowed || dec.RetryAfter != 5 {
		t.Fatalf("dec=%+v", dec)
	}
}
