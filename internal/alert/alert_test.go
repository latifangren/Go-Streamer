package alert

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

type mockAlertRepo struct {
	mu       sync.Mutex
	settings *domain.AlertSettings
}

func newMockAlertRepo(settings *domain.AlertSettings) *mockAlertRepo {
	return &mockAlertRepo{
		settings: settings,
	}
}

func (m *mockAlertRepo) GetSettings(ctx context.Context) (*domain.AlertSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.settings == nil {
		return nil, domain.ErrNotFound
	}
	clone := *m.settings
	return &clone, nil
}

func (m *mockAlertRepo) UpdateSettings(ctx context.Context, s *domain.AlertSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings = s
	return nil
}

// ----------------------------------------------------------------------------
// 1. Telegram Client Tests
// ----------------------------------------------------------------------------

func TestTelegramClient_SendMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedPayload telegramPayload
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/botTOKEN123/sendMessage" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedPayload)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true, "result": {}}`))
		}))
		defer ts.Close()

		client := NewTelegramClient(ts.URL)
		err := client.SendMessage(context.Background(), "TOKEN123", "CHAT456", "Hello from Go-Streamer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if receivedPayload.ChatID != "CHAT456" {
			t.Errorf("expected chat id CHAT456, got %s", receivedPayload.ChatID)
		}
		if receivedPayload.Text != "Hello from Go-Streamer" {
			t.Errorf("expected text Hello from Go-Streamer, got %s", receivedPayload.Text)
		}
	})

	t.Run("api error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"ok": false, "description": "Chat not found"}`))
		}))
		defer ts.Close()

		client := NewTelegramClient(ts.URL)
		err := client.SendMessage(context.Background(), "TOKEN", "INVALID", "Test")
		if err == nil {
			t.Fatalf("expected error from telegram client, got nil")
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		client := NewTelegramClient()
		if err := client.SendMessage(context.Background(), "", "chat", "text"); err == nil {
			t.Errorf("expected error for empty bot token")
		}
		if err := client.SendMessage(context.Background(), "token", "", "text"); err == nil {
			t.Errorf("expected error for empty chat id")
		}
		if err := client.SendMessage(context.Background(), "token", "chat", ""); err == nil {
			t.Errorf("expected error for empty text")
		}
	})
}

// ----------------------------------------------------------------------------
// 2. Discord Client Tests
// ----------------------------------------------------------------------------

