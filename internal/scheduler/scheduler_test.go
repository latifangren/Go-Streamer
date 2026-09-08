package scheduler

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

// mockSupervisor mengimplementasikan StreamSupervisor untuk keperluan pengujian.
type mockSupervisor struct {
	mu           sync.Mutex
	startedSlots map[int]string
	stoppedSlots []int
	failStart    bool
}

func newMockSupervisor() *mockSupervisor {
	return &mockSupervisor{
		startedSlots: make(map[int]string),
		stoppedSlots: make([]int, 0),
	}
}

func (m *mockSupervisor) StartSlot(slot *domain.StreamSlot, inputSource string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failStart {
		return errors.New("simulated start failure")
	}
	m.startedSlots[slot.SlotNumber] = inputSource
	return nil
}

func (m *mockSupervisor) StopSlot(slotNumber int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stoppedSlots = append(m.stoppedSlots, slotNumber)
	delete(m.startedSlots, slotNumber)
	return nil
}

func (m *mockSupervisor) IsStarted(slotNumber int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.startedSlots[slotNumber]
	return ok
}

func (m *mockSupervisor) HasStopped(slotNumber int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, num := range m.stoppedSlots {
		if num == slotNumber {
			return true
		}
	}
	return false
}

// ----------------------------------------------------------------------------
// 1. Tests for CheckOverlap
// ----------------------------------------------------------------------------

func TestCheckOverlap(t *testing.T) {
	t.Run("nil or empty schedule", func(t *testing.T) {
		s, overlapped := CheckOverlap(nil, nil)
		if overlapped || s != nil {
			t.Errorf("expected no overlap for nil schedule")
		}

		emptySched := &domain.Schedule{CronExpr: ""}
		s, overlapped = CheckOverlap(nil, emptySched)
		if overlapped || s != nil {
			t.Errorf("expected no overlap for empty cron expression")
		}
	})

	t.Run("identical schedule times - definite collision", func(t *testing.T) {
		existing := []*domain.Schedule{
			{
				ID:              "sched-1",
				SlotID:          1,
				CronExpr:        "0 12 * * *", // Setiap hari jam 12:00
				DurationMinutes: 60,
				IsEnabled:       true,
			},
		}

		newSched := &domain.Schedule{
			ID:              "sched-2",
			SlotID:          2,
			CronExpr:        "0 12 * * *", // Setiap hari jam 12:00
			DurationMinutes: 30,
			IsEnabled:       true,
		}

		conflict, overlapped := CheckOverlap(existing, newSched)
		if !overlapped {
			t.Fatalf("expected overlap on identical cron schedule times")
		}
		if conflict.ID != "sched-1" {
			t.Errorf("expected conflict with sched-1, got %s", conflict.ID)
		}
	})

	t.Run("overlapping broadcast duration", func(t *testing.T) {
		existing := []*domain.Schedule{
			{
				ID:              "sched-1",
				SlotID:          1,
				CronExpr:        "0 14 * * *", // 14:00 durasi 60m (berakhir 15:00)
				DurationMinutes: 60,
				IsEnabled:       true,
			},
		}

		newSched := &domain.Schedule{
			ID:              "sched-2",
			SlotID:          2,
			CronExpr:        "30 14 * * *", // 14:30 durasi 30m (berakhir 15:00)
			DurationMinutes: 30,
			IsEnabled:       true,
		}

		conflict, overlapped := CheckOverlap(existing, newSched)
		if !overlapped {
			t.Fatalf("expected overlap when start time falls inside existing broadcast window")
		}
		if conflict == nil || conflict.ID != "sched-1" {
			t.Errorf("expected conflict with sched-1")
		}
	})

	t.Run("back-to-back schedules - no collision", func(t *testing.T) {
		existing := []*domain.Schedule{
			{
				ID:              "sched-1",
				SlotID:          1,
				CronExpr:        "0 10 * * *", // 10:00 - 11:00
				DurationMinutes: 60,
				IsEnabled:       true,
			},
		}

		newSched := &domain.Schedule{
			ID:              "sched-2",
			SlotID:          2,
			CronExpr:        "0 11 * * *", // 11:00 - 12:00 (tepat saat yang lama selesai)
			DurationMinutes: 60,
			IsEnabled:       true,
		}

		_, overlapped := CheckOverlap(existing, newSched)
		if overlapped {
			t.Errorf("expected no overlap for back-to-back schedules")
		}
	})

	t.Run("disabled existing schedule ignored", func(t *testing.T) {
		existing := []*domain.Schedule{
			{
				ID:              "sched-disabled",
				SlotID:          1,
				CronExpr:        "0 12 * * *",
				DurationMinutes: 60,
				IsEnabled:       false, // Dimatikan
			},
		}

		newSched := &domain.Schedule{
			ID:              "sched-new",
			SlotID:          2,
			CronExpr:        "0 12 * * *",
			DurationMinutes: 30,
			IsEnabled:       true,
		}

		_, overlapped := CheckOverlap(existing, newSched)
		if overlapped {
			t.Errorf("expected disabled schedule to be skipped")
		}
	})

	t.Run("same schedule ID ignored", func(t *testing.T) {
		existing := []*domain.Schedule{
			{
				ID:              "sched-self",
				SlotID:          1,
				CronExpr:        "0 12 * * *",
				DurationMinutes: 60,
				IsEnabled:       true,
			},
		}

		newSched := &domain.Schedule{
			ID:              "sched-self", // update jadwal yang sama
			SlotID:          1,
			CronExpr:        "0 12 * * *",
			DurationMinutes: 60,
			IsEnabled:       true,
		}

		_, overlapped := CheckOverlap(existing, newSched)
		if overlapped {
			t.Errorf("updating schedule should not conflict with itself")
		}
	})
}

