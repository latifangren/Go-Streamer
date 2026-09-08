package handler

import (
	"encoding/json"
	"net/http"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/domain"
	"go-streamer/internal/service"
)

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

	writeJSON(w, http.StatusOK, settings)
}

// UpdateSettings menangani PUT /api/v1/alerts/settings.
func (h *AlertHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings domain.AlertSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body: "+err.Error())
		return
	}

	if err := h.alertService.UpdateSettings(r.Context(), &settings); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update alert settings: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("alert_settings_updated", settings)
	}

	writeJSON(w, http.StatusOK, settings)
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
