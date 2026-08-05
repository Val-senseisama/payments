package ledger

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

func (s *Store) PostTransaction(ctx context.Context, txnID, companyID, debitAccountID, creditAccountID uuid.UUID, amount int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin ledger posting transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Prevent double posting (DB level unique constraint on ledger_postings.transaction_id)
	postingQuery := `INSERT INTO ledger_postings (transaction_id) VALUES ($1)`
	if _, err := tx.ExecContext(ctx, postingQuery, txnID); err != nil {
		return fmt.Errorf("transaction already posted or invalid: %w", err)
	}

	// 2. Bulk insert debit & credit entries in a single SQL roundtrip
	entriesQuery := `
		INSERT INTO entries (transaction_id, account_id, amount, direction)
		VALUES 
			($1, $2, $3, $4),
			($1, $5, $3, $6)
	`
	if _, err := tx.ExecContext(ctx, entriesQuery, txnID, debitAccountID, amount, types.EntryDebit, creditAccountID, types.EntryCredit); err != nil {
		return fmt.Errorf("failed to insert ledger entries: %w", err)
	}

	// 3. Update cached balance for debit account (Asset/Expense increase on Debit; Liability/Revenue decrease)
	debitUpdateQuery := `
		UPDATE accounts 
		SET cached_balance = CASE 
			WHEN type IN ('asset', 'expense') THEN cached_balance + $1 
			ELSE cached_balance - $1 
		END 
		WHERE id = $2 AND company_id = $3 
		  AND (type IN ('asset', 'expense') OR cached_balance >= $1)
	`
	res, err := tx.ExecContext(ctx, debitUpdateQuery, amount, debitAccountID, companyID)
	if err != nil {
		return fmt.Errorf("failed to update debit account balance: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("insufficient funds or debit account does not belong to company")
	}

	// 4. Update cached balance for credit account (Liability/Revenue increase on Credit; Asset/Expense decrease)
	creditUpdateQuery := `
		UPDATE accounts 
		SET cached_balance = CASE 
			WHEN type IN ('liability', 'revenue') THEN cached_balance + $1 
			ELSE cached_balance - $1 
		END 
		WHERE id = $2 AND company_id = $3 
		  AND (type IN ('liability', 'revenue') OR cached_balance >= $1)
	`
	res, err = tx.ExecContext(ctx, creditUpdateQuery, amount, creditAccountID, companyID)
	if err != nil {
		return fmt.Errorf("failed to update credit account balance: %w", err)
	}
	rows, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("insufficient funds or credit account does not belong to company")
	}

	// 5. Update transaction status to completed
	updateTxnQuery := `UPDATE transactions SET status = 'completed' WHERE id = $1 AND company_id = $2`
	if _, err := tx.ExecContext(ctx, updateTxnQuery, txnID, companyID); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return tx.Commit()
}

func (s *Store) GetEntriesByTransaction(ctx context.Context, txnID uuid.UUID) ([]*types.Entry, error) {
	query := `
		SELECT id, transaction_id, account_id, amount, direction, created_at
		FROM entries
		WHERE transaction_id = $1
		ORDER BY created_at ASC
	`
	rows, err := s.db.QueryContext(ctx, query, txnID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []*types.Entry
	for rows.Next() {
		var e types.Entry
		err := rows.Scan(
			&e.ID,
			&e.TransactionID,
			&e.AccountID,
			&e.Amount,
			&e.Direction,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, &e)
	}
	return entries, nil
}
