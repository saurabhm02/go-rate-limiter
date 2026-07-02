//go:build integration

package integration_test

import (
	"io"
	"net/http"
	"testing"
)

func TestHealthLiveAlwaysOK(t *testing.T) {
	srv, _ := setupFullStack(t)

	resp, err := http.Get(srv.URL + "/health/live")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}

func TestHealthReadyWhenDependenciesUp(t *testing.T) {
	srv, _ := setupFullStack(t)

	resp, err := http.Get(srv.URL + "/health/ready")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}

func TestHealthReadyFailsWhenRedisDown(t *testing.T) {
	pool := setupPostgres(t)
	redisClient, mr := setupMiniredis(t)
	srv := newTestServer(t, pool, serverOptions{withCheck: true, redis: redisClient})

	mr.Close()
	_ = redisClient.Close()

	resp, err := http.Get(srv.URL + "/health/ready")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}
