package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/domain"
	"go-streamer/internal/repository/sqlite"
	"go-streamer/internal/runner"
)

// SlotHandler menangani rute HTTP REST API untuk entitas StreamSlot.
type SlotHandler struct {
	slotRepo   *sqlite.SlotRepository
	videoRepo  *sqlite.VideoRepository
	supervisor *runner.Supervisor
	wsHub      *ws.Hub
	cacheDir   string
}

// NewSlotHandler membuat instans baru SlotHandler.
func NewSlotHandler(
	slotRepo *sqlite.SlotRepository,
	videoRepo *sqlite.VideoRepository,
	supervisor *runner.Supervisor,
	wsHub *ws.Hub,
	cacheDir string,
) *SlotHandler {
	return &SlotHandler{
		slotRepo:   slotRepo,
		videoRepo:  videoRepo,
		supervisor: supervisor,
		wsHub:      wsHub,
		cacheDir:   cacheDir,
	}
}

// ListSlots menangani GET /api/v1/slots dan menyinkronkan status live dari supervisor.
func (h *SlotHandler) ListSlots(w http.ResponseWriter, r *http.Request) {
	slots, err := h.slotRepo.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list slots: "+err.Error())
		return
	}

	// Sinkronkan status realtime dari supervisor
	if h.supervisor != nil {
		for _, s := range slots {
			liveStatus, liveTel, running := h.supervisor.GetSlotStatus(s.SlotNumber)
			if running {
				s.Status = liveStatus
			} else if s.Status == domain.SlotStatusRunning {
				s.Status = domain.SlotStatusIdle
			}
			if liveTel != nil {
				s.LastPTSSyncMS = liveTel.PTSSyncMS
				s.LastNetLatencyMS = liveTel.NetLatencyMS
			}
		}
	}

	writeJSON(w, http.StatusOK, slots)
}

// GetSlot menangani GET /api/v1/slots/{id}.
func (h *SlotHandler) GetSlot(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid slot id")
		return
	}

	slot, err := h.slotRepo.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "slot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get slot: "+err.Error())
		return
	}

	if h.supervisor != nil {
		liveStatus, liveTel, running := h.supervisor.GetSlotStatus(slot.SlotNumber)
		if running {
			slot.Status = liveStatus
		}
		if liveTel != nil {
			slot.LastPTSSyncMS = liveTel.PTSSyncMS
			slot.LastNetLatencyMS = liveTel.NetLatencyMS
		}
	}

	writeJSON(w, http.StatusOK, slot)
}

// UpdateSlot menangani PUT /api/v1/slots/{id}.
func (h *SlotHandler) UpdateSlot(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid slot id")
		return
	}

	slot, err := h.slotRepo.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "slot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get slot: "+err.Error())
		return
	}

	var req domain.StreamSlot
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body: "+err.Error())
		return
	}

	// Update field yang diizinkan
	slot.Name = req.Name
	slot.SourceType = req.SourceType
	slot.VideoID = req.VideoID
	slot.TargetPlatform = req.TargetPlatform
	slot.RTMPURL = req.RTMPURL
	if req.StreamKey != "" {
		slot.StreamKey = req.StreamKey
	}
	slot.Mode = req.Mode
	slot.Quality = req.Quality
	slot.Preset = req.Preset
	slot.LoopPlayback = req.LoopPlayback
	slot.MaxDurationMinutes = req.MaxDurationMinutes
	slot.AutoRestart = req.AutoRestart
	slot.EnableOverlay = req.EnableOverlay
	slot.OverlayClockWIB = req.OverlayClockWIB
	slot.OverlayWatermark = req.OverlayWatermark
	slot.OverlayWatermarkPos = req.OverlayWatermarkPos

	if err := h.slotRepo.Update(r.Context(), slot); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update slot: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("slot_updated", slot)
	}

	writeJSON(w, http.StatusOK, slot)
}

