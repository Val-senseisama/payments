package account

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

func (s *Store) CreateAccount(ctx context.Context, companyID uuid.UUID, profileID *uuid.UUID, accType types.AccountType, name string) (*types.Account, error) {
	query := `
		INSERT INTO accounts (company_id, profile_id, type, name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, company_id, profile_id, type, name, cached_balance, created_at
	`

	var a types.Account
	err := s.db.QueryRowContext(ctx, query, companyID, profileID, accType, name).Scan(
		&a.ID,
		&a.CompanyID,
		&a.ProfileID,
		&a.Type,
		&a.Name,
		&a.CachedBalance,
		&a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &a, nil
}

func (s *Store) GetAccountByID(ctx context.Context, id uuid.UUID) (*types.Account, error) {
	query := `
		SELECT id, company_id, profile_id, type, name, cached_balance, created_at
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
	`

	var a types.Account
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.CompanyID,
		&a.ProfileID,
		&a.Type,
		&a.Name,
		&a.CachedBalance,
		&a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &a, nil
}

func (s *Store) GetAccountsByCompany(ctx context.Context, companyID uuid.UUID) ([]*types.Account, error) {
	query := `
		SELECT id, company_id, profile_id, type, name, cached_balance, created_at
		FROM accounts
		WHERE company_id = $1 AND deleted_at IS NULL
	`

	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts by company: %w", err)
	}
	defer rows.Close()

	var accounts []*types.Account
	for rows.Next() {
		var a types.Account
		err := rows.Scan(
			&a.ID,
			&a.CompanyID,
			&a.ProfileID,
			&a.Type,
			&a.Name,
			&a.CachedBalance,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row: %w", err)
		}
		accounts = append(accounts, &a)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows: %w", err)
	}
	return accounts, nil
}


func (s *Store) SeedDefaultAccounts(ctx context.Context, companyID uuid.UUID) error {
	defaultAccounts := []struct {
		accType types.AccountType
		name    string
	}{
		{types.AccountAsset, "Cash and Cash Equivalents"},
		{types.AccountAsset, "Accounts Receivable"},
		{types.AccountLiability, "Accounts Payable"},
		{types.AccountRevenue, "Revenue"},
		{types.AccountExpense, "Expenses"},
	}

	for _, acc := range defaultAccounts {
		_, err := s.CreateAccount(ctx, companyID, nil, acc.accType, acc.name)
		if err != nil {
			return fmt.Errorf("failed to seed default account: %w", err)
		}
	}
	return nil
}
