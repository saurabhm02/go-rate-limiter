package entity

import "github.com/google/uuid"

type APIKeyStatus string

const (
	APIKeyStatusActive  APIKeyStatus = "active"
	APIKeyStatusRevoked APIKeyStatus = "revoked"
)

type APIKeyRole string

const (
	RoleCheck APIKeyRole = "check"
	RoleAdmin APIKeyRole = "admin"
)

type APIKey struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	KeyPrefix string
	Status    APIKeyStatus
	Role      APIKeyRole
}

func (k APIKey) CanManageProject() bool {
	return k.Role == RoleAdmin
}

func (k APIKey) CanCheck() bool {
	return k.Role == RoleCheck
}

func (k APIKey) IsActive() bool {
	return k.Status == APIKeyStatusActive
}
