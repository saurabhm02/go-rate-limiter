package entity_test

import (
	"testing"

	"github.com/saurabh/distributed-rate-limiter/internal/domain/entity"
)

func TestParseAlgorithm(t *testing.T) {
	a, err := entity.ParseAlgorithm("token_bucket")
	if err != nil || a != entity.AlgorithmTokenBucket {
		t.Fatalf("got %v, %v", a, err)
	}

	_, err = entity.ParseAlgorithm("fixed_window")
	if err == nil {
		t.Fatal("expected error for unknown algorithm")
	}
}
