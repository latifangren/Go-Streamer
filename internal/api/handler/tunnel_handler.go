package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/service"
)

// TunnelHandler menangani endpoint integrasi Cloudflare Tunnel dan Tailscale.
type TunnelHandler struct {
	tunnelService *service.TunnelService
	wsHub         *ws.Hub
}

// NewTunnelHandler membuat instans baru TunnelHandler.
func NewTunnelHandler(tunnelService *service.TunnelService, wsHub *ws.Hub) *TunnelHandler {
	return &TunnelHandler{
		tunnelService: tunnelService,
		wsHub:         wsHub,
	}
}

// GetStatus menangani GET /api/v1/tunnel/status.
func (h *TunnelHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	settings, tailscaleIP, err := h.tunnelService.GetStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get tunnel status: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"settings":     settings,
		"tailscale_ip": tailscaleIP,
	})
}

// StartTunnel menangani POST /api/v1/tunnel/start.
func (h *TunnelHandler) StartTunnel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		Mode     string `json:"mode"`
		Token    string `json:"token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	provider := strings.TrimSpace(req.Provider)
	mode := strings.TrimSpace(req.Mode)
	token := strings.TrimSpace(req.Token)

	settings, err := h.tunnelService.StartTunnel(r.Context(), provider, mode, token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start tunnel: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("tunnel_status_changed", settings)
	}

	writeJSON(w, http.StatusOK, settings)
}

// StopTunnel menangani POST /api/v1/tunnel/stop.
func (h *TunnelHandler) StopTunnel(w http.ResponseWriter, r *http.Request) {
	settings, err := h.tunnelService.StopTunnel(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to stop tunnel: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("tunnel_status_changed", settings)
	}

	writeJSON(w, http.StatusOK, settings)
}
