package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-streamer/internal/domain"
)

type AlertRepository struct {
	db *sql.DB
}

func NewAlertRepository(db *sql.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// GetSettings mengambil pengaturan alert (ID = 1).
func (r *AlertRepository) GetSettings(ctx context.Context) (*domain.AlertSettings, error) {
	query := `
		SELECT id, telegram_enabled, telegram_bot_token, telegram_chat_id,
		       discord_enabled, discord_webhook_url,
		       trigger_on_crash, trigger_on_thermal, thermal_threshold_c,
		       trigger_on_low_storage, low_storage_threshold_gb,
		       last_alert_sent_at, updated_at
		FROM alert_settings
		WHERE id = 1
	`

	row := r.db.QueryRowContext(ctx, query)

	var s domain.AlertSettings
	var lastAlertSentAt sql.NullTime

	err := row.Scan(
		&s.ID,
		&s.TelegramEnabled,
		&s.TelegramBotToken,
		&s.TelegramChatID,
		&s.DiscordEnabled,
		&s.DiscordWebhookURL,
		&s.TriggerOnCrash,
		&s.TriggerOnThermal,
		&s.ThermalThresholdC,
		&s.TriggerOnLowStorage,
		&s.LowStorageThresholdGB,
		&lastAlertSentAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("alert settings not found: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to query alert settings: %w", err)
	}

	if lastAlertSentAt.Valid {
		s.LastAlertSentAt = &lastAlertSentAt.Time
	}

	return &s, nil
}

// UpdateSettings memperbarui konfigurasi alert_settings untuk ID = 1.
func (r *AlertRepository) UpdateSettings(ctx context.Context, s *domain.AlertSettings) error {
	if s == nil {
		return errors.New("alert settings cannot be nil")
	}

	query := `
		UPDATE alert_settings
		SET telegram_enabled = ?,
		    telegram_bot_token = ?,
		    telegram_chat_id = ?,
		    discord_enabled = ?,
		    discord_webhook_url = ?,
		    trigger_on_crash = ?,
		    trigger_on_thermal = ?,
		    thermal_threshold_c = ?,
		    trigger_on_low_storage = ?,
		    low_storage_threshold_gb = ?,
		    last_alert_sent_at = ?,
		    updated_at = ?
		WHERE id = 1
	`

	s.UpdatedAt = time.Now()

	res, err := r.db.ExecContext(ctx, query,
		s.TelegramEnabled,
		s.TelegramBotToken,
		s.TelegramChatID,
		s.DiscordEnabled,
		s.DiscordWebhookURL,
		s.TriggerOnCrash,
		s.TriggerOnThermal,
		s.ThermalThresholdC,
		s.TriggerOnLowStorage,
		s.LowStorageThresholdGB,
		s.LastAlertSentAt,
		s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update alert settings: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("alert settings with id 1 not found: %w", domain.ErrNotFound)
	}

	return nil
}
