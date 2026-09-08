package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go-streamer/internal/domain"
)

type mockAlertServiceRepo struct {
	mu       sync.Mutex
	settings *domain.AlertSettings
}

func newMockAlertServiceRepo(settings *domain.AlertSettings) *mockAlertServiceRepo {
	return &mockAlertServiceRepo{settings: settings}
}

func (m *mockAlertServiceRepo) GetSettings(ctx context.Context) (*domain.AlertSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.settings == nil {
		return nil, domain.ErrNotFound
	}
	clone := *m.settings
	return &clone, nil
}

func (m *mockAlertServiceRepo) UpdateSettings(ctx context.Context, settings *domain.AlertSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings = settings
	return nil
}

type mockAlertServiceDispatcher struct {
	mu              sync.Mutex
	dispatchedEvent *domain.AlertEvent
	failDispatch    bool
}

func (m *mockAlertServiceDispatcher) Dispatch(ctx context.Context, event *domain.AlertEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failDispatch {
		return errors.New("simulated dispatcher failure")
	}
	m.dispatchedEvent = event
	return nil
}

func TestAlertService_GetAndUpdateSettings(t *testing.T) {
	repo := newMockAlertServiceRepo(&domain.AlertSettings{
		ID:              1,
		TelegramEnabled: true,
	})
	dispatcher := &mockAlertServiceDispatcher{}
	svc := NewAlertService(repo, dispatcher)

	// 1. GetSettings
	s, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if !s.TelegramEnabled {
		t.Errorf("expected TelegramEnabled true")
	}

	// 2. UpdateSettings
	s.TelegramEnabled = false
	s.DiscordEnabled = true
	if err := svc.UpdateSettings(context.Background(), s); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	updated, err := svc.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings after update failed: %v", err)
	}
	if updated.TelegramEnabled || !updated.DiscordEnabled {
		t.Errorf("expected TelegramEnabled false and DiscordEnabled true")
	}

	// 3. Update nil settings error
	if err := svc.UpdateSettings(context.Background(), nil); err == nil {
		t.Errorf("expected error for nil settings")
	}
}

func TestAlertService_SendTestAlert(t *testing.T) {
	t.Run("fails when both disabled", func(t *testing.T) {
		repo := newMockAlertServiceRepo(&domain.AlertSettings{
			TelegramEnabled: false,
			DiscordEnabled:  false,
		})
		dispatcher := &mockAlertServiceDispatcher{}
		svc := NewAlertService(repo, dispatcher)

		err := svc.SendTestAlert(context.Background())
		if err == nil {
			t.Fatalf("expected error when neither channel is enabled")
		}
	})

	t.Run("succeeds when enabled", func(t *testing.T) {
		repo := newMockAlertServiceRepo(&domain.AlertSettings{
			TelegramEnabled: true,
			DiscordEnabled:  false,
		})
		dispatcher := &mockAlertServiceDispatcher{}
		svc := NewAlertService(repo, dispatcher)

		err := svc.SendTestAlert(context.Background())
		if err != nil {
			t.Fatalf("unexpected error sending test alert: %v", err)
		}

		if dispatcher.dispatchedEvent == nil {
			t.Fatalf("expected dispatcher to receive event")
		}
		if dispatcher.dispatchedEvent.Type != "test" {
			t.Errorf("expected event type test, got %s", dispatcher.dispatchedEvent.Type)
		}
	})
}
