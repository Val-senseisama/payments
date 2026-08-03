package profiles

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

func (s *Store) CreateProfile(ctx context.Context, companyID uuid.UUID, pType types.ProfileType, name string, email, phone, avatarURL *string, metadata []byte) (*types.Profile, error) {
	query := `
		INSERT INTO profiles (company_id, type, name, email, phone, avatar_url, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, company_id, type, name, email, phone, avatar_url, metadata, created_at, deleted_at
	`
	var p types.Profile
	err := s.db.QueryRowContext(ctx, query, companyID, string(pType), name, email, phone, avatarURL, metadata).Scan(
		&p.ID,
		&p.CompanyID,
		&p.Type,
		&p.Name,
		&p.Email,
		&p.Phone,
		&p.AvatarURL,
		&p.Metadata,
		&p.CreatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}
	return &p, nil
}

func (s *Store) GetProfileByID(ctx context.Context, id uuid.UUID) (*types.Profile, error) {
	query := `
		SELECT id, company_id, type, name, email, phone, avatar_url, metadata, created_at, deleted_at
		FROM profiles
		WHERE id = $1 AND deleted_at IS NULL
	`
	var p types.Profile
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.CompanyID,
		&p.Type,
		&p.Name,
		&p.Email,
		&p.Phone,
		&p.AvatarURL,
		&p.Metadata,
		&p.CreatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found")
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &p, nil
}

func (s *Store) GetProfilesByCompany(ctx context.Context, companyID uuid.UUID, pType *types.ProfileType) ([]*types.Profile, error) {
	query := `
		SELECT id, company_id, type, name, email, phone, avatar_url, metadata, created_at, deleted_at
		FROM profiles
		WHERE company_id = $1 AND deleted_at IS NULL
	`
	args := []any{companyID}

	if pType != nil && *pType != "" {
		query += " AND type = $2"
		args = append(args, string(*pType))
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*types.Profile
	for rows.Next() {
		var p types.Profile
		err := rows.Scan(
			&p.ID,
			&p.CompanyID,
			&p.Type,
			&p.Name,
			&p.Email,
			&p.Phone,
			&p.AvatarURL,
			&p.Metadata,
			&p.CreatedAt,
			&p.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile row: %w", err)
		}
		profiles = append(profiles, &p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating profile rows: %w", err)
	}
	return profiles, nil
}

func (s *Store) UpdateProfile(ctx context.Context, id uuid.UUID, name *string, email, phone, avatarURL *string) (*types.Profile, error) {
	query := `
		UPDATE profiles
		SET 
			name = COALESCE($1, name),
			email = COALESCE($2, email),
			phone = COALESCE($3, phone),
			avatar_url = COALESCE($4, avatar_url)
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING id, company_id, type, name, email, phone, avatar_url, metadata, created_at, deleted_at
	`
	var p types.Profile
	err := s.db.QueryRowContext(ctx, query, name, email, phone, avatarURL, id).Scan(
		&p.ID,
		&p.CompanyID,
		&p.Type,
		&p.Name,
		&p.Email,
		&p.Phone,
		&p.AvatarURL,
		&p.Metadata,
		&p.CreatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found")
		}
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	return &p, nil
}

func (s *Store) DeleteProfile(ctx context.Context, id uuid.UUID) (*types.Profile, error) {
	query := `
		UPDATE profiles
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, company_id, type, name, email, phone, avatar_url, metadata, created_at, deleted_at
	`
	var p types.Profile
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.CompanyID,
		&p.Type,
		&p.Name,
		&p.Email,
		&p.Phone,
		&p.AvatarURL,
		&p.Metadata,
		&p.CreatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile not found")
		}
		return nil, fmt.Errorf("failed to soft delete profile: %w", err)
	}
	return &p, nil
}
