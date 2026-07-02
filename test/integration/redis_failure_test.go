//go:build integration

package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestCheckAPIReturns503WhenRedisDown(t *testing.T) {
	pool := setupPostgres(t)
	redisClient, mr := setupMiniredis(t)
	srv := newTestServer(t, pool, serverOptions{withCheck: true, redis: redisClient})

	mr.Close()
	_ = redisClient.Close()

	body := bytes.NewBufferString(`{"route":"/api/payments/1","cost":1}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", body)
	req.Header.Set("X-API-Key", "rl_demo_abc123xyz")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", resp.StatusCode, respBody)
	}
}
