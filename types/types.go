package types

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ContextKey string

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

type AccountType string

const (
	AccountAsset     AccountType = "asset"
	AccountLiability AccountType = "liability"
	AccountRevenue   AccountType = "revenue"
	AccountExpense   AccountType = "expense"
)

type UserStore interface {
	CreateUser(ctx context.Context, name, email string, role UserRole, avatarURL *string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	AddUserToCompany(ctx context.Context, userID uuid.UUID, companyID uuid.UUID, role UserRole) error
	CreateToken(ctx context.Context, userID uuid.UUID, token string, tokenType string, expiresAt time.Time) (*Token, error)
	GetValidToken(ctx context.Context, userID uuid.UUID, token string, tokenType string) (*Token, error)
	MarkTokenUsed(ctx context.Context, id uuid.UUID) error
}

type TransactionType string

const (
	TxnPaymentIn  TransactionType = "payment_in"
	TxnPaymentOut TransactionType = "payment_out"
	TxnTransfer   TransactionType = "transfer"
)

type TransactionStatus string

const (
	TxnStatusPending    TransactionStatus = "pending"
	TxnStatusProcessing TransactionStatus = "processing"
	TxnStatusCompleted  TransactionStatus = "completed"
	TxnStatusFailed     TransactionStatus = "failed"
)

type EntryDirection string

const (
	EntryDebit  EntryDirection = "debit"
	EntryCredit EntryDirection = "credit"
)

type Transaction struct {
	ID        uuid.UUID         `json:"id"`
	CompanyID uuid.UUID         `json:"company_id"`
	Reference string            `json:"reference"`
	Type      TransactionType   `json:"type"`
	Amount    int64             `json:"amount"` // in kobo (NGN * 100)
	Status    TransactionStatus `json:"status"`
	CreatedBy uuid.UUID         `json:"created_by"`
	CreatedAt time.Time         `json:"created_at"`
}

type Entry struct {
	ID            uuid.UUID      `json:"id"`
	TransactionID uuid.UUID      `json:"transaction_id"`
	AccountID     uuid.UUID      `json:"account_id"`
	Amount        int64          `json:"amount"`
	Direction     EntryDirection `json:"direction"`
	CreatedAt     time.Time      `json:"created_at"`
}

type CreateTransactionPayload struct {
	Reference string          `json:"reference" validate:"required"`
	Type      TransactionType `json:"type" validate:"required,oneof=payment_in payment_out transfer"`
	Amount    int64           `json:"amount" validate:"required,gt=0"`
}

type PostLedgerPayload struct {
	TransactionID uuid.UUID `json:"transaction_id" validate:"required"`
	DebitAccount  uuid.UUID `json:"debit_account" validate:"required"`
	CreditAccount uuid.UUID `json:"credit_account" validate:"required"`
}

type TransactionStore interface {
	CreateTransaction(ctx context.Context, companyID, createdBy uuid.UUID, ref string, tType TransactionType, amount int64) (*Transaction, error)
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	GetTransactionsByCompany(ctx context.Context, companyID uuid.UUID) ([]*Transaction, error)
	UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status TransactionStatus) error
}

type LedgerStore interface {
	PostTransaction(ctx context.Context, txnID, companyID, debitAccountID, creditAccountID uuid.UUID, amount int64) error
	GetEntriesByTransaction(ctx context.Context, txnID uuid.UUID) ([]*Entry, error)
}

type RedisStore interface {
	SaveRefreshToken(ctx context.Context, userID string, tokenID string, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID string) (string, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
	SetIdempotencyKey(ctx context.Context, key string, value []byte, ttl time.Duration) error
	GetIdempotencyKey(ctx context.Context, key string) ([]byte, error)
	AcquirePostingLock(ctx context.Context, txnID string, ttl time.Duration) (bool, error)
	ReleasePostingLock(ctx context.Context, txnID string) error
	PublishTransactionUpdate(ctx context.Context, companyID string, payload []byte) error
	EnqueueLedgerJob(ctx context.Context, payload []byte) error
	DequeueLedgerJob(ctx context.Context, timeout time.Duration) ([]byte, error)
	EnqueueDLQJob(ctx context.Context, payload []byte) error
	SubscribeTransactionUpdates(ctx context.Context, companyID string) (<-chan string, func())
}

type ProfileStore interface {
	CreateProfile(ctx context.Context, companyID uuid.UUID, pType ProfileType, name string, email, phone, avatarURL *string, metadata []byte) (*Profile, error)
	GetProfileByID(ctx context.Context, id uuid.UUID) (*Profile, error)
	GetProfilesByCompany(ctx context.Context, companyID uuid.UUID, pType *ProfileType) ([]*Profile, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, name *string, email, phone, avatarURL *string) (*Profile, error)
	DeleteProfile(ctx context.Context, id uuid.UUID) (*Profile, error)
}

type CompanyStore interface {
	CreateCompany(ctx context.Context, name string, createdBy uuid.UUID) (*Company, error)
	GetCompanyByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetCompanies(ctx context.Context) ([]*Company, error)
	UpdateCompany(ctx context.Context, id uuid.UUID, name string) (*Company, error)
	DeleteCompany(ctx context.Context, id uuid.UUID) (*Company, error)
	CreateInvitationToken(ctx context.Context, companyID uuid.UUID, email string, role UserRole, token string, expiresAt time.Time) (*InvitationToken, error)
	GetInvitationToken(ctx context.Context, token string) (*InvitationToken, error)
	MarkInvitationTokenAccepted(ctx context.Context, id uuid.UUID) error
}

type AccountStore interface {
	CreateAccount(ctx context.Context, companyID uuid.UUID, profileID *uuid.UUID, aType AccountType, name string) (*Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error)
	GetAccountsByCompany(ctx context.Context, companyID uuid.UUID) ([]*Account, error)
	SeedDefaultAccounts(ctx context.Context, companyID uuid.UUID) error
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

type Account struct {
	ID            uuid.UUID   `json:"id"`
	CompanyID     uuid.UUID   `json:"company_id"`
	ProfileID     *uuid.UUID  `json:"profile_id,omitempty"` // nil for company-level defaults
	Type          AccountType `json:"type"`
	Name          string      `json:"name"`
	CachedBalance int64       `json:"cached_balance"` // kobo, always 0 for now
	CreatedAt     time.Time   `json:"created_at"`
}

type InvitationToken struct {
	ID         uuid.UUID  `json:"id"`
	CompanyID  uuid.UUID  `json:"company_id"`
	Email      string     `json:"email"`
	Role       UserRole   `json:"role"`
	Token      string     `json:"token"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Token struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Token     string     `json:"token"`
	Type      string     `json:"type"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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

type UpdateProfilePayload struct {
	Name      *string `json:"name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type InviteUserToCompanyPayload struct {
	Email     string    `json:"email" validate:"required,email"`
	Role      UserRole  `json:"role,omitempty"`
	CompanyID uuid.UUID `json:"company_id" validate:"required"`
}

type AcceptCompanyInvitePayload struct {
	Token string `json:"token" validate:"required"`
}

type CreateAccountPayload struct {
	ProfileID *uuid.UUID  `json:"profile_id,omitempty"`
	Type      AccountType `json:"type" validate:"required,oneof=asset liability revenue expense"`
	Name      string      `json:"name" validate:"required"`
}
