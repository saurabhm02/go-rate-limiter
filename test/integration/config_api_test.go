//go:build integration

package integration_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestConfigAPIWithSeedData(t *testing.T) {
	pool := setupPostgres(t)
	srv := newTestServer(t, pool, serverOptions{})

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/config/tenants", nil)
	req.Header.Set("X-API-Key", "rl_demo_abc123xyz")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "demo-corp") {
		t.Fatalf("expected demo-corp in %s", body)
	}
}
