package transcoder

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

type mockTranscoderRepo struct {
	mu   sync.Mutex
	jobs map[string]*domain.TranscodeJob
}

func newMockTranscoderRepo() *mockTranscoderRepo {
	return &mockTranscoderRepo{
		jobs: make(map[string]*domain.TranscodeJob),
	}
}

func (m *mockTranscoderRepo) CreateJob(ctx context.Context, job *domain.TranscodeJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *mockTranscoderRepo) GetJobByID(ctx context.Context, id string) (*domain.TranscodeJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	// return copy
	clone := *j
	return &clone, nil
}

func (m *mockTranscoderRepo) UpdateJob(ctx context.Context, job *domain.TranscodeJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.jobs[job.ID]; !ok {
		return domain.ErrNotFound
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *mockTranscoderRepo) ListJobs(ctx context.Context) ([]*domain.TranscodeJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*domain.TranscodeJob
	for _, j := range m.jobs {
		list = append(list, j)
	}
	return list, nil
}

func (m *mockTranscoderRepo) GetActiveOrPendingJob(ctx context.Context) (*domain.TranscodeJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if j.Status == domain.TranscodeProcessing || j.Status == domain.TranscodeQueued {
			clone := *j
			return &clone, nil
		}
	}
	return nil, nil
}

type mockVideoRepo struct {
	mu     sync.Mutex
	videos map[string]*domain.Video
}

func newMockVideoRepo() *mockVideoRepo {
	return &mockVideoRepo{
		videos: make(map[string]*domain.Video),
	}
}

func (m *mockVideoRepo) GetByID(ctx context.Context, id string) (*domain.Video, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.videos[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *v
	return &clone, nil
}

func (m *mockVideoRepo) Update(ctx context.Context, v *domain.Video) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.videos[v.ID]; !ok {
		return domain.ErrNotFound
	}
	m.videos[v.ID] = v
	return nil
}

// ----------------------------------------------------------------------------
// 1. Tests for Helper Functions (Time & Scanner Parsing)
// ----------------------------------------------------------------------------

func TestParseFFmpegTime(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"frame=  120 fps= 30 q=28.0 size=    1024kB time=00:00:04.00 bitrate=2097.2kbits/s speed=1.0x", 4.0},
		{"frame=  600 fps= 30 q=28.0 size=    5120kB time=00:01:25.50 bitrate=2097.2kbits/s speed=1.0x", 85.5},
		{"time=01:02:03.456", 3723.456},
		{"no time match here", 0.0},
	}

	for _, tt := range tests {
		got := ParseFFmpegTime(tt.input)
		if math.Abs(got-tt.expected) > 0.001 {
			t.Errorf("ParseFFmpegTime(%q) = %v; want %v", tt.input, got, tt.expected)
		}
	}
}

