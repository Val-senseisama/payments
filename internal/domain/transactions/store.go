package transactions

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

func (s *Store) CreateTransaction(ctx context.Context, companyID, createdBy uuid.UUID, ref string, tType types.TransactionType, amount int64) (*types.Transaction, error) {
	query := `
		INSERT INTO transactions (company_id, created_by, reference, type, amount, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, company_id, reference, type, amount, status, created_by, created_at
	`

	var t types.Transaction
	err := s.db.QueryRowContext(ctx, query, companyID, createdBy, ref, tType, amount).Scan(
		&t.ID,
		&t.CompanyID,
		&t.Reference,
		&t.Type,
		&t.Amount,
		&t.Status,
		&t.CreatedBy,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &t, nil
}

func (s *Store) GetTransactionByID(ctx context.Context, id uuid.UUID) (*types.Transaction, error) {
	query := `
		SELECT id, company_id, reference, type, amount, status, created_by, created_at
		FROM transactions
		WHERE id = $1
	`

	var t types.Transaction
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.CompanyID,
		&t.Reference,
		&t.Type,
		&t.Amount,
		&t.Status,
		&t.CreatedBy,
		&t.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &t, nil
}

func (s *Store) GetTransactionsByCompany(ctx context.Context, companyID uuid.UUID) ([]*types.Transaction, error) {
	query := `
		SELECT id, company_id, reference, type, amount, status, created_by, created_at
		FROM transactions
		WHERE company_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var txns []*types.Transaction
	for rows.Next() {
		var t types.Transaction
		err := rows.Scan(
			&t.ID,
			&t.CompanyID,
			&t.Reference,
			&t.Type,
			&t.Amount,
			&t.Status,
			&t.CreatedBy,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		txns = append(txns, &t)
	}
	return txns, nil
}

func (s *Store) UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status types.TransactionStatus) error {
	query := `
		UPDATE transactions
		SET status = $1
		WHERE id = $2
	`
	res, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
