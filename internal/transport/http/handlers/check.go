package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/observability"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

// RateLimitChecker is the application seam used by the check HTTP handler.
type RateLimitChecker interface {
	Check(ctx context.Context, tenantID uuid.UUID, route, subject string, cost int64) (entity.RateLimitDecision, error)
}

type CheckHandler struct {
	checker RateLimitChecker
}

func NewCheckHandler(checker RateLimitChecker) *CheckHandler {
	return &CheckHandler{checker: checker}
}

type checkRequest struct {
	Route string `json:"route"`
	// Subject is optional. It discriminates the counter (per IP, per account,
	// ...) without affecting which rule matches — route stays the rule selector.
	Subject string `json:"subject"`
	Cost    int64  `json:"cost"`
}

type checkResponse struct {
	Allowed    bool   `json:"allowed"`
	Limit      int64  `json:"limit,omitempty"`
	Remaining  int64  `json:"remaining,omitempty"`
	ResetAt    int64  `json:"reset_at,omitempty"`
	RetryAfter *int64 `json:"retry_after,omitempty"`
	Algorithm  string `json:"algorithm,omitempty"`
}

func (h *CheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	tenantID, ok := httputil.TenantIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	var req checkRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
			return
		}
	}

	req.Route = strings.TrimSpace(req.Route)
	if req.Route == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	req.Subject = strings.TrimSpace(req.Subject)
	if req.Cost <= 0 {
		req.Cost = 1
	}

	checkStart := time.Now()
	decision, err := h.checker.Check(r.Context(), tenantID, req.Route, req.Subject, req.Cost)
	observability.RecordRateLimitCheck(decision, err, time.Since(checkStart))
	if err != nil {
		if errors.Is(err, domainerrors.ErrRateLimitBackend) {
			httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
			return
		}
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}

	httputil.ApplyRateLimitHeaders(w, decision)
	status := httputil.CheckStatusCode(decision)

	resp := checkResponse{
		Allowed:    decision.Allowed,
		Limit:      decision.Limit,
		Remaining:  decision.Remaining,
		ResetAt:    httputil.FormatResetAt(decision.ResetAt),
		RetryAfter: httputil.FormatRetryAfter(decision),
	}
	if decision.Algorithm != "" {
		resp.Algorithm = decision.Algorithm.String()
	}

	httputil.WriteJSON(w, status, resp)
}
