package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-streamer/internal/alert"
	"go-streamer/internal/domain"
)

type AlertRepository interface {
	GetSettings(ctx context.Context) (*domain.AlertSettings, error)
	UpdateSettings(ctx context.Context, settings *domain.AlertSettings) error
}

type AlertDispatcher interface {
	Dispatch(ctx context.Context, event *domain.AlertEvent) error
}

type AlertService struct {
	repo       AlertRepository
	dispatcher AlertDispatcher
}

func NewAlertService(repo AlertRepository, dispatcher AlertDispatcher) *AlertService {
	return &AlertService{
		repo:       repo,
		dispatcher: dispatcher,
	}
}

// GetSettings mengembalikan pengaturan notifikasi remote saat ini.
func (s *AlertService) GetSettings(ctx context.Context) (*domain.AlertSettings, error) {
	return s.repo.GetSettings(ctx)
}

// UpdateSettings menyimpan konfigurasi alert terbaru.
func (s *AlertService) UpdateSettings(ctx context.Context, settings *domain.AlertSettings) error {
	if settings == nil {
		return errors.New("alert settings cannot be nil")
	}
	return s.repo.UpdateSettings(ctx, settings)
}

// SendTestAlert mengirimkan sample test alert bertema Authentic Neobrutalism ke channel yang aktif.
func (s *AlertService) SendTestAlert(ctx context.Context) error {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve alert settings: %w", err)
	}

	if !settings.TelegramEnabled && !settings.DiscordEnabled {
		return errors.New("cannot send test alert: neither Telegram nor Discord is enabled")
	}

	testEvent := &domain.AlertEvent{
		Type:      "test",
		Level:     "info",
		Title:     "⚡ [GO-STREAMER] AUTHENTIC NEOBRUTALISM TEST",
		Message:   "SYSTEM STATUS: ONLINE\nDISPATCHER: ACTIVE\nWEBHOOK VERIFICATION: 200 OK\n━━━━━━━━━━━━━━━━━━━━━\nRemote notifications are configured and operational!",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"environment": "go-streamer-embedded",
			"verified_at": time.Now().Format(time.RFC3339),
		},
	}

	if s.dispatcher == nil {
		return errors.New("alert dispatcher is not configured")
	}

	if err := s.dispatcher.Dispatch(ctx, testEvent); err != nil {
		if errors.Is(err, alert.ErrCooldownActive) {
			return errors.New("test alert suppressed: cooldown active")
		}
		return fmt.Errorf("failed to dispatch test alert: %w", err)
	}

	return nil
}
