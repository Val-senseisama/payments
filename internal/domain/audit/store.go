package audit

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

func (s *Store) CreateAuditLog(ctx context.Context, auditLog *types.AuditLog) error {
	query := `
		INSERT INTO audit_log (company_id, entity_type, entity_id, action, changed_by, new_values)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := s.db.QueryRowContext(
		ctx,
		query,
		auditLog.CompanyID,
		auditLog.EntityType,
		auditLog.EntityID,
		auditLog.Action,
		auditLog.ChangedBy,
		auditLog.NewValues,
	).Scan(&auditLog.ID, &auditLog.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

func (s *Store) GetAuditLogsByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*types.AuditLog, error) {
	query := `
		SELECT id, company_id, entity_type, entity_id, action, changed_by, new_values, created_at
		FROM audit_log
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*types.AuditLog
	for rows.Next() {
		var l types.AuditLog
		err := rows.Scan(
			&l.ID,
			&l.CompanyID,
			&l.EntityType,
			&l.EntityID,
			&l.Action,
			&l.ChangedBy,
			&l.NewValues,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log row: %w", err)
		}
		logs = append(logs, &l)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit log rows: %w", err)
	}
	return logs, nil
}

func (s *Store) GetAuditLogsByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*types.AuditLog, error) {
	query := `
		SELECT id, company_id, entity_type, entity_id, action, changed_by, new_values, created_at
		FROM audit_log
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs by company: %w", err)
	}
	defer rows.Close()

	var logs []*types.AuditLog
	for rows.Next() {
		var l types.AuditLog
		err := rows.Scan(
			&l.ID,
			&l.CompanyID,
			&l.EntityType,
			&l.EntityID,
			&l.Action,
			&l.ChangedBy,
			&l.NewValues,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log row: %w", err)
		}
		logs = append(logs, &l)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit log rows: %w", err)
	}
	return logs, nil
}