// ----------------------------------------------------------------------------
// 2. Tests for ResolveConflict
// ----------------------------------------------------------------------------

func TestResolveConflict(t *testing.T) {
	tests := []struct {
		name                string
		policy              string
		activeSlotNumber    int
		candidateSlotNumber int
		expected            string
	}{
		{
			name:                "YieldPriority - active has higher priority (slot 1 <= slot 2)",
			policy:              domain.OverlapYieldPriority,
			activeSlotNumber:    1,
			candidateSlotNumber: 2,
			expected:            "yield_to_priority",
		},
		{
			name:                "YieldPriority - equal slot priority (slot 1 <= slot 1)",
			policy:              domain.OverlapYieldPriority,
			activeSlotNumber:    1,
			candidateSlotNumber: 1,
			expected:            "yield_to_priority",
		},
		{
			name:                "YieldPriority - candidate has higher priority (active 3 > candidate 1)",
			policy:              domain.OverlapYieldPriority,
			activeSlotNumber:    3,
			candidateSlotNumber: 1,
			expected:            "proceed",
		},
		{
			name:                "TerminatePrev policy",
			policy:              domain.OverlapTerminatePrev,
			activeSlotNumber:    1,
			candidateSlotNumber: 2,
			expected:            "terminate_previous",
		},
		{
			name:                "DenyNew policy",
			policy:              domain.OverlapDenyNew,
			activeSlotNumber:    1,
			candidateSlotNumber: 2,
			expected:            "deny_candidate",
		},
		{
			name:                "Unknown policy defaults to deny_candidate",
			policy:              "unknown_policy",
			activeSlotNumber:    1,
			candidateSlotNumber: 2,
			expected:            "deny_candidate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveConflict(tt.policy, tt.activeSlotNumber, tt.candidateSlotNumber)
			if got != tt.expected {
				t.Errorf("ResolveConflict(%s, %d, %d) = %s; want %s",
					tt.policy, tt.activeSlotNumber, tt.candidateSlotNumber, got, tt.expected)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// 3. Tests for Scheduler (Add, Remove, Lifecycle, Execution)
// ----------------------------------------------------------------------------

func TestScheduler_AddAndRemoveJob(t *testing.T) {
	mockSup := newMockSupervisor()
	s := NewScheduler(mockSup, "")
	s.Start()
	defer s.Stop()

	sched := &domain.Schedule{
		ID:              "job-1",
		SlotID:          1,
		CronExpr:        "0 0 * * *", // daily at midnight
		DurationMinutes: 60,
		IsEnabled:       true,
	}

	slot := &domain.StreamSlot{
		ID:         1,
		SlotNumber: 1,
		Name:       "Test Slot 1",
	}

	// 1. Add valid job
	err := s.AddStreamJob(sched, slot, "/path/to/video.mp4")
	if err != nil {
		t.Fatalf("failed to add stream job: %v", err)
	}

	if !s.HasJob("job-1") {
		t.Errorf("expected job-1 to exist in scheduler")
	}
	if s.ActiveJobCount() != 1 {
		t.Errorf("expected active job count 1, got %d", s.ActiveJobCount())
	}

	// 2. Remove job
	err = s.RemoveStreamJob("job-1")
	if err != nil {
		t.Fatalf("failed to remove stream job: %v", err)
	}

	if s.HasJob("job-1") {
		t.Errorf("expected job-1 to be removed from scheduler")
	}
	if s.ActiveJobCount() != 0 {
		t.Errorf("expected active job count 0, got %d", s.ActiveJobCount())
	}

	// 3. Remove non-existent job
	err = s.RemoveStreamJob("job-unknown")
	if err == nil {
		t.Errorf("expected error when removing unknown job, got nil")
	}
}

func TestScheduler_Validation(t *testing.T) {
	mockSup := newMockSupervisor()
	s := NewScheduler(mockSup, "")

	slot := &domain.StreamSlot{SlotNumber: 1}

	if err := s.AddStreamJob(nil, slot, "video.mp4"); err == nil {
		t.Errorf("expected error for nil schedule")
	}

	sched := &domain.Schedule{ID: "test", CronExpr: "invalid cron"}
	if err := s.AddStreamJob(sched, nil, "video.mp4"); err == nil {
		t.Errorf("expected error for nil slot")
	}

	schedEmptyID := &domain.Schedule{ID: "", CronExpr: "0 0 * * *"}
	if err := s.AddStreamJob(schedEmptyID, slot, "video.mp4"); err == nil {
		t.Errorf("expected error for empty schedule ID")
	}

	if err := s.AddStreamJob(sched, slot, "video.mp4"); err == nil {
		t.Errorf("expected error for invalid cron expression")
	}
}

// ----------------------------------------------------------------------------
// 4. Tests for CleanExpiredCache
// ----------------------------------------------------------------------------

func TestCleanExpiredCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sched-cache-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	now := time.Now()
	oldTime := now.Add(-30 * time.Hour) // > 24 jam yang lalu
	freshTime := now.Add(-1 * time.Hour) // baru 1 jam

	// Buat file thumbnail .jpg lama
	oldFile := filepath.Join(tmpDir, "snapshot_old.jpg")
	if err := os.WriteFile(oldFile, []byte("old-data"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(oldFile, oldTime, oldTime)

	// Buat file thumbnail .jpg baru
	freshFile := filepath.Join(tmpDir, "snapshot_fresh.jpg")
	if err := os.WriteFile(freshFile, []byte("fresh-data"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(freshFile, freshTime, freshTime)

	// Buat file non-jpg lama (tidak boleh dihapus)
	otherFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(otherFile, []byte("txt-data"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(otherFile, oldTime, oldTime)

	// Eksekusi CleanExpiredCache dengan batas 24 jam
	cleaned, err := CleanExpiredCache(tmpDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("CleanExpiredCache returned error: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("expected 1 file cleaned, got %d", cleaned)
	}

	// Verifikasi: file lama terhapus
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("expected oldFile to be deleted")
	}

	// Verifikasi: file baru tetap ada
	if _, err := os.Stat(freshFile); err != nil {
		t.Errorf("expected freshFile to remain, got err: %v", err)
	}

	// Verifikasi: non-jpg tetap ada
	if _, err := os.Stat(otherFile); err != nil {
		t.Errorf("expected otherFile (non-jpg) to remain, got err: %v", err)
	}
}
