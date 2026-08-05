package payments

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

func (s *Store) CreatePaymentAttempt(ctx context.Context, attempt *types.PaymentAttempt) (*types.PaymentAttempt, error) {
	stmt := `
		INSERT INTO payment_attempts (
			id, transaction_id, psp, status, external_reference, retry_count, next_retry_at, response, created_at
		) VALUES (
			COALESCE(NULLIF($1, '00000000-0000-0000-0000-000000000000'::uuid), gen_random_uuid()),
			$2, $3, $4, $5, $6, $7, $8, NOW()
		) 
		RETURNING id, transaction_id, psp, status, external_reference, retry_count, next_retry_at, response, created_at
	`

	var result types.PaymentAttempt
	err := s.db.QueryRowContext(
		ctx,
		stmt,
		attempt.ID,
		attempt.TransactionID,
		attempt.PSP,
		attempt.Status,
		attempt.ExternalReference,
		attempt.RetryCount,
		attempt.NextRetryAt,
		attempt.Response,
	).Scan(
		&result.ID,
		&result.TransactionID,
		&result.PSP,
		&result.Status,
		&result.ExternalReference,
		&result.RetryCount,
		&result.NextRetryAt,
		&result.Response,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Store) UpdatePaymentAttemptStatus(ctx context.Context, attempt *types.PaymentAttempt) (*types.PaymentAttempt, error) {
	stmt := `
		UPDATE payment_attempts 
		SET 
			status = $1, 
			external_reference = $2, 
			retry_count = $3, 
			next_retry_at = $4, 
			response = $5, 
			updated_at = NOW() 
		WHERE id = $6 
		RETURNING id, transaction_id, psp, status, external_reference, retry_count, next_retry_at, response, created_at
	`
	row := s.db.QueryRowContext(ctx, stmt, attempt.Status, attempt.ExternalReference, attempt.RetryCount, attempt.NextRetryAt, attempt.Response, attempt.ID)

	var result types.PaymentAttempt
	err := row.Scan(&result.ID, &result.TransactionID, &result.PSP, &result.Status, &result.ExternalReference, &result.RetryCount, &result.NextRetryAt, &result.Response, &result.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Store) GetPaymentAttemptsByTransactionID(ctx context.Context, txnID uuid.UUID) ([]*types.PaymentAttempt, error) {
	stmt := `
		SELECT 
			id,
			transaction_id,
			psp,
			status,
			external_reference,
			retry_count,
			next_retry_at,
			response,
			created_at
		FROM payment_attempts
		WHERE transaction_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, stmt, txnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []*types.PaymentAttempt
	for rows.Next() {
		var attempt types.PaymentAttempt
		err := rows.Scan(
			&attempt.ID,
			&attempt.TransactionID,
			&attempt.PSP,
			&attempt.Status,
			&attempt.ExternalReference,
			&attempt.RetryCount,
			&attempt.NextRetryAt,
			&attempt.Response,
			&attempt.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, &attempt)
	}
	return attempts, nil

}
