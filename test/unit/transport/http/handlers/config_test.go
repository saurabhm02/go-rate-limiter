package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/handlers"
	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

var (
	callerTenant = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	otherTenant  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

type fakeTenants map[uuid.UUID]entity.Tenant

func (f fakeTenants) GetByID(_ context.Context, id uuid.UUID) (*entity.Tenant, error) {
	t, ok := f[id]
	if !ok {
		return nil, domainerrors.ErrTenantNotFound
	}
	return &t, nil
}

type fakeRules map[uuid.UUID][]entity.Rule

func (f fakeRules) ListByTenant(_ context.Context, id uuid.UUID) ([]entity.Rule, error) {
	return f[id], nil
}

func (f fakeRules) ListAll(context.Context) ([]entity.Rule, error) { return nil, nil }

func configFixture() *handlers.ConfigHandler {
	return handlers.NewConfigHandler(
		fakeTenants{
			callerTenant: {ID: callerTenant, Name: "caller-co", Status: entity.TenantStatusActive},
			otherTenant:  {ID: otherTenant, Name: "someone-else", Status: entity.TenantStatusActive},
		},
		fakeRules{
			callerTenant: {{ID: uuid.New(), RoutePattern: "/mine/*", Algorithm: entity.AlgorithmSlidingWindow, Enabled: true, LimitCount: 5, WindowSeconds: 600}},
			otherTenant:  {{ID: uuid.New(), RoutePattern: "/secret/*", Algorithm: entity.AlgorithmSlidingWindow, Enabled: true, LimitCount: 99, WindowSeconds: 60}},
		},
	)
}

func asCaller(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	return req.WithContext(httputil.WithTenantID(req.Context(), callerTenant))
}

// The tenant list must not be a directory of everyone using the service.
func TestListTenantsReturnsOnlyTheCaller(t *testing.T) {
	rec := httptest.NewRecorder()
	configFixture().ListTenants(rec, asCaller(http.MethodGet, "/v1/config/tenants"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "caller-co") {
		t.Fatalf("caller's own tenant missing: %s", body)
	}
	if strings.Contains(body, "someone-else") || strings.Contains(body, otherTenant.String()) {
		t.Fatalf("leaked another tenant: %s", body)
	}
}

// Another tenant's rules would expose their route patterns and limits.
func TestListTenantRulesRejectsOtherTenants(t *testing.T) {
	h := configFixture()

	t.Run("own rules are readable", func(t *testing.T) {
		req := asCaller(http.MethodGet, "/v1/config/tenants/"+callerTenant.String()+"/rules")
		req.SetPathValue("id", callerTenant.String())
		rec := httptest.NewRecorder()
		h.ListTenantRules(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "/mine/*") {
			t.Fatalf("own rules missing: %s", rec.Body.String())
		}
	})

	t.Run("another tenant's rules are not", func(t *testing.T) {
		req := asCaller(http.MethodGet, "/v1/config/tenants/"+otherTenant.String()+"/rules")
		req.SetPathValue("id", otherTenant.String())
		rec := httptest.NewRecorder()
		h.ListTenantRules(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("cross-tenant read should 404, got %d body=%s", rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "/secret/*") {
			t.Fatalf("leaked another tenant's rules: %s", rec.Body.String())
		}
	})

	// A foreign tenant and a nonexistent one must be indistinguishable, or the
	// endpoint becomes an oracle for which tenant IDs are real.
	t.Run("foreign and nonexistent are indistinguishable", func(t *testing.T) {
		missing := uuid.MustParse("33333333-3333-3333-3333-333333333333")

		reqForeign := asCaller(http.MethodGet, "/v1/config/tenants/x/rules")
		reqForeign.SetPathValue("id", otherTenant.String())
		recForeign := httptest.NewRecorder()
		h.ListTenantRules(recForeign, reqForeign)

		reqMissing := asCaller(http.MethodGet, "/v1/config/tenants/x/rules")
		reqMissing.SetPathValue("id", missing.String())
		recMissing := httptest.NewRecorder()
		h.ListTenantRules(recMissing, reqMissing)

		if recForeign.Code != recMissing.Code || recForeign.Body.String() != recMissing.Body.String() {
			t.Fatalf("responses differ: foreign %d %s vs missing %d %s",
				recForeign.Code, recForeign.Body.String(), recMissing.Code, recMissing.Body.String())
		}
	})
}

// Without auth context there is no tenant to scope to; never fall back to "all".
func TestConfigHandlersRequireTenantContext(t *testing.T) {
	h := configFixture()

	rec := httptest.NewRecorder()
	h.ListTenants(rec, httptest.NewRequest(http.MethodGet, "/v1/config/tenants", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ListTenants without context: %d %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/config/tenants/x/rules", nil)
	req.SetPathValue("id", callerTenant.String())
	rec = httptest.NewRecorder()
	h.ListTenantRules(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("ListTenantRules without context: %d %s", rec.Code, rec.Body.String())
	}
}
