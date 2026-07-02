package httputil

import (
	"net/http"
	"strconv"

	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

const (
	HeaderRateLimitLimit     = "X-RateLimit-Limit"
	HeaderRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderRateLimitReset     = "X-RateLimit-Reset"
	HeaderRetryAfter         = "Retry-After"
)

// ApplyRateLimitHeaders sets standard rate limit response headers.
func ApplyRateLimitHeaders(w http.ResponseWriter, dec entity.RateLimitDecision) {
	if dec.Limit <= 0 && dec.Algorithm == "" {
		return
	}

	w.Header().Set(HeaderRateLimitLimit, strconv.FormatInt(dec.Limit, 10))
	w.Header().Set(HeaderRateLimitRemaining, strconv.FormatInt(dec.Remaining, 10))
	w.Header().Set(HeaderRateLimitReset, strconv.FormatInt(dec.ResetAt, 10))

	if !dec.Allowed && dec.RetryAfter > 0 {
		w.Header().Set(HeaderRetryAfter, strconv.FormatInt(dec.RetryAfter, 10))
	}
}

// CheckStatusCode returns 200 for allowed checks and 429 for denied.
func CheckStatusCode(dec entity.RateLimitDecision) int {
	if dec.Allowed {
		return http.StatusOK
	}
	return http.StatusTooManyRequests
}

// FormatResetAt returns reset timestamp for JSON responses.
func FormatResetAt(resetAt int64) int64 {
	return resetAt
}

// FormatRetryAfter returns retry_after for JSON when denied.
func FormatRetryAfter(dec entity.RateLimitDecision) *int64 {
	if dec.Allowed || dec.RetryAfter <= 0 {
		return nil
	}
	v := dec.RetryAfter
	return &v
}
