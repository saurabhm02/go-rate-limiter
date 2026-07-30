package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

type ProjectCreator interface {
	CreateProject(ctx context.Context, name, keyHash, keyPrefix string, rules []entity.Rule) (uuid.UUID, error)
	ListProjects(ctx context.Context) ([]ports.ProjectSummary, error)
	AddAPIKey(ctx context.Context, tenantID uuid.UUID, keyHash, keyPrefix string) error
	RevokeAPIKey(ctx context.Context, tenantID, keyID uuid.UUID) error
}

type AdminHandler struct {
	projects ProjectCreator
}

func NewAdminHandler(projects ProjectCreator) *AdminHandler {
	return &AdminHandler{projects: projects}
}

type createProjectRequest struct {
	Name      string `json:"name"`
	KeyHash   string `json:"key_hash"`
	KeyPrefix string `json:"key_prefix"`
	Rules     []struct {
		RoutePattern   string  `json:"route_pattern"`
		Algorithm      string  `json:"algorithm"`
		LimitCount     int64   `json:"limit_count"`
		WindowSeconds  int64   `json:"window_seconds"`
		BucketCapacity int64   `json:"bucket_capacity"`
		RefillRate     float64 `json:"refill_rate"`
		Enabled        *bool   `json:"enabled"`
	} `json:"rules"`
}

type createProjectResponse struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Rules    int    `json:"rules"`
}

// CreateProject makes a project, its first key and its rules.
// The body carries the hash of the key, never the key itself.
func (h *AdminHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	var req createProjectRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	rules := make([]entity.Rule, 0, len(req.Rules))
	for _, in := range req.Rules {
		enabled := true
		if in.Enabled != nil {
			enabled = *in.Enabled
		}
		rules = append(rules, entity.Rule{
			RoutePattern:   in.RoutePattern,
			Algorithm:      entity.Algorithm(in.Algorithm),
			Enabled:        enabled,
			LimitCount:     in.LimitCount,
			WindowSeconds:  in.WindowSeconds,
			BucketCapacity: in.BucketCapacity,
			RefillRate:     in.RefillRate,
		})
	}

	tenantID, err := h.projects.CreateProject(r.Context(), req.Name, req.KeyHash, req.KeyPrefix, rules)
	if err != nil {
		switch {
		case errors.Is(err, domainerrors.ErrProjectExists):
			httputil.WriteError(w, http.StatusConflict, "project_exists")
		case errors.Is(err, domainerrors.ErrInvalidInput):
			httputil.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
		}
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, createProjectResponse{
		TenantID: tenantID.String(),
		Name:     req.Name,
		Rules:    len(rules),
	})
}

type keyView struct {
	ID        string `json:"id"`
	Prefix    string `json:"prefix"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type ruleView struct {
	ID             string  `json:"id"`
	RoutePattern   string  `json:"route_pattern"`
	Algorithm      string  `json:"algorithm"`
	Enabled        bool    `json:"enabled"`
	LimitCount     int64   `json:"limit_count"`
	WindowSeconds  int64   `json:"window_seconds"`
	BucketCapacity int64   `json:"bucket_capacity"`
	RefillRate     float64 `json:"refill_rate"`
}

type projectView struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	CreatedAt string     `json:"created_at"`
	RuleCount int        `json:"rule_count"`
	Rules     []ruleView `json:"rules"`
	Keys      []keyView  `json:"keys"`
}

// ListProjects returns every project with its rules and key info.
// Key hashes are left out. Nothing outside login has any use for them.
func (h *AdminHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projects.ListProjects(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}

	out := make([]projectView, 0, len(projects))
	for _, p := range projects {
		pv := projectView{
			ID:        p.ID.String(),
			Name:      p.Name,
			Status:    p.Status,
			CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
			RuleCount: p.RuleCount,
			Rules:     make([]ruleView, 0, len(p.Rules)),
			Keys:      make([]keyView, 0, len(p.Keys)),
		}
		for _, rule := range p.Rules {
			pv.Rules = append(pv.Rules, ruleView{
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
		for _, k := range p.Keys {
			pv.Keys = append(pv.Keys, keyView{
				ID:        k.ID.String(),
				Prefix:    k.Prefix,
				Status:    k.Status,
				CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		out = append(out, pv)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"projects": out})
}

type addKeyRequest struct {
	KeyHash   string `json:"key_hash"`
	KeyPrefix string `json:"key_prefix"`
}

// AddKey mints another key for a project so it can be rotated with no downtime.
func (h *AdminHandler) AddKey(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var req addKeyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	switch err := h.projects.AddAPIKey(r.Context(), tenantID, req.KeyHash, req.KeyPrefix); {
	case err == nil:
		w.WriteHeader(http.StatusCreated)
	case errors.Is(err, domainerrors.ErrTenantNotFound):
		httputil.WriteError(w, http.StatusNotFound, "project_not_found")
	case errors.Is(err, domainerrors.ErrProjectExists):
		httputil.WriteError(w, http.StatusConflict, "key_exists")
	case errors.Is(err, domainerrors.ErrInvalidInput):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
	}
}

// RevokeKey stops a key working. The row stays so the history is still there.
func (h *AdminHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	keyID, err := uuid.Parse(r.PathValue("keyId"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	switch err := h.projects.RevokeAPIKey(r.Context(), tenantID, keyID); {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, domainerrors.ErrAPIKeyNotFound):
		httputil.WriteError(w, http.StatusNotFound, "key_not_found")
	default:
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
	}
}
