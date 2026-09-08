package alert

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-streamer/internal/domain"
)

var (
	ErrAlertsDisabled = errors.New("all alert channels are disabled")
	ErrCooldownActive = errors.New("alert suppressed due to anti-flood cooldown")
)

type AlertSettingsRepository interface {
	GetSettings(ctx context.Context) (*domain.AlertSettings, error)
	UpdateSettings(ctx context.Context, settings *domain.AlertSettings) error
}

type AlertDispatcher struct {
	repo             AlertSettingsRepository
	telegram         *TelegramClient
	discord          *DiscordClient
	cooldownDuration time.Duration
	lastSent         map[string]time.Time
	mu               sync.Mutex
}

func NewAlertDispatcher(repo AlertSettingsRepository, telegram *TelegramClient, discord *DiscordClient) *AlertDispatcher {
	return &AlertDispatcher{
		repo:             repo,
		telegram:         telegram,
		discord:          discord,
		cooldownDuration: 5 * time.Minute,
		lastSent:         make(map[string]time.Time),
	}
}

// SetCooldownDuration mengubah durasi anti-flood (berguna untuk testing).
func (d *AlertDispatcher) SetCooldownDuration(dur time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cooldownDuration = dur
}

// Dispatch memvalidasi status alert, menerapkan proteksi anti-flood 5 menit per tipe event,
// lalu mengirimkan notifikasi ke Telegram dan/atau Discord yang aktif.
func (d *AlertDispatcher) Dispatch(ctx context.Context, event *domain.AlertEvent) error {
	if event == nil {
		return errors.New("alert event cannot be nil")
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	settings, err := d.repo.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to load alert settings: %w", err)
	}

	// 1. Cek trigger filter sesuai konfigurasi
	switch event.Type {
	case "crash":
		if !settings.TriggerOnCrash {
			return nil // Dilewati karena trigger dinonaktifkan
		}
	case "thermal":
		if !settings.TriggerOnThermal {
			return nil
		}
	case "low_storage":
		if !settings.TriggerOnLowStorage {
			return nil
		}
	}

	// 2. Cek apakah ada channel notifikasi yang aktif
	if !settings.TelegramEnabled && !settings.DiscordEnabled {
		return ErrAlertsDisabled
	}

	// 3. Mekanisme Anti-Flood Cooldown (Kecuali event berjenis "test")
	if event.Type != "test" {
		d.mu.Lock()
		if lastTime, exists := d.lastSent[event.Type]; exists {
			if time.Since(lastTime) < d.cooldownDuration {
				d.mu.Unlock()
				return ErrCooldownActive
			}
		}
		d.mu.Unlock()
	}

	var dispatchErrors []string
	sentSuccessfully := false

	// 4. Kirim ke Telegram jika aktif
	if settings.TelegramEnabled && d.telegram != nil && settings.TelegramBotToken != "" && settings.TelegramChatID != "" {
		tgMsg := formatTelegramMessage(event)
		if err := d.telegram.SendMessage(ctx, settings.TelegramBotToken, settings.TelegramChatID, tgMsg); err != nil {
			dispatchErrors = append(dispatchErrors, fmt.Sprintf("telegram: %v", err))
		} else {
			sentSuccessfully = true
		}
	}

	// 5. Kirim ke Discord jika aktif
	if settings.DiscordEnabled && d.discord != nil && settings.DiscordWebhookURL != "" {
		colorHex := getDiscordColor(event.Level)
		desc := formatDiscordDescription(event)
		if err := d.discord.SendEmbed(ctx, settings.DiscordWebhookURL, event.Title, desc, colorHex); err != nil {
			dispatchErrors = append(dispatchErrors, fmt.Sprintf("discord: %v", err))
		} else {
			sentSuccessfully = true
		}
	}

	// 6. Jika berhasil terkirim minimal ke salah satu channel, catat waktu kirim
	if sentSuccessfully {
		now := time.Now()
		d.mu.Lock()
		d.lastSent[event.Type] = now
		d.mu.Unlock()

		settings.LastAlertSentAt = &now
		_ = d.repo.UpdateSettings(ctx, settings)
	}

	if len(dispatchErrors) > 0 && !sentSuccessfully {
		return fmt.Errorf("alert dispatch failed: %s", strings.Join(dispatchErrors, "; "))
	}

	return nil
}

// TriggerCrash mengirimkan alert ketika stream slot mengalami crash.
func (d *AlertDispatcher) TriggerCrash(ctx context.Context, slotNumber int, slotName, errMsg string) error {
	event := &domain.AlertEvent{
		Type:      "crash",
		Level:     "critical",
		Title:     fmt.Sprintf("Stream Crash: Slot %d (%s)", slotNumber, slotName),
		Message:   fmt.Sprintf("Process FFmpeg exited unexpectedly: %s", errMsg),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"slot_number": slotNumber,
			"slot_name":   slotName,
			"error":       errMsg,
		},
	}
	return d.Dispatch(ctx, event)
}

// TriggerThermal mengirimkan alert saat suhu perangkat melebihi batas aman.
func (d *AlertDispatcher) TriggerThermal(ctx context.Context, tempC, thresholdC float64) error {
	event := &domain.AlertEvent{
		Type:      "thermal",
		Level:     "warning",
		Title:     "High CPU Temperature Alert",
		Message:   fmt.Sprintf("System temperature reached %.1f°C (threshold: %.1f°C). Potential thermal throttling!", tempC, thresholdC),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"temperature_c": tempC,
			"threshold_c":   thresholdC,
		},
	}
	return d.Dispatch(ctx, event)
}

// TriggerLowStorage mengirimkan alert saat sisa penyimpanan internal kritis.
func (d *AlertDispatcher) TriggerLowStorage(ctx context.Context, freeGB, thresholdGB float64) error {
	event := &domain.AlertEvent{
		Type:      "low_storage",
		Level:     "warning",
		Title:     "Low Storage Space Warning",
		Message:   fmt.Sprintf("Available disk space is only %.2f GB (threshold: %.2f GB). Please clean video cache.", freeGB, thresholdGB),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"free_gb":      freeGB,
			"threshold_gb": thresholdGB,
		},
	}
	return d.Dispatch(ctx, event)
}

func formatTelegramMessage(event *domain.AlertEvent) string {
	icon := "⚠️"
	if event.Level == "critical" {
		icon = "🚨"
	} else if event.Type == "test" {
		icon = "⚡"
	}

	timeStr := event.Timestamp.Format("2006-01-02 15:04:05 WIB")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%s [GO-STREAMER ALERT]*\n", icon))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("*Title:* %s\n", event.Title))
	sb.WriteString(fmt.Sprintf("*Level:* `%s`\n", strings.ToUpper(event.Level)))
	sb.WriteString(fmt.Sprintf("*Time:* `%s`\n", timeStr))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("%s\n", event.Message))

	return sb.String()
}

func formatDiscordDescription(event *domain.AlertEvent) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Severity:** `%s`\n", strings.ToUpper(event.Level)))
	sb.WriteString(fmt.Sprintf("**Event:** `%s`\n\n", strings.ToUpper(event.Type)))
	sb.WriteString(fmt.Sprintf(">>> %s\n", event.Message))
	return sb.String()
}

func getDiscordColor(level string) int {
	switch strings.ToLower(level) {
	case "critical":
		return 0xEF4444 // Red
	case "warning":
		return 0xF59E0B // Amber/Orange
	case "info":
		return 0x3B82F6 // Blue
	default:
		return 0xFACC15 // Neobrutalism Vibrant Yellow
	}
}
