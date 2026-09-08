package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/domain"
	"go-streamer/internal/service"
)

// UpdateAlertSettingsRequest mendefinisikan payload pembaruan pengaturan notifikasi.
type UpdateAlertSettingsRequest struct {
	TelegramEnabled       bool    `json:"telegram_enabled"`
	TelegramBotToken      *string `json:"telegram_bot_token,omitempty"`
	TelegramChatID        string  `json:"telegram_chat_id"`
	DiscordEnabled        bool    `json:"discord_enabled"`
	DiscordWebhookURL     *string `json:"discord_webhook_url,omitempty"`
	TriggerOnCrash        bool    `json:"trigger_on_crash"`
	TriggerOnThermal      bool    `json:"trigger_on_thermal"`
	ThermalThresholdC     float64 `json:"thermal_threshold_c"`
	TriggerOnLowStorage   bool    `json:"trigger_on_low_storage"`
	LowStorageThresholdGB float64 `json:"low_storage_threshold_gb"`
}

type updateAlertRequest = UpdateAlertSettingsRequest

type alertSettingsResponse struct {
	ID                    int        `json:"id"`
	TelegramEnabled       bool       `json:"telegram_enabled"`
	TelegramBotToken      string     `json:"telegram_bot_token,omitempty"`
	TelegramChatID        string     `json:"telegram_chat_id"`
	HasTelegramToken      bool       `json:"has_telegram_token"`
	DiscordEnabled        bool       `json:"discord_enabled"`
	DiscordWebhookURL     string     `json:"discord_webhook_url,omitempty"`
	HasDiscordWebhook     bool       `json:"has_discord_webhook"`
	TriggerOnCrash        bool       `json:"trigger_on_crash"`
	TriggerOnThermal      bool       `json:"trigger_on_thermal"`
	ThermalThresholdC     float64    `json:"thermal_threshold_c"`
	TriggerOnLowStorage   bool       `json:"trigger_on_low_storage"`
	LowStorageThresholdGB float64    `json:"low_storage_threshold_gb"`
	LastAlertSentAt       *time.Time `json:"last_alert_sent_at,omitempty"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func toAlertSettingsResponse(s *domain.AlertSettings) alertSettingsResponse {
	resp := alertSettingsResponse{
		ID:                    s.ID,
		TelegramEnabled:       s.TelegramEnabled,
		TelegramChatID:        s.TelegramChatID,
		HasTelegramToken:      s.TelegramBotToken != "",
		DiscordEnabled:        s.DiscordEnabled,
		HasDiscordWebhook:     s.DiscordWebhookURL != "",
		TriggerOnCrash:        s.TriggerOnCrash,
		TriggerOnThermal:      s.TriggerOnThermal,
		ThermalThresholdC:     s.ThermalThresholdC,
		TriggerOnLowStorage:   s.TriggerOnLowStorage,
		LowStorageThresholdGB: s.LowStorageThresholdGB,
		LastAlertSentAt:       s.LastAlertSentAt,
		UpdatedAt:             s.UpdatedAt,
	}
	if s.TelegramBotToken != "" {
		resp.TelegramBotToken = "********"
	}
	if s.DiscordWebhookURL != "" {
		resp.DiscordWebhookURL = "********"
	}
	return resp
}

// AlertHandler menangani rute HTTP konfigurasi notifikasi Telegram dan Discord.
type AlertHandler struct {
	alertService *service.AlertService
	wsHub        *ws.Hub
}

// NewAlertHandler membuat instans baru AlertHandler.
func NewAlertHandler(alertService *service.AlertService, wsHub *ws.Hub) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
		wsHub:        wsHub,
	}
}

// GetSettings menangani GET /api/v1/alerts/settings.
func (h *AlertHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.alertService.GetSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get alert settings: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAlertSettingsResponse(settings))
}

// UpdateSettings menangani PUT /api/v1/alerts/settings.
func (h *AlertHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body: "+err.Error())
		return
	}

	curr, err := h.alertService.GetSettings(r.Context())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			curr = &domain.AlertSettings{ID: 1}
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get current alert settings: "+err.Error())
			return
		}
	}
	if curr == nil {
		curr = &domain.AlertSettings{ID: 1}
	}

	if req.TelegramBotToken != nil {
		if *req.TelegramBotToken == "" {
			curr.TelegramBotToken = ""
		} else if !strings.Contains(*req.TelegramBotToken, "****") {
			curr.TelegramBotToken = *req.TelegramBotToken
		}
	}
	if req.DiscordWebhookURL != nil {
		if *req.DiscordWebhookURL == "" {
			curr.DiscordWebhookURL = ""
		} else if !strings.Contains(*req.DiscordWebhookURL, "****") {
			curr.DiscordWebhookURL = *req.DiscordWebhookURL
		}
	}
	curr.TelegramEnabled = req.TelegramEnabled
	curr.TelegramChatID = req.TelegramChatID
	curr.DiscordEnabled = req.DiscordEnabled
	curr.TriggerOnCrash = req.TriggerOnCrash
	curr.TriggerOnThermal = req.TriggerOnThermal
	curr.ThermalThresholdC = req.ThermalThresholdC
	curr.TriggerOnLowStorage = req.TriggerOnLowStorage
	curr.LowStorageThresholdGB = req.LowStorageThresholdGB

	if err := h.alertService.UpdateSettings(r.Context(), curr); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update alert settings: "+err.Error())
		return
	}

	resp := toAlertSettingsResponse(curr)
	if h.wsHub != nil {
		h.wsHub.Broadcast("alert_settings_updated", resp)
	}

	writeJSON(w, http.StatusOK, resp)
}

// TestAlert menangani POST /api/v1/alerts/test.
func (h *AlertHandler) TestAlert(w http.ResponseWriter, r *http.Request) {
	if err := h.alertService.SendTestAlert(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send test alert: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "test alert sent successfully",
	})
}
