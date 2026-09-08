package scheduler

import (
	"time"

	"github.com/robfig/cron/v3"
	"go-streamer/internal/domain"
)

// CheckOverlap mengevaluasi apakah newSchedule bertabrakan dengan jadwal yang sudah ada
// dalam rentang waktu siaran tertentu di masa depan (simulasi hingga 14 hari ke depan).
// Mengembalikan jadwal yang bentrok (jika ada) dan status boolean overlap.
func CheckOverlap(existingSchedules []*domain.Schedule, newSchedule *domain.Schedule) (*domain.Schedule, bool) {
	if newSchedule == nil || newSchedule.CronExpr == "" {
		return nil, false
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	newCron, err := parser.Parse(newSchedule.CronExpr)
	if err != nil {
		return nil, false
	}

	newDur := time.Duration(newSchedule.DurationMinutes) * time.Minute
	if newDur <= 0 {
		newDur = 1 * time.Minute
	}

	now := time.Now().Truncate(time.Minute)
	horizon := now.Add(14 * 24 * time.Hour)

	for _, existing := range existingSchedules {
		if existing == nil || !existing.IsEnabled || existing.ID == newSchedule.ID {
			continue
		}

		existCron, err := parser.Parse(existing.CronExpr)
		if err != nil {
			continue
		}

		existDur := time.Duration(existing.DurationMinutes) * time.Minute
		if existDur <= 0 {
			existDur = 1 * time.Minute
		}

		// Iterasi jadwal newSchedule dan cek apakah ada irisan waktu dengan existing
		currNew := now
		iterations := 0
		maxIterations := 200

		for iterations < maxIterations {
			newStart := newCron.Next(currNew)
			if newStart.IsZero() || newStart.After(horizon) {
				break
			}
			newEnd := newStart.Add(newDur)

			// Cari kejadian 'existing' yang dimulai setelah searchStart (newStart - existDur - 1s)
			// agar mencakup siaran existing yang masih aktif saat newStart dimulai
			searchStart := newStart.Add(-existDur).Add(-time.Second)
			existStart := existCron.Next(searchStart)

			for !existStart.IsZero() && existStart.Before(newEnd) {
				existEnd := existStart.Add(existDur)

				// Overlap terjadi jika interval [newStart, newEnd) dan [existStart, existEnd) beririsan
				// newStart < existEnd DAN existStart < newEnd
				if newStart.Before(existEnd) && existStart.Before(newEnd) {
					return existing, true
				}

				existStart = existCron.Next(existStart)
			}

			currNew = newStart
			iterations++
		}
	}

	return nil, false
}

// ResolveConflict menentukan aksi saat terjadi bentrok jadwal berdasarkan policy yang dikonfigurasi:
// - OverlapYieldPriority: jika activeSlotNumber <= candidateSlotNumber (slot 1 lebih prioritas),
//   kembalikan "yield_to_priority", jika sebaliknya kembalikan "proceed".
// - OverlapTerminatePrev: kembalikan "terminate_previous".
// - OverlapDenyNew: kembalikan "deny_candidate".
func ResolveConflict(policy string, activeSlotNumber, candidateSlotNumber int) string {
	switch policy {
	case domain.OverlapYieldPriority:
		if activeSlotNumber <= candidateSlotNumber {
			return "yield_to_priority"
		}
		return "proceed"
	case domain.OverlapTerminatePrev:
		return "terminate_previous"
	case domain.OverlapDenyNew:
		return "deny_candidate"
	default:
		return "deny_candidate"
	}
}
