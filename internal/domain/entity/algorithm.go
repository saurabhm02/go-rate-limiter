package entity

import "fmt"

type Algorithm string

const (
	AlgorithmTokenBucket   Algorithm = "token_bucket"
	AlgorithmSlidingWindow Algorithm = "sliding_window"
)

func ParseAlgorithm(value string) (Algorithm, error) {
	switch Algorithm(value) {
	case AlgorithmTokenBucket, AlgorithmSlidingWindow:
		return Algorithm(value), nil
	default:
		return "", fmt.Errorf("unknown algorithm %q", value)
	}
}

func (a Algorithm) String() string {
	return string(a)
}

func (a Algorithm) Valid() bool {
	return a == AlgorithmTokenBucket || a == AlgorithmSlidingWindow
}
