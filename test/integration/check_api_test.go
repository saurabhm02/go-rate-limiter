//go:build integration

package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestCheckAPIEndpoint(t *testing.T) {
	srv, _ := setupFullStack(t)

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", bytes.NewBufferString(`{"route":"/api/payments/1","cost":1}`))
		req.Header.Set("X-API-Key", "rl_demo_abc123xyz")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d status=%d", i+1, resp.StatusCode)
		}
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/check", bytes.NewBufferString(`{"route":"/api/payments/1","cost":1}`))
	req.Header.Set("X-API-Key", "rl_demo_abc123xyz")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", resp.StatusCode, respBody)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Fatalf("missing Retry-After, headers=%v", resp.Header)
	}
}
