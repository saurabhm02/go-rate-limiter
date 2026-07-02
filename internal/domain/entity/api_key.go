package entity

import "github.com/google/uuid"


type APIKeyStatus string

const (
	APIKeyStatusActive  APIKeyStatus = "active"
	APIKeyStatusRevoked APIKeyStatus = "revoked"
)


type APIKey struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	KeyPrefix string
	Status    APIKeyStatus
}

func (k APIKey) IsActive() bool {
	return k.Status == APIKeyStatusActive
}
