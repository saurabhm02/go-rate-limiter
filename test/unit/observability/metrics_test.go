package observability_test

import (
	"testing"
	"time"

	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
	domainerrors "github.com/saurabh/distributed-rate-limiter/internal/domain/errors"
	"github.com/saurabh/distributed-rate-limiter/internal/observability"
)

func TestRecordRateLimitCheckDoesNotPanic(t *testing.T) {
	observability.RecordRateLimitCheck(entity.RateLimitDecision{Allowed: true}, nil, time.Millisecond)
	observability.RecordRateLimitCheck(entity.RateLimitDecision{
		Allowed:   false,
		Limit:     10,
		Algorithm: entity.AlgorithmSlidingWindow,
	}, nil, time.Millisecond)
	observability.RecordRateLimitCheck(entity.RateLimitDecision{}, domainerrors.ErrRateLimitBackend, time.Millisecond)
}

func TestRecordHTTPRequestDoesNotPanic(t *testing.T) {
	observability.RecordHTTPRequest("GET", "/health/ready", 200, time.Millisecond)
	observability.RecordHTTPRequest("POST", "/v1/check", 429, time.Millisecond)
}
