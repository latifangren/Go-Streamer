package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-streamer/internal/domain"
)

// TunnelRepository mengimplementasikan domain.TunnelRepository untuk tabel tunnel_settings di SQLite.
type TunnelRepository struct {
	db *sql.DB
}

// NewTunnelRepository membuat instans baru TunnelRepository.
func NewTunnelRepository(db *sql.DB) *TunnelRepository {
	return &TunnelRepository{db: db}
}

// GetSettings mengambil baris konfigurasi tunnel tunggal (id = 1).
func (r *TunnelRepository) GetSettings(ctx context.Context) (*domain.TunnelSettings, error) {
	query := `
		SELECT id, provider, mode, tunnel_token, is_active, public_url, last_status, updated_at
		FROM tunnel_settings
		WHERE id = 1
	`

	var s domain.TunnelSettings
	err := r.db.QueryRowContext(ctx, query).Scan(
		&s.ID,
		&s.Provider,
		&s.Mode,
		&s.TunnelToken,
		&s.IsActive,
		&s.PublicURL,
		&s.LastStatus,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query tunnel settings: %w", err)
	}

	return &s, nil
}

// UpdateSettings memperbarui atau meng-upsert baris konfigurasi tunnel tunggal (id = 1).
func (r *TunnelRepository) UpdateSettings(ctx context.Context, s *domain.TunnelSettings) error {
	if s == nil {
		return fmt.Errorf("tunnel settings cannot be nil: %w", domain.ErrInvalidInput)
	}

	now := time.Now().UTC()
	s.UpdatedAt = now

	query := `
		INSERT INTO tunnel_settings (id, provider, mode, tunnel_token, is_active, public_url, last_status, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider = excluded.provider,
			mode = excluded.mode,
			tunnel_token = excluded.tunnel_token,
			is_active = excluded.is_active,
			public_url = excluded.public_url,
			last_status = excluded.last_status,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		s.Provider,
		s.Mode,
		s.TunnelToken,
		s.IsActive,
		s.PublicURL,
		s.LastStatus,
		s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update tunnel settings: %w", err)
	}

	return nil
}
