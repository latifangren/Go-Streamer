package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

func TestScheduleRepository_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_sched.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := NewScheduleRepository(db.DB)

	sc := &domain.Schedule{
		ID:                 "sched-1",
		Title:              "Daily Morning Stream",
		SlotID:             1,
		CronExpr:           "0 8 * * *",
		DurationMinutes:    120,
		IsEnabled:          true,
		OverlapGuardPolicy: domain.OverlapYieldPriority,
	}

	if err := repo.Create(ctx, sc); err != nil {
		t.Fatalf("Create schedule failed: %v", err)
	}

	// GetByID
	fetched, err := repo.GetByID(ctx, "sched-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.Title != "Daily Morning Stream" {
		t.Errorf("expected Title 'Daily Morning Stream', got '%s'", fetched.Title)
	}
	if fetched.SlotID != 1 {
		t.Errorf("expected SlotID 1, got %d", fetched.SlotID)
	}

	// ListAll
	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(all))
	}
	if all[0].Title != "Daily Morning Stream" {
		t.Errorf("expected Title 'Daily Morning Stream' in list, got '%s'", all[0].Title)
	}

	// Delete
	if err := repo.Delete(ctx, "sched-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = repo.GetByID(ctx, "sched-1")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}
