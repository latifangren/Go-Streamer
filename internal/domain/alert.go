package domain

import "time"

type AlertSettings struct {
	ID                    int        `json:"id"`
	TelegramEnabled       bool       `json:"telegram_enabled"`
	TelegramBotToken      string     `json:"-"`
	TelegramChatID        string     `json:"telegram_chat_id"`
	DiscordEnabled        bool       `json:"discord_enabled"`
	DiscordWebhookURL     string     `json:"-"`
	TriggerOnCrash        bool       `json:"trigger_on_crash"`
	TriggerOnThermal      bool       `json:"trigger_on_thermal"`
	ThermalThresholdC     float64    `json:"thermal_threshold_c"`
	TriggerOnLowStorage   bool       `json:"trigger_on_low_storage"`
	LowStorageThresholdGB float64    `json:"low_storage_threshold_gb"`
	LastAlertSentAt       *time.Time `json:"last_alert_sent_at,omitempty"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type AlertEvent struct {
	Type      string                 `json:"type"` // "crash", "thermal", "low_storage", "info"
	Level     string                 `json:"level"` // "info", "warning", "critical"
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}
