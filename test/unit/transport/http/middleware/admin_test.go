package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saurabh/distributed-rate-limiter/internal/transport/http/middleware"
)

func reached() (http.Handler, *bool) {
	hit := false
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}), &hit
}

func TestAdminAuthFailsClosedWithoutToken(t *testing.T) {
	h, hit := reached()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/projects", nil)
	req.Header.Set("Authorization", "Bearer anything")

	middleware.AdminAuth("", h).ServeHTTP(rec, req)

	if *hit {
		t.Fatal("handler ran with no admin token configured")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503", rec.Code)
	}
}

func TestAdminAuthRejectsBadTokens(t *testing.T) {
	cases := map[string]string{
		"missing header": "",
		"empty bearer":   "Bearer ",
		"wrong token":    "Bearer nope",
		"prefix of real": "Bearer s3cr3",
		"no bearer word": "s3cr3t",
		"tenant style":   "s3cr3t-but-wrong",
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			h, hit := reached()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/admin/projects", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}

			middleware.AdminAuth("s3cr3t", h).ServeHTTP(rec, req)

			if *hit {
				t.Fatalf("handler ran for %q", header)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d want 401", rec.Code)
			}
		})
	}
}

func TestAdminAuthAcceptsTheToken(t *testing.T) {
	h, hit := reached()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/projects", nil)
	req.Header.Set("Authorization", "Bearer s3cr3t")

	middleware.AdminAuth("s3cr3t", h).ServeHTTP(rec, req)

	if !*hit || rec.Code != http.StatusOK {
		t.Fatalf("valid token rejected: hit=%v status=%d", *hit, rec.Code)
	}
}

// Admin paths bypass tenant auth, so the prefix check must not be loose.
func TestIsAdminPath(t *testing.T) {
	cases := map[string]bool{
		"/v1/admin/projects":      true,
		"/v1/admin/projects/x":    true,
		"/v1/check":               false,
		"/v1/config/tenants":      false,
		"/health/live":            false,
		"/v1/admin":               false,
		"/evil/v1/admin/projects": false,
		"/v1/adminx/projects":     false,
	}
	for path, want := range cases {
		if got := middleware.IsAdminPath(path); got != want {
			t.Errorf("IsAdminPath(%q)=%v want %v", path, got, want)
		}
	}
}

func TestCORSOnlyEchoesAllowedOrigins(t *testing.T) {
	allowed := []string{"https://gorate.pages.dev"}

	t.Run("allowed origin is echoed", func(t *testing.T) {
		h, _ := reached()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/check", nil)
		req.Header.Set("Origin", "https://gorate.pages.dev")

		middleware.CORS(allowed, h).ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://gorate.pages.dev" {
			t.Fatalf("ACAO=%q", got)
		}
		if rec.Header().Get("Vary") != "Origin" {
			t.Fatalf("missing Vary: Origin")
		}
	})

	t.Run("foreign origin gets nothing", func(t *testing.T) {
		h, hit := reached()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/v1/check", nil)
		req.Header.Set("Origin", "https://evil.example")

		middleware.CORS(allowed, h).ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("leaked ACAO to foreign origin: %q", got)
		}
		if !*hit {
			t.Fatal("handler should still run; CORS is a browser-side control")
		}
	})

	t.Run("preflight short-circuits before auth", func(t *testing.T) {
		h, hit := reached()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/v1/admin/projects", nil)
		req.Header.Set("Origin", "https://gorate.pages.dev")
		req.Header.Set("Access-Control-Request-Method", "POST")

		middleware.CORS(allowed, h).ServeHTTP(rec, req)

		if *hit {
			t.Fatal("preflight should not reach the handler")
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status=%d want 204", rec.Code)
		}
	})
}
