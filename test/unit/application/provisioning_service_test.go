package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/application"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/ports"
)

type spyProjects struct {
	got ports.NewProject
	err error

	addedKeyFor uuid.UUID
	addedHash   string
}

func (s *spyProjects) CreateProject(_ context.Context, p ports.NewProject) error {
	s.got = p
	return s.err
}

func (s *spyProjects) ListProjects(context.Context) ([]ports.ProjectSummary, error) {
	return nil, s.err
}

func (s *spyProjects) AddAPIKey(_ context.Context, tenantID uuid.UUID, keyHash, keyPrefix string) error {
	s.addedKeyFor, s.addedHash = tenantID, keyHash
	return s.err
}

func (s *spyProjects) RevokeAPIKey(context.Context, uuid.UUID, uuid.UUID) error {
	return s.err
}

func goodRules() []entity.Rule {
	return []entity.Rule{{
		RoutePattern:  "/app/signup/*",
		Algorithm:     entity.AlgorithmSlidingWindow,
		Enabled:       true,
		LimitCount:    3,
		WindowSeconds: 3600,
	}}
}

const goodHash = "cf3186cdc08eee7c8c16b40d70c34d330941a861c9a64e8afa2870d607cd22bb"

func TestCreateProjectWritesEverythingTogether(t *testing.T) {
	spy := &spyProjects{}
	svc := application.NewProvisioningService(spy)

	id, err := svc.CreateProject(context.Background(), "  MyApp  ", goodHash, "rl_myapp_", goodRules())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("no tenant id returned")
	}
	if spy.got.Name != "myapp" {
		t.Fatalf("name not normalised: %q", spy.got.Name)
	}
	if spy.got.TenantID != id {
		t.Fatalf("tenant id mismatch")
	}
	// Rules must carry the tenant, or they would be written unattached.
	for _, r := range spy.got.Rules {
		if r.TenantID != id {
			t.Fatalf("rule not stamped with tenant id: %+v", r)
		}
	}
}

func TestCreateProjectRejectsBadInput(t *testing.T) {
	cases := map[string]struct {
		name, hash, prefix string
		rules              []entity.Rule
	}{
		"empty name":        {"", goodHash, "rl_", goodRules()},
		"name too short":    {"ab", goodHash, "rl_", goodRules()},
		"name with spaces":  {"my app", goodHash, "rl_", goodRules()},
		"name with slash":   {"my/app", goodHash, "rl_", goodRules()},
		"leading hyphen":    {"-app", goodHash, "rl_", goodRules()},
		"hash not hex":      {"myapp", strings.Repeat("z", 64), "rl_", goodRules()},
		"hash wrong length": {"myapp", "abc123", "rl_", goodRules()},
		"empty prefix":      {"myapp", goodHash, "", goodRules()},
		"no rules":          {"myapp", goodHash, "rl_", nil},
		"rule missing limit": {"myapp", goodHash, "rl_", []entity.Rule{{
			RoutePattern: "/x/*", Algorithm: entity.AlgorithmSlidingWindow, Enabled: true,
			WindowSeconds: 60, // LimitCount omitted
		}}},
		"unknown algorithm": {"myapp", goodHash, "rl_", []entity.Rule{{
			RoutePattern: "/x/*", Algorithm: entity.Algorithm("leaky_bucket"), Enabled: true,
			LimitCount: 1, WindowSeconds: 60,
		}}},
		"empty pattern": {"myapp", goodHash, "rl_", []entity.Rule{{
			RoutePattern: "   ", Algorithm: entity.AlgorithmSlidingWindow, Enabled: true,
			LimitCount: 1, WindowSeconds: 60,
		}}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			spy := &spyProjects{}
			svc := application.NewProvisioningService(spy)

			_, err := svc.CreateProject(context.Background(), tc.name, tc.hash, tc.prefix, tc.rules)
			if err == nil {
				t.Fatal("expected rejection")
			}
			if !errors.Is(err, domainerrors.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput so the handler can answer 400, got %v", err)
			}
			if spy.got.Name != "" {
				t.Fatal("invalid input reached the database")
			}
		})
	}
}

// UNIQUE (tenant_id, route_pattern) would reject this anyway; catching it here
// names the offending pattern instead of surfacing a constraint error.
func TestCreateProjectRejectsDuplicatePatterns(t *testing.T) {
	spy := &spyProjects{}
	svc := application.NewProvisioningService(spy)

	rules := []entity.Rule{
		{RoutePattern: "/a/*", Algorithm: entity.AlgorithmSlidingWindow, Enabled: true, LimitCount: 1, WindowSeconds: 60},
		{RoutePattern: "/a/*", Algorithm: entity.AlgorithmSlidingWindow, Enabled: true, LimitCount: 2, WindowSeconds: 60},
	}
	_, err := svc.CreateProject(context.Background(), "myapp", goodHash, "rl_", rules)
	if err == nil || !strings.Contains(err.Error(), "/a/*") {
		t.Fatalf("want a duplicate error naming the pattern, got %v", err)
	}
}

func TestCreateProjectPropagatesConflict(t *testing.T) {
	spy := &spyProjects{err: domainerrors.ErrProjectExists}
	svc := application.NewProvisioningService(spy)

	_, err := svc.CreateProject(context.Background(), "myapp", goodHash, "rl_", goodRules())
	if !errors.Is(err, domainerrors.ErrProjectExists) {
		t.Fatalf("want ErrProjectExists so the handler can answer 409, got %v", err)
	}
}

// Token bucket uses different columns; the same validation must apply.
func TestCreateProjectAcceptsTokenBucket(t *testing.T) {
	spy := &spyProjects{}
	svc := application.NewProvisioningService(spy)

	rules := []entity.Rule{{
		RoutePattern: "/pay/*", Algorithm: entity.AlgorithmTokenBucket, Enabled: true,
		BucketCapacity: 10, RefillRate: 2,
	}}
	if _, err := svc.CreateProject(context.Background(), "payments", goodHash, "rl_pay_", rules); err != nil {
		t.Fatalf("valid token bucket rejected: %v", err)
	}
}

// Rotating a key must apply the same hash validation as creating one — the
// browser is not a trust boundary.
func TestAddAPIKeyValidatesTheHash(t *testing.T) {
	tenantID := uuid.New()

	for name, hash := range map[string]string{
		"not hex":      strings.Repeat("z", 64),
		"too short":    "abc123",
		"empty":        "",
		"uppercase ok": strings.ToUpper(goodHash),
	} {
		t.Run(name, func(t *testing.T) {
			spy := &spyProjects{}
			svc := application.NewProvisioningService(spy)
			err := svc.AddAPIKey(context.Background(), tenantID, hash, "rl_x_")

			if name == "uppercase ok" {
				// Normalised to lowercase rather than rejected.
				if err != nil {
					t.Fatalf("uppercase hash should normalise, got %v", err)
				}
				if spy.addedHash != goodHash {
					t.Fatalf("hash not lowercased: %q", spy.addedHash)
				}
				return
			}
			if !errors.Is(err, domainerrors.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if spy.addedHash != "" {
				t.Fatal("invalid hash reached the store")
			}
		})
	}
}
