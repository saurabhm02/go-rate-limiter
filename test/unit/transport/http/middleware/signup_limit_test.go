package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/middleware"
)

func post(h http.Handler, remoteAddr, xff string) int {
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestSignupLimitStopsAFlood(t *testing.T) {
	h, _ := reached()
	limited := middleware.SignupLimit(3, time.Hour, false, h)

	for i := 1; i <= 3; i++ {
		if code := post(limited, "203.0.113.7:1000", ""); code != http.StatusOK {
			t.Fatalf("request %d should pass, got %d", i, code)
		}
	}
	if code := post(limited, "203.0.113.7:1000", ""); code != http.StatusTooManyRequests {
		t.Fatalf("4th request should be limited, got %d", code)
	}
	if code := post(limited, "198.51.100.9:1000", ""); code != http.StatusOK {
		t.Fatalf("other client should pass, got %d", code)
	}
}

func TestSignupLimitIgnoresForwardedHeaderWhenNotBehindProxy(t *testing.T) {
	h, _ := reached()
	limited := middleware.SignupLimit(2, time.Hour, false, h)

	for i := 0; i < 2; i++ {
		post(limited, "203.0.113.7:1000", "1.1.1.1")
	}
	// Same connection, different forged header: still the same bucket.
	if code := post(limited, "203.0.113.7:1000", "9.9.9.9"); code != http.StatusTooManyRequests {
		t.Fatalf("forged X-Forwarded-For bought a fresh allowance, got %d", code)
	}
}

func TestSignupLimitUsesLastForwardedHop(t *testing.T) {
	h, _ := reached()
	limited := middleware.SignupLimit(1, time.Hour, true, h)

	if code := post(limited, "10.0.0.1:1000", "9.9.9.9, 203.0.113.7"); code != http.StatusOK {
		t.Fatalf("first should pass, got %d", code)
	}
	if code := post(limited, "10.0.0.1:1000", "1.2.3.4, 203.0.113.7"); code != http.StatusTooManyRequests {
		t.Fatalf("spoofed prefix bought a fresh allowance, got %d", code)
	}
}