func TestScanLinesOrCarriageReturns(t *testing.T) {
	data := []byte("line1\rline2\nline3\r\n")
	advance, token, err := scanLinesOrCarriageReturns(data, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(token) != "line1" || advance != 6 {
		t.Errorf("unexpected first token: %s, advance: %d", token, advance)
	}

	data = data[advance:]
	advance, token, err = scanLinesOrCarriageReturns(data, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(token) != "line2" || advance != 6 {
		t.Errorf("unexpected second token: %s, advance: %d", token, advance)
	}
}

// ----------------------------------------------------------------------------
// 2. Tests for CodecFixer Lifecycle
// ----------------------------------------------------------------------------

func TestCodecFixer_Lifecycle_SourceNotFound(t *testing.T) {
	jobRepo := newMockTranscoderRepo()
	videoRepo := newMockVideoRepo()

	fixer := NewCodecFixer("ffmpeg", jobRepo, videoRepo)
	fixer.Start()
	defer fixer.Stop()

	// Video ada di DB tetapi filenya tidak ada di disk
	video := &domain.Video{
		ID:              "vid-1",
		Filename:        "missing.mp4",
		FilePath:        "/tmp/non_existent_file_987654.mp4",
		DurationSeconds: 60.0,
	}
	videoRepo.videos[video.ID] = video

	job := &domain.TranscodeJob{
		ID:             "job-1",
		SourceVideoID:  video.ID,
		TargetFilePath: "/tmp/fixed_missing.mp4",
		Status:         domain.TranscodeQueued,
		SourceVideo:    video,
	}
	_ = jobRepo.CreateJob(context.Background(), job)

	fixer.Enqueue(job.ID)

	// Berikan waktu sejenak agar worker memproses
	time.Sleep(100 * time.Millisecond)

	updatedJob, err := jobRepo.GetJobByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if updatedJob.Status != domain.TranscodeFailed {
		t.Errorf("expected status %s, got %s", domain.TranscodeFailed, updatedJob.Status)
	}
	if updatedJob.ErrorMessage == "" {
		t.Errorf("expected error message to be set for missing source file")
	}
}

func TestCodecFixer_Lifecycle_SuccessMockFFmpeg(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fixer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Buat fake source file
	sourcePath := filepath.Join(tmpDir, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("fake-video-bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(tmpDir, "fixed_source.mp4")

	jobRepo := newMockTranscoderRepo()
	videoRepo := newMockVideoRepo()

	video := &domain.Video{
		ID:                 "vid-success",
		Filename:           "source.mp4",
		FilePath:           sourcePath,
		DurationSeconds:    10.0,
		VideoCodec:         "hevc",
		AudioCodec:         "opus",
		IsPassthroughReady: false,
	}
	videoRepo.videos[video.ID] = video

	job := &domain.TranscodeJob{
		ID:             "job-success",
		SourceVideoID:  video.ID,
		TargetFilePath: targetPath,
		Status:         domain.TranscodeQueued,
		SourceVideo:    video,
	}
	_ = jobRepo.CreateJob(context.Background(), job)

	// Buat script mock executable ffmpeg yang menghasilkan targetPath dan exit 0
	var mockFFmpegPath string
	if os.PathSeparator == '\\' {
		// Windows batch script
		mockFFmpegPath = filepath.Join(tmpDir, "mock_ffmpeg.bat")
		batContent := "@echo off\r\necho frame= 300 fps=30 time=00:00:10.00 >&2\r\ntype nul > \"%~21\"\r\nexit /b 0\r\n"
		if err := os.WriteFile(mockFFmpegPath, []byte(batContent), 0755); err != nil {
			t.Fatal(err)
		}
	} else {
		// Unix shell script
		mockFFmpegPath = filepath.Join(tmpDir, "mock_ffmpeg.sh")
		shContent := "#!/bin/sh\necho 'frame= 300 fps=30 time=00:00:10.00' >&2\ntouch \"$18\"\nexit 0\n"
		if err := os.WriteFile(mockFFmpegPath, []byte(shContent), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Gunakan CodecFixer dengan direct call processJob
	fixer := NewCodecFixer(mockFFmpegPath, jobRepo, videoRepo)
	fixer.processJob(job.ID)

	updatedJob, err := jobRepo.GetJobByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if updatedJob.Status != domain.TranscodeCompleted {
		t.Errorf("expected status %s, got %s (err: %s)", domain.TranscodeCompleted, updatedJob.Status, updatedJob.ErrorMessage)
	}
	if updatedJob.ProgressPercent != 100.0 {
		t.Errorf("expected progress 100.0, got %f", updatedJob.ProgressPercent)
	}

	// Verifikasi update ke model Video
	updatedVideo, err := videoRepo.GetByID(context.Background(), video.ID)
	if err != nil {
		t.Fatalf("failed to get updated video: %v", err)
	}

	if !updatedVideo.IsPassthroughReady {
		t.Errorf("expected IsPassthroughReady to be true")
	}
	if updatedVideo.VideoCodec != "h264" {
		t.Errorf("expected video_codec h264, got %s", updatedVideo.VideoCodec)
	}
	if updatedVideo.AudioCodec != "aac" {
		t.Errorf("expected audio_codec aac, got %s", updatedVideo.AudioCodec)
	}
	if updatedVideo.FilePath != targetPath {
		t.Errorf("expected file_path %s, got %s", targetPath, updatedVideo.FilePath)
	}
}

func TestCodecFixer_GracefulStop(t *testing.T) {
	jobRepo := newMockTranscoderRepo()
	videoRepo := newMockVideoRepo()

	fixer := NewCodecFixer("ffmpeg", jobRepo, videoRepo)
	fixer.Start()

	// Pastikan worker bisa di-stop tanpa hang
	stopped := make(chan struct{})
	go func() {
		fixer.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		// Selesai dengan sukses
	case <-time.After(2 * time.Second):
		t.Fatalf("fixer.Stop() timed out")
	}
}
