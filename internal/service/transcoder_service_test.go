package service

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"go-streamer/internal/domain"
)

type mockTranscodeJobRepo struct {
	mu   sync.Mutex
	jobs map[string]*domain.TranscodeJob
}

func newMockTranscodeJobRepo() *mockTranscodeJobRepo {
	return &mockTranscodeJobRepo{
		jobs: make(map[string]*domain.TranscodeJob),
	}
}

func (m *mockTranscodeJobRepo) CreateJob(ctx context.Context, job *domain.TranscodeJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *mockTranscodeJobRepo) GetJobByID(ctx context.Context, id string) (*domain.TranscodeJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *j
	return &clone, nil
}

func (m *mockTranscodeJobRepo) UpdateJob(ctx context.Context, job *domain.TranscodeJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[job.ID]; !ok {
		return domain.ErrNotFound
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *mockTranscodeJobRepo) ListJobs(ctx context.Context) ([]*domain.TranscodeJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*domain.TranscodeJob
	for _, j := range m.jobs {
		list = append(list, j)
	}
	return list, nil
}

type mockVideoLookupRepo struct {
	mu     sync.Mutex
	videos map[string]*domain.Video
}

func newMockVideoLookupRepo() *mockVideoLookupRepo {
	return &mockVideoLookupRepo{
		videos: make(map[string]*domain.Video),
	}
}

func (m *mockVideoLookupRepo) GetByID(ctx context.Context, id string) (*domain.Video, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.videos[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *v
	return &clone, nil
}

type mockJobEnqueuer struct {
	mu         sync.Mutex
	enqueuedID []string
}

func (m *mockJobEnqueuer) Enqueue(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enqueuedID = append(m.enqueuedID, jobID)
}

func (m *mockJobEnqueuer) LastEnqueued() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.enqueuedID) == 0 {
		return ""
	}
	return m.enqueuedID[len(m.enqueuedID)-1]
}

func TestTranscoderService_CreateFixJob(t *testing.T) {
	jobRepo := newMockTranscodeJobRepo()
	videoRepo := newMockVideoLookupRepo()
	enqueuer := &mockJobEnqueuer{}
	videoDir := "/data/videos"

	svc := NewTranscoderService(jobRepo, videoRepo, enqueuer, videoDir)

	// 1. Video not found
	_, err := svc.CreateFixJob(context.Background(), "unknown-vid")
	if err == nil {
		t.Errorf("expected error for non-existent video")
	}

	// 2. Video exists
	video := &domain.Video{
		ID:       "vid-101",
		Filename: "intro.mp4",
		FilePath: "/data/videos/intro.mp4",
	}
	videoRepo.videos[video.ID] = video

	job, err := svc.CreateFixJob(context.Background(), video.ID)
	if err != nil {
		t.Fatalf("failed to create fix job: %v", err)
	}

	expectedTarget := filepath.Join(videoDir, "fixed_vid-101_intro.mp4")
	if job.TargetFilePath != expectedTarget {
		t.Errorf("expected target path %s, got %s", expectedTarget, job.TargetFilePath)
	}
	if job.Status != domain.TranscodeQueued {
		t.Errorf("expected status queued, got %s", job.Status)
	}
	if enqueuer.LastEnqueued() != job.ID {
		t.Errorf("expected job %s to be enqueued", job.ID)
	}

	// 3. Duplicate active job prevention
	_, err = svc.CreateFixJob(context.Background(), video.ID)
	if err == nil {
		t.Errorf("expected error preventing duplicate active job")
	}
}

func TestTranscoderService_GetAndListJobs(t *testing.T) {
	jobRepo := newMockTranscodeJobRepo()
	videoRepo := newMockVideoLookupRepo()
	enqueuer := &mockJobEnqueuer{}
	svc := NewTranscoderService(jobRepo, videoRepo, enqueuer, "/tmp")

	job1 := &domain.TranscodeJob{
		ID:            "job-1",
		SourceVideoID: "vid-1",
		Status:        domain.TranscodeCompleted,
	}
	_ = jobRepo.CreateJob(context.Background(), job1)

	got, err := svc.GetJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("unexpected error getting job: %v", err)
	}
	if got.ID != "job-1" {
		t.Errorf("expected job-1, got %s", got.ID)
	}

	list, err := svc.ListJobs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error listing jobs: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 job, got %d", len(list))
	}
}
