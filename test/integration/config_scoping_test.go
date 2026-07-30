//go:build integration

package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const (
	admitdeskTenantID = "8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d11"
	demoTenantID      = "550e8400-e29b-41d4-a716-446655440000"
	demoAPIKey        = "rl_demo_abc123xyz"
)

func get(t *testing.T, url, apiKey string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestConfigAPIOnlyExposesCallersOwnTenant(t *testing.T) {
	pool := setupPostgres(t)
	srv := newTestServer(t, pool, serverOptions{})

	status, body := get(t, srv.URL+"/v1/config/tenants", demoAPIKey)
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, body)
	}

	var payload struct {
		Tenants []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"tenants"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}

	if len(payload.Tenants) != 1 {
		t.Fatalf("expected exactly the caller's tenant, got %d: %s", len(payload.Tenants), body)
	}
	if payload.Tenants[0].Name != "demo-corp" || payload.Tenants[0].ID != demoTenantID {
		t.Fatalf("wrong tenant returned: %s", body)
	}
	if strings.Contains(body, "admitdesk") {
		t.Fatalf("leaked another tenant: %s", body)
	}
}

// Reading another tenant's rules would expose their route patterns and limits.
func TestConfigAPIRejectsOtherTenantsRules(t *testing.T) {
	pool := setupPostgres(t)
	srv := newTestServer(t, pool, serverOptions{})

	// Own rules: allowed.
	status, body := get(t, srv.URL+"/v1/config/tenants/"+demoTenantID+"/rules", demoAPIKey)
	if status != http.StatusOK {
		t.Fatalf("own rules should be readable: status=%d body=%s", status, body)
	}
	if !strings.Contains(body, "/api/payments*") {
		t.Fatalf("expected own seeded rules in %s", body)
	}

	status, body = get(t, srv.URL+"/v1/config/tenants/"+admitdeskTenantID+"/rules", demoAPIKey)
	if status != http.StatusNotFound {
		t.Fatalf("cross-tenant read should 404, got status=%d body=%s", status, body)
	}
	if strings.Contains(body, "admitdesk") {
		t.Fatalf("leaked another tenant's rules: %s", body)
	}

	statusMissing, _ := get(t, srv.URL+"/v1/config/tenants/11111111-1111-1111-1111-111111111111/rules", demoAPIKey)
	if statusMissing != status {
		t.Fatalf("nonexistent tenant returns %d but foreign tenant returns %d — leaks existence", statusMissing, status)
	}
}
