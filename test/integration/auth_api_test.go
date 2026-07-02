//go:build integration

package integration_test

import (
	"io"
	"net/http"
	"testing"
)

func TestCheckAPIRejectsMissingAPIKey(t *testing.T) {
	srv, _ := setupFullStack(t)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}

func TestCheckAPIRejectsInvalidAPIKey(t *testing.T) {
	srv, _ := setupFullStack(t)

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", nil)
	req.Header.Set("X-API-Key", "rl_invalid_key")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}

func TestConfigAPIRejectsMissingAPIKey(t *testing.T) {
	pool := setupPostgres(t)
	srv := newTestServer(t, pool, serverOptions{})

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/config/tenants", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
