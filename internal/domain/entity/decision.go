package entity

// RateLimitDecision is the outcome of evaluating a request against a rule.
// It is an immutable value object passed between application and transport layers.
type RateLimitDecision struct {
	Allowed    bool
	Limit      int64
	Remaining  int64
	ResetAt    int64 
	RetryAfter int64 
	Algorithm  Algorithm
}
