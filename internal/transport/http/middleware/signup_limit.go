package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/saurabh/distributed-rate-limiter/pkg/httputil"
)

type signupLimiter struct {
	mu      sync.Mutex
	seen    map[string][]time.Time
	limit   int
	window  time.Duration
	trustXF bool
}

func SignupLimit(limit int, window time.Duration, trustForwarded bool, next http.Handler) http.Handler {
	l := &signupLimiter{
		seen:    make(map[string][]time.Time),
		limit:   limit,
		window:  window,
		trustXF: trustForwarded,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(l.client(r), time.Now()) {
			w.Header().Set("Retry-After", "3600")
			httputil.WriteError(w, http.StatusTooManyRequests, "too_many_projects")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *signupLimiter) allow(client string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	kept := l.seen[client][:0]
	for _, t := range l.seen[client] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.limit {
		l.seen[client] = kept
		return false
	}
	l.seen[client] = append(kept, now)

	if len(l.seen) > 10_000 {
		for k, v := range l.seen {
			if len(v) == 0 || v[len(v)-1].Before(cutoff) {
				delete(l.seen, k)
			}
		}
	}
	return true
}

func (l *signupLimiter) client(r *http.Request) string {
	if l.trustXF {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := splitAndTrim(xff)
			if n := len(parts); n > 0 {
				return parts[n-1]
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func splitAndTrim(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
