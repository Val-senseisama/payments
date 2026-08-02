package company

import (
	"context"
	"database/sql"
	"fmt"

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