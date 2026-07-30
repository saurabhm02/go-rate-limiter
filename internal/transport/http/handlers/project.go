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

type ProjectService interface {
	CreateProject(ctx context.Context, name, keyHash, keyPrefix string, keyRole entity.APIKeyRole, rules []entity.Rule) (uuid.UUID, error)
	GetProject(ctx context.Context, tenantID uuid.UUID) (*ports.ProjectSummary, error)
	AddAPIKey(ctx context.Context, tenantID uuid.UUID, keyHash, keyPrefix string, role entity.APIKeyRole) error
	RevokeAPIKey(ctx context.Context, tenantID, keyID uuid.UUID) error
}

type ProjectHandler struct {
	projects ProjectService
}

func NewProjectHandler(projects ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

type createProjectBody struct {
	Name      string     `json:"name"`
	KeyHash   string     `json:"key_hash"`
	KeyPrefix string     `json:"key_prefix"`
	Rules     []ruleBody `json:"rules"`
}

type ruleBody struct {
	RoutePattern   string  `json:"route_pattern"`
	Algorithm      string  `json:"algorithm"`
	LimitCount     int64   `json:"limit_count"`
	WindowSeconds  int64   `json:"window_seconds"`
	BucketCapacity int64   `json:"bucket_capacity"`
	RefillRate     float64 `json:"refill_rate"`
	Enabled        *bool   `json:"enabled"`
}

func (b ruleBody) toEntity() entity.Rule {
	enabled := true
	if b.Enabled != nil {
		enabled = *b.Enabled
	}
	return entity.Rule{
		RoutePattern:   b.RoutePattern,
		Algorithm:      entity.Algorithm(b.Algorithm),
		Enabled:        enabled,
		LimitCount:     b.LimitCount,
		WindowSeconds:  b.WindowSeconds,
		BucketCapacity: b.BucketCapacity,
		RefillRate:     b.RefillRate,
	}
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createProjectBody
	if !decodeBody(w, r, &body) {
		return
	}

	rules := make([]entity.Rule, 0, len(body.Rules))
	for _, rb := range body.Rules {
		rules = append(rules, rb.toEntity())
	}

	tenantID, err := h.projects.CreateProject(r.Context(), body.Name, body.KeyHash, body.KeyPrefix, entity.RoleAdmin, rules)
	if err != nil {
		writeProjectError(w, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":    tenantID.String(),
		"name":  body.Name,
		"rules": len(rules),
	})
}

func (h *ProjectHandler) Me(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := httputil.TenantIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	p, err := h.projects.GetProject(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrTenantNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "project_not_found")
			return
		}
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, projectPayload(p))
}

type addKeyBody struct {
	KeyHash   string `json:"key_hash"`
	KeyPrefix string `json:"key_prefix"`
	Role      string `json:"role"`
}

func (h *ProjectHandler) AddKey(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := httputil.TenantIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body addKeyBody
	if !decodeBody(w, r, &body) {
		return
	}

	role := entity.APIKeyRole(body.Role)
	if role == "" {
		role = entity.RoleCheck
	}

	if err := h.projects.AddAPIKey(r.Context(), tenantID, body.KeyHash, body.KeyPrefix, role); err != nil {
		writeProjectError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *ProjectHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := httputil.TenantIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized")
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

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	return true
}

func writeProjectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrProjectExists):
		httputil.WriteError(w, http.StatusConflict, "project_exists")
	case errors.Is(err, domainerrors.ErrTenantNotFound):
		httputil.WriteError(w, http.StatusNotFound, "project_not_found")
	case errors.Is(err, domainerrors.ErrInvalidInput):
		httputil.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable")
	}
}

func projectPayload(p *ports.ProjectSummary) map[string]any {
	rules := make([]ruleView, 0, len(p.Rules))
	for _, r := range p.Rules {
		rules = append(rules, ruleView{
			ID:             r.ID.String(),
			RoutePattern:   r.RoutePattern,
			Algorithm:      string(r.Algorithm),
			Enabled:        r.Enabled,
			LimitCount:     r.LimitCount,
			WindowSeconds:  r.WindowSeconds,
			BucketCapacity: r.BucketCapacity,
			RefillRate:     r.RefillRate,
		})
	}
	keys := make([]keyView, 0, len(p.Keys))
	for _, k := range p.Keys {
		keys = append(keys, keyView{
			ID:        k.ID.String(),
			Prefix:    k.Prefix,
			Status:    k.Status,
			Role:      k.Role,
			CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{
		"id":         p.ID.String(),
		"name":       p.Name,
		"status":     p.Status,
		"created_at": p.CreatedAt.UTC().Format(time.RFC3339),
		"rule_count": p.RuleCount,
		"rules":      rules,
		"keys":       keys,
	}
}
