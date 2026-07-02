package entity

import "github.com/google/uuid"

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
)

type Tenant struct {
	ID     uuid.UUID
	Name   string
	Status TenantStatus
}

func (t Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}
