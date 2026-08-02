package users

import (
	"context"
	"database/sql"

	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateUser creates a new user on the platform
func (s *Store) CreateUser(ctx context.Context, name, email string, role types.UserRole, avatarURL *string) (*types.User, error) {
	query := `
		INSERT INTO users (name, email, role, avatar_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, company_id, name, email, avatar_url, role, created_at, deleted_at
	`

	user := new(types.User)
	var companyID sql.NullString

	err := s.db.QueryRowContext(ctx, query, name, email, role, avatarURL).Scan(
		&user.ID,
		&companyID,
		&user.Name,
		&user.Email,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	if companyID.Valid {
		user.CompanyID, _ = uuid.Parse(companyID.String)
	}

	return user, nil
}

// AddUserToCompany assigns an existing user to a company with a specific role
func (s *Store) AddUserToCompany(ctx context.Context, userID uuid.UUID, companyID uuid.UUID, role types.UserRole) error {
	query := `
		UPDATE users
		SET company_id = $1, role = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, companyID, role, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	query := `
		SELECT id, company_id, name, email, avatar_url, role, created_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	user := new(types.User)
	var companyID sql.NullString
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&companyID,
		&user.Name,
		&user.Email,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	if companyID.Valid {
		user.CompanyID, _ = uuid.Parse(companyID.String)
	}
	return user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	query := `
		SELECT id, company_id, name, email, avatar_url, role, created_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	user := new(types.User)
	var companyID sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&companyID,
		&user.Name,
		&user.Email,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	if companyID.Valid {
		user.CompanyID, _ = uuid.Parse(companyID.String)
	}
	return user, nil
}
