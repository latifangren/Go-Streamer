package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go-streamer/internal/domain"
)

type TranscodeJobRepository interface {
	CreateJob(ctx context.Context, job *domain.TranscodeJob) error
	GetJobByID(ctx context.Context, id string) (*domain.TranscodeJob, error)
	UpdateJob(ctx context.Context, job *domain.TranscodeJob) error
	ListJobs(ctx context.Context) ([]*domain.TranscodeJob, error)
}

type VideoLookupRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Video, error)
}

type JobEnqueuer interface {
	Enqueue(jobID string)
}

type TranscoderService struct {
	jobRepo   TranscodeJobRepository
	videoRepo VideoLookupRepository
	enqueuer  JobEnqueuer
	videoDir  string
}

func NewTranscoderService(
	jobRepo TranscodeJobRepository,
	videoRepo VideoLookupRepository,
	enqueuer JobEnqueuer,
	videoDir string,
) *TranscoderService {
	return &TranscoderService{
		jobRepo:   jobRepo,
		videoRepo: videoRepo,
		enqueuer:  enqueuer,
		videoDir:  videoDir,
	}
}

// CreateFixJob memvalidasi video, mencegah duplikasi job aktif,
// menentukan path target, menyimpan job ke DB, dan meng-enqueue ke CodecFixer.
func (s *TranscoderService) CreateFixJob(ctx context.Context, videoID string) (*domain.TranscodeJob, error) {
	if videoID == "" {
		return nil, errors.New("video_id cannot be empty")
	}

	// 1. Validasi video ada di database
	video, err := s.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}
	if video == nil {
		return nil, fmt.Errorf("video with id %s not found: %w", videoID, domain.ErrNotFound)
	}

	// 2. Cek apakah sudah ada job pending/processing untuk video tersebut
	existingJobs, err := s.jobRepo.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing jobs: %w", err)
	}
	for _, ej := range existingJobs {
		if ej.SourceVideoID == videoID && (ej.Status == domain.TranscodeQueued || ej.Status == domain.TranscodeProcessing) {
			return nil, fmt.Errorf("transcode job already in progress for this video (job_id: %s)", ej.ID)
		}
	}

	// 3. Tentukan path target: filepath.Join(videoDir, fmt.Sprintf("fixed_%s_%s", video.ID, video.Filename))
	targetFilename := fmt.Sprintf("fixed_%s_%s", video.ID, video.Filename)
	targetFilePath := filepath.Join(s.videoDir, targetFilename)

	// 4. Simpan job ke DB
	now := time.Now()
	job := &domain.TranscodeJob{
		ID:              uuid.New().String(),
		SourceVideoID:   video.ID,
		TargetFilePath:  targetFilePath,
		Status:          domain.TranscodeQueued,
		ProgressPercent: 0.0,
		ErrorMessage:    "",
		CreatedAt:       now,
		UpdatedAt:       now,
		SourceVideo:     video,
	}

	if err := s.jobRepo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to save transcode job: %w", err)
	}

	// 5. Enqueue ke CodecFixer
	if s.enqueuer != nil {
		s.enqueuer.Enqueue(job.ID)
	}

	return job, nil
}

// GetJob mengambil detail job berdasarkan jobID.
func (s *TranscoderService) GetJob(ctx context.Context, jobID string) (*domain.TranscodeJob, error) {
	if jobID == "" {
		return nil, errors.New("job_id cannot be empty")
	}
	job, err := s.jobRepo.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// ListJobs mengembalikan semua transcode jobs.
func (s *TranscoderService) ListJobs(ctx context.Context) ([]*domain.TranscodeJob, error) {
	return s.jobRepo.ListJobs(ctx)
}
