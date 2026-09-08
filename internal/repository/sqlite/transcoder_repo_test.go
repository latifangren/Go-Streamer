package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

func TestTranscoderRepository_CRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_transcode.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	videoRepo := NewVideoRepository(db.DB)
	transcodeRepo := NewTranscoderRepository(db.DB)

	// 1. Setup video
	video := &domain.Video{
		ID:                 "v-100",
		UserID:             "usr_admin_default", // seeded by InitDB/New
		Filename:           "clip.mp4",
		OriginalName:       "original_clip.mp4",
		FilePath:           "/data/videos/clip.mp4",
		FileSize:           1024000,
		DurationSeconds:    45.5,
		Resolution:         "1280x720",
		VideoCodec:         "hevc",
		AudioCodec:         "opus",
		FPS:                30.0,
		GOPSize:            2.0,
		IsPassthroughReady: false,
		CreatedAt:          time.Now(),
	}
	if err := videoRepo.Create(ctx, video); err != nil {
		t.Fatalf("failed to create video: %v", err)
	}

	// 2. Create TranscodeJob
	job := &domain.TranscodeJob{
		ID:              "job-100",
		SourceVideoID:   video.ID,
		TargetFilePath:  "/data/videos/fixed_clip.mp4",
		Status:          domain.TranscodeQueued,
		ProgressPercent: 0.0,
		ErrorMessage:    "",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := transcodeRepo.CreateJob(ctx, job); err != nil {
		t.Fatalf("failed to create transcode job: %v", err)
	}

	// 3. GetJobByID with join to source video
	fetched, err := transcodeRepo.GetJobByID(ctx, "job-100")
	if err != nil {
		t.Fatalf("GetJobByID failed: %v", err)
	}
	if fetched.Status != domain.TranscodeQueued {
		t.Errorf("expected status queued, got %s", fetched.Status)
	}
	if fetched.SourceVideo == nil || fetched.SourceVideo.Filename != "clip.mp4" {
		t.Errorf("expected joined source video filename clip.mp4, got %+v", fetched.SourceVideo)
	}

	// 4. UpdateJob
	fetched.Status = domain.TranscodeProcessing
	fetched.ProgressPercent = 50.0
	if err := transcodeRepo.UpdateJob(ctx, fetched); err != nil {
		t.Fatalf("UpdateJob failed: %v", err)
	}

	// 5. GetActiveOrPendingJob
	activeJob, err := transcodeRepo.GetActiveOrPendingJob(ctx)
	if err != nil {
		t.Fatalf("GetActiveOrPendingJob failed: %v", err)
	}
	if activeJob == nil || activeJob.ID != "job-100" {
		t.Fatalf("expected active job-100, got %+v", activeJob)
	}

	// 6. ListJobs
	jobs, err := transcodeRepo.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}
