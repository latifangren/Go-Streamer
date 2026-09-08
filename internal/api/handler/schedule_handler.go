package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"go-streamer/internal/api/ws"
	"go-streamer/internal/domain"
	"go-streamer/internal/repository/sqlite"
	"go-streamer/internal/scheduler"
)

// ScheduleHandler menangani endpoint penjadwalan streaming dan pemeriksaan overlap.
type ScheduleHandler struct {
	scheduleRepo *sqlite.ScheduleRepository
	slotRepo     *sqlite.SlotRepository
	scheduler    *scheduler.Scheduler
	wsHub        *ws.Hub
}

// NewScheduleHandler membuat instans baru ScheduleHandler.
func NewScheduleHandler(
	scheduleRepo *sqlite.ScheduleRepository,
	slotRepo *sqlite.SlotRepository,
	sched *scheduler.Scheduler,
	wsHub *ws.Hub,
) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleRepo: scheduleRepo,
		slotRepo:     slotRepo,
		scheduler:    sched,
		wsHub:        wsHub,
	}
}

// ListSchedules menangani GET /api/v1/schedules.
func (h *ScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.scheduleRepo.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list schedules: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, schedules)
}

// CreateSchedule menangani POST /api/v1/schedules dengan proteksi OverlapGuard.
func (h *ScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req domain.Schedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body: "+err.Error())
		return
	}

	req.CronExpr = strings.TrimSpace(req.CronExpr)
	if req.CronExpr == "" {
		writeError(w, http.StatusBadRequest, "cron_expr is required")
		return
	}
	if req.SlotID <= 0 {
		writeError(w, http.StatusBadRequest, "slot_id must be valid")
		return
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 60 // default 60 menit
	}
	if req.OverlapGuardPolicy == "" {
		req.OverlapGuardPolicy = domain.OverlapYieldPriority
	}

	// 1. Verifikasi slot ada
	slot, err := h.slotRepo.GetByID(r.Context(), req.SlotID)
	if err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "target slot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to query target slot: "+err.Error())
		return
	}

	// 2. Evaluasi tabrakan jadwal (OverlapGuard)
	existingSchedules, err := h.scheduleRepo.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch existing schedules: "+err.Error())
		return
	}

	if conflicting, overlap := scheduler.CheckOverlap(existingSchedules, &req); overlap {
		if req.OverlapGuardPolicy == domain.OverlapDenyNew {
			writeJSON(w, http.StatusConflict, map[string]interface{}{
				"error":                "schedule overlaps with existing schedule and policy is deny_new",
				"conflicting_schedule": conflicting,
			})
			return
		}
	}

	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	if err := h.scheduleRepo.Create(r.Context(), &req); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create schedule: "+err.Error())
		return
	}

	// 3. Daftarkan cron task ke active scheduler
	if h.scheduler != nil && req.IsEnabled {
		_ = h.scheduler.AddStreamJob(&req, slot, "")
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("schedule_created", req)
	}

	writeJSON(w, http.StatusCreated, req)
}

// DeleteSchedule menangani DELETE /api/v1/schedules/{id}.
func (h *ScheduleHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "schedule id is required")
		return
	}

	if err := h.scheduleRepo.Delete(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			writeError(w, http.StatusNotFound, "schedule not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete schedule: "+err.Error())
		return
	}

	// Hapus task dari scheduler jika sedang aktif
	if h.scheduler != nil {
		_ = h.scheduler.RemoveStreamJob(id)
	}

	if h.wsHub != nil {
		h.wsHub.Broadcast("schedule_deleted", map[string]string{"id": id})
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "schedule deleted successfully",
		"id":      id,
	})
}

// GetOverlapStatus menangani GET /api/v1/schedules/overlap-status.
func (h *ScheduleHandler) GetOverlapStatus(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.scheduleRepo.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list schedules: "+err.Error())
		return
	}

	type OverlapPair struct {
		ScheduleA string `json:"schedule_a"`
		ScheduleB string `json:"schedule_b"`
	}

	var conflicts []OverlapPair
	for i := 0; i < len(schedules); i++ {
		for j := i + 1; j < len(schedules); j++ {
			if _, overlap := scheduler.CheckOverlap([]*domain.Schedule{schedules[i]}, schedules[j]); overlap {
				conflicts = append(conflicts, OverlapPair{
					ScheduleA: schedules[i].ID,
					ScheduleB: schedules[j].ID,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"has_overlap":     len(conflicts) > 0,
		"conflict_count":  len(conflicts),
		"conflicts":       conflicts,
		"total_schedules": len(schedules),
	})
}
