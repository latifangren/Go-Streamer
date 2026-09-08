package handler

import (
	"net/http"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/runner"
	"go-streamer/internal/sysinfo"
)

// SystemHandler menangani endpoint telemetri sistem dan emergency kill-switch.
type SystemHandler struct {
	collector  *sysinfo.Collector
	supervisor *runner.Supervisor
	wsHub      *ws.Hub
}

// NewSystemHandler membuat instans baru SystemHandler.
func NewSystemHandler(
	collector *sysinfo.Collector,
	supervisor *runner.Supervisor,
	wsHub *ws.Hub,
) *SystemHandler {
	return &SystemHandler{
		collector:  collector,
		supervisor: supervisor,
		wsHub:      wsHub,
	}
}

// GetMetrics menangani GET /api/v1/system/metrics.
func (h *SystemHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if h.collector == nil {
		writeError(w, http.StatusInternalServerError, "system collector is not initialized")
		return
	}

	metrics, err := h.collector.Collect()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to collect system metrics: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, metrics)
}

// KillSwitch menangani POST /api/v1/system/killswitch untuk mematikan semua slot streaming seketika.
func (h *SystemHandler) KillSwitch(w http.ResponseWriter, r *http.Request) {
	if h.supervisor != nil {
		h.supervisor.StopAll()
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("killswitch_activated", map[string]string{
			"message": "all streaming slots terminated immediately",
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "all streaming slots terminated immediately",
	})
}
