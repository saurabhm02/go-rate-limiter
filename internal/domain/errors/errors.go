package errors

import "errors"

var (
	ErrTenantNotFound   = errors.New("tenant not found")
	ErrTenantSuspended  = errors.New("tenant suspended")
	ErrAPIKeyNotFound   = errors.New("api key not found")
	ErrAPIKeyRevoked    = errors.New("api key revoked")
	ErrInvalidRule      = errors.New("invalid rule")
	ErrRateLimitBackend = errors.New("rate limit backend unavailable")
	ErrProjectExists    = errors.New("project already exists")
	ErrInvalidInput     = errors.New("invalid input")
)
