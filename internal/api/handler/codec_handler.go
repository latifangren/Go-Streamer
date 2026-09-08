package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/domain"
	"go-streamer/internal/service"
)

// CodecHandler menangani permintaan transcoding offline dan standardisasi video passthrough.
type CodecHandler struct {
	transcoderService *service.TranscoderService
	wsHub             *ws.Hub
}

// NewCodecHandler membuat instans baru CodecHandler.
func NewCodecHandler(transcoderService *service.TranscoderService, wsHub *ws.Hub) *CodecHandler {
	return &CodecHandler{
		transcoderService: transcoderService,
		wsHub:             wsHub,
	}
}

// CreateFixJob menangani POST /api/v1/codec/fix.
func (h *CodecHandler) CreateFixJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceVideoID string `json:"source_video_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body: "+err.Error())
		return
	}

	req.SourceVideoID = strings.TrimSpace(req.SourceVideoID)
	if req.SourceVideoID == "" {
		writeError(w, http.StatusBadRequest, "source_video_id is required")
		return
	}

	job, err := h.transcoderService.CreateFixJob(r.Context(), req.SourceVideoID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "source video not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create transcode job: "+err.Error())
		return
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("codec_job_created", job)
	}

	writeJSON(w, http.StatusCreated, job)
}

// GetJob menangani GET /api/v1/codec/jobs/{id}.
func (h *CodecHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "job id is required")
		return
	}

	job, err := h.transcoderService.GetJob(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "transcode job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get transcode job: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, job)
}

// ListJobs menangani GET /api/v1/codec/jobs.
func (h *CodecHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.transcoderService.ListJobs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list transcode jobs: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, jobs)
}
