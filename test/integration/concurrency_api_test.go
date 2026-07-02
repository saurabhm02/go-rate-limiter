//go:build integration

package integration_test

import (
	"bytes"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
)

// Full HTTP stack: 500 concurrent checks against /v1/check rule (limit 100/min).
// Documented tolerance: at most limit+1 allowed under race.
func TestConcurrentCheckAPIRespectsLimit(t *testing.T) {
	srv, _ := setupFullStack(t)

	const (
		limit      = 100
		goroutines = 500
	)

	var allowed int64
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/check",
				bytes.NewBufferString(`{"route":"/v1/check","cost":1}`))
			if err != nil {
				t.Errorf("new request: %v", err)
				return
			}
			req.Header.Set("X-API-Key", "rl_demo_abc123xyz")
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("do request: %v", err)
				return
			}
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	if allowed > limit+1 {
		t.Fatalf("allowed %d requests, limit %d (tolerance +1)", allowed, limit)
	}
	if allowed < limit-1 {
		t.Fatalf("allowed only %d, expected close to %d", allowed, limit)
	}
}