// StartSlot menangani POST /api/v1/slots/{id}/start.
func (h *SlotHandler) StartSlot(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid slot id")
		return
	}

	slot, err := h.slotRepo.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "slot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get slot: "+err.Error())
		return
	}

	if h.supervisor == nil {
		writeError(w, http.StatusInternalServerError, "supervisor is not initialized")
		return
	}

	// Tentukan inputSource berdasarkan konfigurasi slot
	var inputSource string

	// 1. Cek apakah ada override input_source di request body (opsional)
	var bodyReq struct {
		InputSource string `json:"input_source"`
	}
	_ = json.NewDecoder(r.Body).Decode(&bodyReq)
	if strings.TrimSpace(bodyReq.InputSource) != "" {
		inputSource = strings.TrimSpace(bodyReq.InputSource)
	} else if slot.SourceType == domain.SourceTypeSingle && slot.VideoID != nil {
		// 2. Ambil path video dari tabel videos
		if h.videoRepo != nil {
			v, vErr := h.videoRepo.GetByID(r.Context(), *slot.VideoID)
			if vErr == nil && v != nil {
				inputSource = v.FilePath
			}
		}
	} else if slot.SourceType == domain.SourceTypePlaylist {
		// 3. Concat manifest
		manifestPath := filepath.Join(h.cacheDir, fmt.Sprintf("concat_slot_%d.txt", slot.ID))
		if _, err := os.Stat(manifestPath); err == nil {
			inputSource = manifestPath
		}
	}

	if inputSource == "" {
		writeError(w, http.StatusBadRequest, "no video or input source configured for slot")
		return
	}

	if err := h.supervisor.StartSlot(slot, inputSource); err != nil {
		if err == domain.ErrSlotBusy {
			writeError(w, http.StatusConflict, "slot is already running")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to start slot: "+err.Error())
		return
	}

	_ = h.slotRepo.UpdateStatus(r.Context(), slot.ID, domain.SlotStatusRunning)

	if h.wsHub != nil {
		h.wsHub.Broadcast("slot_status_changed", map[string]interface{}{
			"slot_id":     slot.ID,
			"slot_number": slot.SlotNumber,
			"status":      domain.SlotStatusRunning,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "slot streaming started",
		"slot_id":     slot.ID,
		"slot_number": slot.SlotNumber,
		"status":      domain.SlotStatusRunning,
	})
}

// StopSlot menangani POST /api/v1/slots/{id}/stop.
func (h *SlotHandler) StopSlot(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid slot id")
		return
	}

	slot, err := h.slotRepo.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "slot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get slot: "+err.Error())
		return
	}

	if h.supervisor == nil {
		writeError(w, http.StatusInternalServerError, "supervisor is not initialized")
		return
	}

	if err := h.supervisor.StopSlot(slot.SlotNumber); err != nil {
		if err == domain.ErrSlotNotRunning {
			writeError(w, http.StatusConflict, "slot is not running")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to stop slot: "+err.Error())
		return
	}

	_ = h.slotRepo.UpdateStatus(r.Context(), slot.ID, domain.SlotStatusIdle)

	if h.wsHub != nil {
		h.wsHub.Broadcast("slot_status_changed", map[string]interface{}{
			"slot_id":     slot.ID,
			"slot_number": slot.SlotNumber,
			"status":      domain.SlotStatusIdle,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "slot streaming stopped",
		"slot_id":     slot.ID,
		"slot_number": slot.SlotNumber,
		"status":      domain.SlotStatusIdle,
	})
}

// GetSnapshot menangani GET /api/v1/slots/{id}/snapshot.
func (h *SlotHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid slot id")
		return
	}

	slot, err := h.slotRepo.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "slot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get slot: "+err.Error())
		return
	}

	snapshotPath := slot.LastSnapshotPath
	if snapshotPath == "" || !fileExists(snapshotPath) {
		fallback := filepath.Join(h.cacheDir, fmt.Sprintf("snapshot_slot_%d.jpg", slot.SlotNumber))
		if fileExists(fallback) {
			snapshotPath = fallback
		}
	}

	if snapshotPath == "" || !fileExists(snapshotPath) {
		writeError(w, http.StatusNotFound, "snapshot not available yet")
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, snapshotPath)
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
