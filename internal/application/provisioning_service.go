// Creating projects and their API keys.
// Everything the browser sends gets checked here before it reaches the
// database, because the browser is not something we can trust.
package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

var (
	projectNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)
	sha256HexRe   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

const maxRulesPerProject = 50

type ProvisioningService struct {
	projects ports.ProjectStore
}

// NewProvisioningService wires the service to whatever stores projects.
func NewProvisioningService(projects ports.ProjectStore) *ProvisioningService {
	return &ProvisioningService{projects: projects}
}

// CreateProject checks the name, the key hash and every rule, then writes them.
// It returns the new tenant id.
//
// We only ever receive the hash of the key. The real key is made in the browser
// and shown to the person once, so it never lands in our logs or a backup.
func (s *ProvisioningService) CreateProject(ctx context.Context, name, keyHash, keyPrefix string, keyRole entity.APIKeyRole, rules []entity.Rule) (uuid.UUID, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !projectNameRe.MatchString(name) {
		return uuid.Nil, fmt.Errorf("%w: name must be 3-40 chars, lowercase letters, digits and hyphens, not starting or ending with a hyphen", domainerrors.ErrInvalidInput)
	}

	keyHash = strings.ToLower(strings.TrimSpace(keyHash))
	if !sha256HexRe.MatchString(keyHash) {
		return uuid.Nil, fmt.Errorf("%w: key_hash must be 64 lowercase hex characters (sha256)", domainerrors.ErrInvalidInput)
	}

	keyPrefix = strings.TrimSpace(keyPrefix)
	if keyPrefix == "" || len(keyPrefix) > 32 {
		return uuid.Nil, fmt.Errorf("%w: key_prefix must be 1-32 characters", domainerrors.ErrInvalidInput)
	}

	if len(rules) == 0 {
		return uuid.Nil, fmt.Errorf("%w: at least one rule is required", domainerrors.ErrInvalidInput)
	}
	if len(rules) > maxRulesPerProject {
		return uuid.Nil, fmt.Errorf("%w: at most %d rules per project", domainerrors.ErrInvalidInput, maxRulesPerProject)
	}
	tenantID := uuid.New()

	seen := make(map[string]struct{}, len(rules))
	for i := range rules {
		rules[i].TenantID = tenantID
		rules[i].RoutePattern = strings.TrimSpace(rules[i].RoutePattern)

		if _, dup := seen[rules[i].RoutePattern]; dup {
			return uuid.Nil, fmt.Errorf("%w: duplicate route pattern %q", domainerrors.ErrInvalidInput, rules[i].RoutePattern)
		}
		seen[rules[i].RoutePattern] = struct{}{}

		if err := rules[i].Validate(); err != nil {
			return uuid.Nil, fmt.Errorf("%w: rule %d (%s): %v", domainerrors.ErrInvalidInput, i+1, rules[i].RoutePattern, err)
		}
	}

	if keyRole != entity.RoleAdmin && keyRole != entity.RoleCheck {
		keyRole = entity.RoleAdmin
	}

	p := ports.NewProject{
		TenantID:  tenantID,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeyRole:   keyRole,
		Rules:     rules,
	}
	if err := s.projects.CreateProject(ctx, p); err != nil {
		return uuid.Nil, err
	}
	return tenantID, nil
}

// ListProjects hands the dashboard every project.
func (s *ProvisioningService) ListProjects(ctx context.Context) ([]ports.ProjectSummary, error) {
	return s.projects.ListProjects(ctx)
}

// AddAPIKey adds another key to an existing project. Same checks as creating
// one, since this is just as much a way in.
func (s *ProvisioningService) AddAPIKey(ctx context.Context, tenantID uuid.UUID, keyHash, keyPrefix string, role entity.APIKeyRole) error {
	keyHash = strings.ToLower(strings.TrimSpace(keyHash))
	if !sha256HexRe.MatchString(keyHash) {
		return fmt.Errorf("%w: key_hash must be 64 lowercase hex characters (sha256)", domainerrors.ErrInvalidInput)
	}
	keyPrefix = strings.TrimSpace(keyPrefix)
	if keyPrefix == "" || len(keyPrefix) > 32 {
		return fmt.Errorf("%w: key_prefix must be 1-32 characters", domainerrors.ErrInvalidInput)
	}
	if role != entity.RoleAdmin && role != entity.RoleCheck {
		return fmt.Errorf("%w: role must be admin or check", domainerrors.ErrInvalidInput)
	}
	return s.projects.AddAPIKey(ctx, tenantID, keyHash, keyPrefix, role)
}

// RevokeAPIKey turns a key off without losing the record of it.
func (s *ProvisioningService) RevokeAPIKey(ctx context.Context, tenantID, keyID uuid.UUID) error {
	return s.projects.RevokeAPIKey(ctx, tenantID, keyID)
}

// GetProject loads the one project a console session belongs to.
func (s *ProvisioningService) GetProject(ctx context.Context, tenantID uuid.UUID) (*ports.ProjectSummary, error) {
	return s.projects.GetProject(ctx, tenantID)
}