func TestDiscordClient_SendEmbed(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var receivedPayload discordWebhookPayload
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedPayload)

			w.WriteHeader(http.StatusNoContent) // Discord standard 204
		}))
		defer ts.Close()

		client := NewDiscordClient()
		err := client.SendEmbed(context.Background(), ts.URL, "Title Test", "Desc Test", 0xEF4444)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(receivedPayload.Embeds) != 1 {
			t.Fatalf("expected 1 embed, got %d", len(receivedPayload.Embeds))
		}
		embed := receivedPayload.Embeds[0]
		if embed.Title != "Title Test" || embed.Description != "Desc Test" || embed.Color != 0xEF4444 {
			t.Errorf("embed mismatch: %+v", embed)
		}
	})

	t.Run("webhook error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Unknown Webhook"}`))
		}))
		defer ts.Close()

		client := NewDiscordClient()
		err := client.SendEmbed(context.Background(), ts.URL, "Title", "Desc", 0)
		if err == nil {
			t.Fatalf("expected error from discord webhook, got nil")
		}
	})

	t.Run("empty webhook url", func(t *testing.T) {
		client := NewDiscordClient()
		err := client.SendEmbed(context.Background(), "", "Title", "Desc", 0)
		if err == nil {
			t.Errorf("expected error for empty webhook url")
		}
	})
}

// ----------------------------------------------------------------------------
// 3. Dispatcher Tests (Filters, Anti-Flood Cooldown, Helpers)
// ----------------------------------------------------------------------------

func TestAlertDispatcher_Dispatch(t *testing.T) {
	t.Run("disabled channels return ErrAlertsDisabled", func(t *testing.T) {
		settings := &domain.AlertSettings{
			TelegramEnabled: false,
			DiscordEnabled:  false,
			TriggerOnCrash:  true,
		}
		repo := newMockAlertRepo(settings)
		dispatcher := NewAlertDispatcher(repo, nil, nil)

		event := &domain.AlertEvent{
			Type:    "crash",
			Level:   "critical",
			Title:   "Crash",
			Message: "Crash msg",
		}

		err := dispatcher.Dispatch(context.Background(), event)
		if !errors.Is(err, ErrAlertsDisabled) {
			t.Errorf("expected ErrAlertsDisabled, got %v", err)
		}
	})

	t.Run("trigger flag disabled skips silently", func(t *testing.T) {
		settings := &domain.AlertSettings{
			TelegramEnabled:  true,
			TelegramBotToken: "tok",
			TelegramChatID:   "chat",
			TriggerOnCrash:   false, // Dinonaktifkan
		}
		repo := newMockAlertRepo(settings)
		dispatcher := NewAlertDispatcher(repo, nil, nil)

		event := &domain.AlertEvent{
			Type:    "crash",
			Level:   "critical",
			Title:   "Crash",
			Message: "Crash msg",
		}

		err := dispatcher.Dispatch(context.Background(), event)
		if err != nil {
			t.Errorf("expected nil for disabled trigger, got %v", err)
		}
	})

	t.Run("successful dispatch to telegram and discord with anti-flood cooldown", func(t *testing.T) {
		var tgCallCount, discordCallCount int
		var mu sync.Mutex

		tgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			tgCallCount++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true}`))
		}))
		defer tgServer.Close()

		discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			discordCallCount++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}))
		defer discordServer.Close()

		settings := &domain.AlertSettings{
			TelegramEnabled:   true,
			TelegramBotToken:  "test-bot-token",
			TelegramChatID:    "test-chat-id",
			DiscordEnabled:    true,
			DiscordWebhookURL: discordServer.URL,
			TriggerOnCrash:    true,
			TriggerOnThermal:  true,
		}
		repo := newMockAlertRepo(settings)

		tgClient := NewTelegramClient(tgServer.URL)
		discordClient := NewDiscordClient()

		dispatcher := NewAlertDispatcher(repo, tgClient, discordClient)
		dispatcher.SetCooldownDuration(500 * time.Millisecond)

		// 1. Dispatch event pertama -> harus sukses
		event := &domain.AlertEvent{
			Type:      "crash",
			Level:     "critical",
			Title:     "Crash 1",
			Message:   "Slot 1 died",
			Timestamp: time.Now(),
		}

		err := dispatcher.Dispatch(context.Background(), event)
		if err != nil {
			t.Fatalf("expected dispatch success, got: %v", err)
		}

		mu.Lock()
		if tgCallCount != 1 || discordCallCount != 1 {
			t.Errorf("expected 1 tg call and 1 discord call, got tg=%d, discord=%d", tgCallCount, discordCallCount)
		}
		mu.Unlock()

		// 2. Dispatch event kedua dalam rentang cooldown -> harus ErrCooldownActive
		err = dispatcher.Dispatch(context.Background(), event)
		if !errors.Is(err, ErrCooldownActive) {
			t.Errorf("expected ErrCooldownActive, got: %v", err)
		}

		// 3. Dispatch event tipe lain (thermal) -> harus tembus karena per-type cooldown
		thermalEvent := &domain.AlertEvent{
			Type:      "thermal",
			Level:     "warning",
			Title:     "Overheat",
			Message:   "52C",
			Timestamp: time.Now(),
		}
		err = dispatcher.Dispatch(context.Background(), thermalEvent)
		if err != nil {
			t.Fatalf("expected thermal dispatch success, got: %v", err)
		}

		// 4. Test event bypass cooldown
		testEvent := &domain.AlertEvent{
			Type:      "test",
			Level:     "info",
			Title:     "Test",
			Message:   "Test msg",
			Timestamp: time.Now(),
		}
		err = dispatcher.Dispatch(context.Background(), testEvent)
		if err != nil {
			t.Fatalf("expected test event to bypass cooldown, got: %v", err)
		}

		// 5. Tunggu cooldown selesai untuk tipe crash
		time.Sleep(600 * time.Millisecond)
		err = dispatcher.Dispatch(context.Background(), event)
		if err != nil {
			t.Fatalf("expected crash dispatch success after cooldown, got: %v", err)
		}
	})

	t.Run("helper methods", func(t *testing.T) {
		settings := &domain.AlertSettings{
			TelegramEnabled:     true,
			TelegramBotToken:    "token",
			TelegramChatID:      "chat",
			TriggerOnCrash:      true,
			TriggerOnThermal:    true,
			TriggerOnLowStorage: true,
		}
		repo := newMockAlertRepo(settings)

		var lastMsg string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var p telegramPayload
			_ = json.Unmarshal(body, &p)
			lastMsg = p.Text
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok": true}`))
		}))
		defer ts.Close()

		dispatcher := NewAlertDispatcher(repo, NewTelegramClient(ts.URL), nil)
		dispatcher.SetCooldownDuration(1 * time.Millisecond)

		// Test TriggerCrash
		if err := dispatcher.TriggerCrash(context.Background(), 1, "Slot YouTube", "exit status 1"); err != nil {
			t.Fatalf("TriggerCrash error: %v", err)
		}
		time.Sleep(10 * time.Millisecond)

		// Test TriggerThermal
		if err := dispatcher.TriggerThermal(context.Background(), 52.5, 48.0); err != nil {
			t.Fatalf("TriggerThermal error: %v", err)
		}
		time.Sleep(10 * time.Millisecond)

		// Test TriggerLowStorage
		if err := dispatcher.TriggerLowStorage(context.Background(), 2.5, 5.0); err != nil {
			t.Fatalf("TriggerLowStorage error: %v", err)
		}

		if lastMsg == "" {
			t.Errorf("expected last message to be sent")
		}
	})
}
