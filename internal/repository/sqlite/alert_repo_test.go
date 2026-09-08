package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestAlertRepository_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_alerts.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewAlertRepository(db.DB)

	// 1. GetSettings (Seeded default)
	settings, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.ID != 1 {
		t.Errorf("expected ID 1, got %d", settings.ID)
	}
	if settings.TelegramEnabled || settings.DiscordEnabled {
		t.Errorf("expected default alerts to be disabled")
	}
	if !settings.TriggerOnCrash || !settings.TriggerOnThermal || !settings.TriggerOnLowStorage {
		t.Errorf("expected default triggers to be true")
	}
	if settings.ThermalThresholdC != 48.0 {
		t.Errorf("expected thermal threshold 48.0, got %f", settings.ThermalThresholdC)
	}
	if settings.LowStorageThresholdGB != 5.0 {
		t.Errorf("expected low storage threshold 5.0, got %f", settings.LowStorageThresholdGB)
	}

	// 2. UpdateSettings
	now := time.Now().Truncate(time.Second)
	settings.TelegramEnabled = true
	settings.TelegramBotToken = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	settings.TelegramChatID = "-1001234567890"
	settings.DiscordEnabled = true
	settings.DiscordWebhookURL = "https://discord.com/api/webhooks/test/123"
	settings.TriggerOnCrash = true
	settings.TriggerOnThermal = false
	settings.ThermalThresholdC = 50.0
	settings.TriggerOnLowStorage = true
	settings.LowStorageThresholdGB = 3.0
	settings.LastAlertSentAt = &now

	if err := repo.UpdateSettings(ctx, settings); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// 3. GetSettings after update
	updated, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings after update failed: %v", err)
	}
	if !updated.TelegramEnabled || !updated.DiscordEnabled {
		t.Errorf("expected telegram and discord to be enabled")
	}
	if updated.TelegramBotToken != "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" {
		t.Errorf("telegram bot token mismatch, got %s", updated.TelegramBotToken)
	}
	if updated.TelegramChatID != "-1001234567890" {
		t.Errorf("telegram chat id mismatch, got %s", updated.TelegramChatID)
	}
	if updated.DiscordWebhookURL != "https://discord.com/api/webhooks/test/123" {
		t.Errorf("discord webhook url mismatch, got %s", updated.DiscordWebhookURL)
	}
	if updated.TriggerOnThermal {
		t.Errorf("expected TriggerOnThermal to be false")
	}
	if updated.ThermalThresholdC != 50.0 {
		t.Errorf("expected thermal threshold 50.0, got %f", updated.ThermalThresholdC)
	}
	if updated.LowStorageThresholdGB != 3.0 {
		t.Errorf("expected low storage threshold 3.0, got %f", updated.LowStorageThresholdGB)
	}
	if updated.LastAlertSentAt == nil {
		t.Errorf("expected LastAlertSentAt to not be nil")
	}

	// 4. Update nil settings error check
	if err := repo.UpdateSettings(ctx, nil); err == nil {
		t.Errorf("expected error when updating nil settings, got nil")
	}
}
