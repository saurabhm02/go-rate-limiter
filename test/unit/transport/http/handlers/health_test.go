package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/handlers"
)

func TestHealthLive(t *testing.T) {
	h := handlers.NewHealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHealthReady(t *testing.T) {
	h := handlers.NewHealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
