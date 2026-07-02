package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/handlers"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

type mockChecker struct {
	decision entity.RateLimitDecision
	err      error
}

func (m mockChecker) Check(ctx context.Context, tenantID uuid.UUID, route string, cost int64) (entity.RateLimitDecision, error) {
	return m.decision, m.err
}

func TestCheckHandlerAllowed(t *testing.T) {
	h := handlers.NewCheckHandler(mockChecker{
		decision: entity.RateLimitDecision{
			Allowed:   true,
			Limit:     100,
			Remaining: 42,
			ResetAt:   1719854400,
			Algorithm: entity.AlgorithmSlidingWindow,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewBufferString(`{"route":"/api/payments","cost":1}`))
	req = req.WithContext(httputil.WithTenantID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-RateLimit-Limit") != "100" {
		t.Fatalf("headers=%v", rec.Header())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["allowed"] != true {
		t.Fatalf("resp=%v", resp)
	}
}

func TestCheckHandlerDenied(t *testing.T) {
	retry := int64(30)
	h := handlers.NewCheckHandler(mockChecker{
		decision: entity.RateLimitDecision{
			Allowed:    false,
			Limit:      100,
			Remaining:  0,
			ResetAt:    1719854400,
			RetryAfter: retry,
			Algorithm:  entity.AlgorithmSlidingWindow,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewBufferString(`{"route":"/api/payments"}`))
	req = req.WithContext(httputil.WithTenantID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "30" {
		t.Fatalf("retry-after=%q", rec.Header().Get("Retry-After"))
	}
}

func TestCheckHandlerInvalidBody(t *testing.T) {
	h := handlers.NewCheckHandler(mockChecker{})
	req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewBufferString(`{"route":""}`))
	req = req.WithContext(httputil.WithTenantID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestCheckHandlerRedisDown(t *testing.T) {
	h := handlers.NewCheckHandler(mockChecker{err: domainerrors.ErrRateLimitBackend})
	req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewBufferString(`{"route":"/api/payments"}`))
	req = req.WithContext(httputil.WithTenantID(req.Context(), uuid.New()))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}
