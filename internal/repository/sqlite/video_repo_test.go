package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

func setupTestDB(t *testing.T) (*DB, *VideoRepository) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_repo.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}

	repo := NewVideoRepository(db.DB)
	return db, repo
}

func TestVideoRepository_CRUD(t *testing.T) {
	db, repo := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	v1 := &domain.Video{
		ID:                 "vid-uuid-1",
		UserID:             "usr_admin_default",
		Filename:           "vid-uuid-1_video1.mp4",
		OriginalName:       "video1.mp4",
		FilePath:           "/data/videos/video1.mp4",
		FileSize:           1024 * 1024 * 10, // 10MB
		DurationSeconds:    60.5,
		Resolution:         "1920x1080",
		VideoCodec:         "h264",
		AudioCodec:         "aac",
		FPS:                30.0,
		GOPSize:            2.0,
		IsPassthroughReady: true,
	}

	// 1. Create
	if err := repo.Create(ctx, v1); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 2. GetByID
	got, err := repo.GetByID(ctx, "vid-uuid-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.ID != v1.ID || got.OriginalName != v1.OriginalName || got.FileSize != v1.FileSize {
		t.Errorf("GetByID mismatch, got %+v", got)
	}

	// 3. GetByUserID
	userVideos, err := repo.GetByUserID(ctx, "usr_admin_default")
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	if len(userVideos) != 1 || userVideos[0].ID != "vid-uuid-1" {
		t.Errorf("expected 1 video for user, got %d", len(userVideos))
	}

	// 4. Update
	v1.OriginalName = "video1_renamed.mp4"
	v1.FileSize = 1024 * 1024 * 12
	if err := repo.Update(ctx, v1); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, err := repo.GetByID(ctx, "vid-uuid-1")
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if updated.OriginalName != "video1_renamed.mp4" || updated.FileSize != 1024*1024*12 {
		t.Errorf("updated fields mismatch, got %+v", updated)
	}

	// 5. Create second video and test ListAll and GetTotalStorageBytes
	v2 := &domain.Video{
		ID:                 "vid-uuid-2",
		UserID:             "usr_admin_default",
		Filename:           "vid-uuid-2_video2.mp4",
		OriginalName:       "video2.mp4",
		FilePath:           "/data/videos/video2.mp4",
		FileSize:           1024 * 1024 * 8, // 8MB
		DurationSeconds:    45.0,
		Resolution:         "1280x720",
		VideoCodec:         "h264",
		AudioCodec:         "aac",
		FPS:                25.0,
		GOPSize:            2.0,
		IsPassthroughReady: true,
	}
	if err := repo.Create(ctx, v2); err != nil {
		t.Fatalf("Create v2 failed: %v", err)
	}

	allVideos, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(allVideos) != 2 {
		t.Errorf("expected 2 videos, got %d", len(allVideos))
	}

	totalBytes, err := repo.GetTotalStorageBytes(ctx)
	if err != nil {
		t.Fatalf("GetTotalStorageBytes failed: %v", err)
	}
	expectedBytes := int64((1024 * 1024 * 12) + (1024 * 1024 * 8))
	if totalBytes != expectedBytes {
		t.Errorf("expected %d total bytes, got %d", expectedBytes, totalBytes)
	}

	// 6. Delete
	if err := repo.Delete(ctx, "vid-uuid-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByID(ctx, "vid-uuid-1")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}

	// Delete non-existent
	if err := repo.Delete(ctx, "non-existent"); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for non-existent delete, got %v", err)
	}
}
