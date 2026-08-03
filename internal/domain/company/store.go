package company

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Val-senseisama/payments/types"
	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateCompany(ctx context.Context, name string, createdBy uuid.UUID) (*types.Company, error) {
	query := `
		INSERT INTO companies (name, created_by)
		VALUES ($1, $2)
		RETURNING id, name, created_by, created_at, deleted_at
	`

	var company types.Company
	err := s.db.QueryRowContext(ctx, query, name, createdBy).Scan(
		&company.ID,
		&company.Name,
		&company.CreatedBy,
		&company.CreatedAt,
		&company.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return &company, nil
}

func (s *Store) GetCompanyByID(ctx context.Context, id uuid.UUID) (*types.Company, error) {
	query := `
		SELECT id, name, created_by, created_at, deleted_at
		FROM companies
		WHERE id = $1 AND deleted_at IS NULL
	`

	var company types.Company
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&company.ID,
		&company.Name,
		&company.CreatedBy,
		&company.CreatedAt,
		&company.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("company not found")
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	return &company, nil
}

func (s *Store) GetCompanies(ctx context.Context) ([]*types.Company, error) {
	query := `
		SELECT id, name, created_by, created_at, deleted_at
		FROM companies
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query companies: %w", err)
	}
	defer rows.Close()

	var companies []*types.Company
	for rows.Next() {
		var company types.Company
		err := rows.Scan(
			&company.ID,
			&company.Name,
			&company.CreatedBy,
			&company.CreatedAt,
			&company.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan company row: %w", err)
		}
		companies = append(companies, &company)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating company rows: %w", err)
	}
	return companies, nil
}

func (s *Store) UpdateCompany(ctx context.Context, id uuid.UUID, name string) (*types.Company, error) {
	query := `
		UPDATE companies
		SET name = $1
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, name, created_by, created_at, deleted_at
	`

	var company types.Company
	err := s.db.QueryRowContext(ctx, query, name, id).Scan(
		&company.ID,
		&company.Name,
		&company.CreatedBy,
		&company.CreatedAt,
		&company.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("company not found")
		}
		return nil, fmt.Errorf("failed to update company: %w", err)
	}

	return &company, nil
}

func (s *Store) DeleteCompany(ctx context.Context, id uuid.UUID) (*types.Company, error) {
	query := `
		UPDATE companies
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, name, created_by, created_at, deleted_at
	`

	var company types.Company
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&company.ID,
		&company.Name,
		&company.CreatedBy,
		&company.CreatedAt,
		&company.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("company not found")
		}
		return nil, fmt.Errorf("failed to soft delete company: %w", err)
	}

	return &company, nil
}

func (s *Store) CreateInvitationToken(ctx context.Context, companyID uuid.UUID, email string, role types.UserRole, token string, expiresAt time.Time) (*types.InvitationToken, error) {
	query := `
		INSERT INTO invitation_tokens (company_id, email, role, token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, company_id, email, role, token, expires_at, accepted_at, created_at
	`
	var inv types.InvitationToken
	err := s.db.QueryRowContext(ctx, query, companyID, email, string(role), token, expiresAt).Scan(
		&inv.ID,
		&inv.CompanyID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation token: %w", err)
	}
	return &inv, nil
}

func (s *Store) GetInvitationToken(ctx context.Context, token string) (*types.InvitationToken, error) {
	query := `
		SELECT id, company_id, email, role, token, expires_at, accepted_at, created_at
		FROM invitation_tokens
		WHERE token = $1
	`
	var inv types.InvitationToken
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID,
		&inv.CompanyID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invitation token not found")
		}
		return nil, fmt.Errorf("failed to get invitation token: %w", err)
	}
	return &inv, nil
}

func (s *Store) MarkInvitationTokenAccepted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE invitation_tokens
		SET accepted_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}