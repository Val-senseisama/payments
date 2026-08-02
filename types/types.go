package types

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UserRole represents the authorized role of a user within a company
type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleAdmin      UserRole = "admin"
	RoleFinance    UserRole = "finance"
	RoleViewer     UserRole = "viewer"
)

// ProfileType represents the category of a financial counterparty
type ProfileType string

const (
	ProfileCustomer ProfileType = "customer"
	ProfileVendor   ProfileType = "vendor"
	ProfileEmployee ProfileType = "employee"
)

// AuditAction represents a strongly-typed audit event action
type AuditAction string

const (
	AuditActionUserCreated    AuditAction = "user.created"
	AuditActionUserUpdated    AuditAction = "user.updated"
	AuditActionUserDeleted    AuditAction = "user.deleted"
	AuditActionCompanyCreated AuditAction = "company.created"
	AuditActionCompanyUpdated AuditAction = "company.updated"
	AuditActionCompanyDeleted AuditAction = "company.deleted"
)

type UserStore interface {
	CreateUser(ctx context.Context, name, email string, role UserRole, avatarURL *string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	AddUserToCompany(ctx context.Context, userID uuid.UUID, companyID uuid.UUID, role UserRole) error
}

type RedisStore interface {
	SaveRefreshToken(ctx context.Context, userID string, tokenID string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID string) (string, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
}

type CompanyStore interface {
	CreateCompany(ctx context.Context, name string, createdBy uuid.UUID) (*Company, error)
	GetCompanyByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetCompanies(ctx context.Context) ([]*Company, error)
	UpdateCompany(ctx context.Context, id uuid.UUID, name string) (*Company, error)
	DeleteCompany(ctx context.Context, id uuid.UUID) (*Company, error)
}

type AuditStore interface {
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*AuditLog, error)
	GetAuditLogsByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*AuditLog, error)
}

// AuditLog represents a lean record in the audit_log table
type AuditLog struct {
	ID         uuid.UUID       `json:"id"`
	CompanyID  *uuid.UUID      `json:"company_id,omitempty"`
	EntityType string          `json:"entity_type"`
	EntityID   *uuid.UUID      `json:"entity_id,omitempty"`
	Action     AuditAction     `json:"action"`
	ChangedBy  *uuid.UUID      `json:"changed_by,omitempty"`
	NewValues  json.RawMessage `json:"new_values,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Company represents the multi-tenant organization entity
type Company struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	CreatedBy uuid.UUID  `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// User represents an admin or staff user belonging to a company
type User struct {
	ID        uuid.UUID  `json:"id"`
	CompanyID uuid.UUID  `json:"company_id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	AvatarURL *string    `json:"avatar_url,omitempty"`
	Role      UserRole   `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type AuthenthicatedUser struct {
	UserID    uuid.UUID
	CompanyID *uuid.UUID
	Role      UserRole
}

// Profile represents a financial entity (customer, vendor, or employee)
type Profile struct {
	ID        uuid.UUID   `json:"id"`
	CompanyID uuid.UUID   `json:"company_id"`
	Type      ProfileType `json:"type"`
	Name      string      `json:"name"`
	Email     *string     `json:"email,omitempty"`
	Phone     *string     `json:"phone,omitempty"`
	AvatarURL *string     `json:"avatar_url,omitempty"`
	Metadata  []byte      `json:"metadata,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	DeletedAt *time.Time  `json:"deleted_at,omitempty"`
}

// CreateCompanyPayload represents HTTP request payload for creating a company
type CreateCompanyPayload struct {
	Name string `json:"name" validate:"required"`
}

// CreateUserPayload represents HTTP request payload for creating a user
type CreateUserPayload struct {
	CompanyID *uuid.UUID `json:"company_id,omitempty"`
	Name      string     `json:"name" validate:"required"`
	Email     string     `json:"email" validate:"required,email"`
	AvatarURL *string    `json:"avatar_url,omitempty"`
	Role      UserRole   `json:"role,omitempty"`
}

type RequestLoginOTPPayload struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginWithOTPPayload struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required"`
}

// CreateProfilePayload represents HTTP request payload for creating a profile
type CreateProfilePayload struct {
	CompanyID uuid.UUID   `json:"company_id" validate:"required"`
	Type      ProfileType `json:"type" validate:"required"`
	Name      string      `json:"name" validate:"required"`
	Email     *string     `json:"email,omitempty"`
	Phone     *string     `json:"phone,omitempty"`
	AvatarURL *string     `json:"avatar_url,omitempty"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ContextKey string
