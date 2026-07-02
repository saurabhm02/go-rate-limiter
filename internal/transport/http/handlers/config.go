package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
	"github.com/saurabh/distributed-rate-limiter/internal/infrastructure/postgres"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

type ConfigHandler struct {
	tenants *postgres.TenantRepository
	rules   ports.RuleRepository
}

func NewConfigHandler(tenants *postgres.TenantRepository, rules ports.RuleRepository) *ConfigHandler {
	return &ConfigHandler{tenants: tenants, rules: rules}
}

type tenantResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ruleResponse struct {
	ID            string  `json:"id"`
	RoutePattern  string  `json:"route_pattern"`
	Algorithm     string  `json:"algorithm"`
	Enabled       bool    `json:"enabled"`
	LimitCount    int64   `json:"limit_count,omitempty"`
	WindowSeconds int64   `json:"window_seconds,omitempty"`
	BucketCapacity int64  `json:"bucket_capacity,omitempty"`
	RefillRate    float64 `json:"refill_rate,omitempty"`
}

func (h *ConfigHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.tenants.List(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}

	out := make([]tenantResponse, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, tenantResponse{
			ID:     t.ID.String(),
			Name:   t.Name,
			Status: string(t.Status),
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"tenants": out})
}

func (h *ConfigHandler) ListTenantRules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if _, err := h.tenants.GetByID(r.Context(), tenantID); err != nil {
		httputil.WriteError(w, http.StatusNotFound, "tenant_not_found")
		return
	}

	rules, err := h.rules.ListByTenant(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}

	out := make([]ruleResponse, 0, len(rules))
	for _, rule := range rules {
		out = append(out, ruleResponse{
			ID:             rule.ID.String(),
			RoutePattern:   rule.RoutePattern,
			Algorithm:      string(rule.Algorithm),
			Enabled:        rule.Enabled,
			LimitCount:     rule.LimitCount,
			WindowSeconds:  rule.WindowSeconds,
			BucketCapacity: rule.BucketCapacity,
			RefillRate:     rule.RefillRate,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"rules": out})
}
