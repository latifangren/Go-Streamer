package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"go-streamer/internal/domain"
)

type mockVideoRepo struct {
	mu     sync.Mutex
	videos map[string]*domain.Video
}

func newMockVideoRepo() *mockVideoRepo {
	return &mockVideoRepo{
		videos: make(map[string]*domain.Video),
	}
}

func (m *mockVideoRepo) Create(ctx context.Context, video *domain.Video) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.videos[video.ID]; exists {
		return domain.ErrAlreadyExists
	}
	// clone
	v := *video
	m.videos[video.ID] = &v
	return nil
}

func (m *mockVideoRepo) GetByID(ctx context.Context, id string) (*domain.Video, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, exists := m.videos[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	clone := *v
	return &clone, nil
}

func (m *mockVideoRepo) GetByUserID(ctx context.Context, userID string) ([]*domain.Video, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []*domain.Video
	for _, v := range m.videos {
		if v.UserID == userID {
			clone := *v
			list = append(list, &clone)
		}
	}
	return list, nil
}

func (m *mockVideoRepo) ListAll(ctx context.Context) ([]*domain.Video, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []*domain.Video
	for _, v := range m.videos {
		clone := *v
		list = append(list, &clone)
	}
	return list, nil
}

func (m *mockVideoRepo) Update(ctx context.Context, video *domain.Video) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.videos[video.ID]; !exists {
		return domain.ErrNotFound
	}
	clone := *video
	m.videos[video.ID] = &clone
	return nil
}

func (m *mockVideoRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.videos[id]; !exists {
		return domain.ErrNotFound
	}
	delete(m.videos, id)
	return nil
}

func (m *mockVideoRepo) GetTotalStorageBytes(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var total int64
	for _, v := range m.videos {
		total += v.FileSize
	}
	return total, nil
}

func setupTestService(t *testing.T) (*VideoService, *mockVideoRepo, string, string) {
	tmpDir := t.TempDir()
	videoDir := filepath.Join(tmpDir, "videos")
	cacheDir := filepath.Join(tmpDir, "cache")

	repo := newMockVideoRepo()
	svc := NewVideoService(repo, videoDir, cacheDir, "", "")

	return svc, repo, videoDir, cacheDir
}

func TestVideoService_SaveUploadedVideo(t *testing.T) {
	svc, _, videoDir, _ := setupTestService(t)
	ctx := context.Background()

	content := "dummy-video-binary-data-stream-content"
	reader := strings.NewReader(content)

	video, err := svc.SaveUploadedVideo(ctx, "usr_1", "promo test @ 2026.mp4", reader)
	if err != nil {
		t.Fatalf("SaveUploadedVideo failed: %v", err)
	}

	// 1. Verifikasi ID UUID v4
	if _, err := uuid.Parse(video.ID); err != nil {
		t.Errorf("expected valid UUID for video ID, got: %s", video.ID)
	}

	// 2. Verifikasi metadata dasar
	if video.UserID != "usr_1" {
		t.Errorf("expected userID usr_1, got: %s", video.UserID)
	}
	if video.OriginalName != "promo test @ 2026.mp4" {
		t.Errorf("expected originalName promo test @ 2026.mp4, got: %s", video.OriginalName)
	}
	if video.FileSize != int64(len(content)) {
		t.Errorf("expected fileSize %d, got: %d", len(content), video.FileSize)
	}

	// 3. Verifikasi file fisik ada di disk
	if _, err := os.Stat(video.FilePath); err != nil {
		t.Fatalf("expected video file at %s, got error: %v", video.FilePath, err)
	}

	savedBytes, err := os.ReadFile(video.FilePath)
	if err != nil {
		t.Fatalf("failed to read saved video file: %v", err)
	}
	if !bytes.Equal(savedBytes, []byte(content)) {
		t.Errorf("file content mismatch")
	}

	// 4. Verifikasi sanitasi nama file
	if !strings.HasPrefix(filepath.Dir(video.FilePath), videoDir) {
		t.Errorf("expected video inside videoDir %s, got %s", videoDir, video.FilePath)
	}
	if strings.Contains(filepath.Base(video.FilePath), "@") || strings.Contains(filepath.Base(video.FilePath), " ") {
		t.Errorf("filename not sanitized: %s", video.FilePath)
	}
}

func TestVideoService_SaveUploadedVideo_Validation(t *testing.T) {
	svc, _, _, _ := setupTestService(t)
	ctx := context.Background()

	reader := strings.NewReader("data")

	// Empty userID
	if _, err := svc.SaveUploadedVideo(ctx, "", "video.mp4", reader); err == nil {
		t.Errorf("expected error for empty userID")
	}

	// Empty originalName
	if _, err := svc.SaveUploadedVideo(ctx, "u1", "", reader); err == nil {
		t.Errorf("expected error for empty originalName")
	}

	// Nil reader
	if _, err := svc.SaveUploadedVideo(ctx, "u1", "video.mp4", nil); err == nil {
		t.Errorf("expected error for nil src")
	}
}

func TestVideoService_ListAndGet(t *testing.T) {
	svc, repo, _, _ := setupTestService(t)
	ctx := context.Background()

	v1 := &domain.Video{
		ID:           "vid-1",
		UserID:       "user-a",
		OriginalName: "video-a.mp4",
		FileSize:     1000,
		CreatedAt:    time.Now(),
	}
	v2 := &domain.Video{
		ID:           "vid-2",
		UserID:       "user-b",
		OriginalName: "video-b.mp4",
		FileSize:     2000,
		CreatedAt:    time.Now(),
	}
	_ = repo.Create(ctx, v1)
	_ = repo.Create(ctx, v2)

	// List all
	all, err := svc.ListVideos(ctx, "")
	if err != nil {
		t.Fatalf("ListVideos all failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 videos, got %d", len(all))
	}

	// List user-a
	userAVideos, err := svc.ListVideos(ctx, "user-a")
	if err != nil {
		t.Fatalf("ListVideos user-a failed: %v", err)
	}
	if len(userAVideos) != 1 || userAVideos[0].ID != "vid-1" {
		t.Errorf("expected 1 video for user-a, got %d", len(userAVideos))
	}

	// Get existing
	got, err := svc.GetVideo(ctx, "vid-1")
	if err != nil {
		t.Fatalf("GetVideo failed: %v", err)
	}
	if got.OriginalName != "video-a.mp4" {
		t.Errorf("expected video-a.mp4, got %s", got.OriginalName)
	}

	// Get non-existent
	if _, err := svc.GetVideo(ctx, "non-existent"); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Get invalid ID
	if _, err := svc.GetVideo(ctx, ""); err != domain.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestVideoService_RenameVideo(t *testing.T) {
	svc, repo, _, _ := setupTestService(t)
	ctx := context.Background()

	v := &domain.Video{
		ID:           "vid-rename",
		UserID:       "user-1",
		OriginalName: "old_name.mp4",
		FileSize:     1000,
		CreatedAt:    time.Now(),
	}
	_ = repo.Create(ctx, v)

	// Valid rename
	err := svc.RenameVideo(ctx, "vid-rename", "new_name.mp4")
	if err != nil {
		t.Fatalf("RenameVideo failed: %v", err)
	}

	fetched, _ := svc.GetVideo(ctx, "vid-rename")
	if fetched.OriginalName != "new_name.mp4" {
		t.Errorf("expected new_name.mp4, got %s", fetched.OriginalName)
	}

	// Invalid inputs
	if err := svc.RenameVideo(ctx, "", "abc.mp4"); err != domain.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput for empty ID, got %v", err)
	}
	if err := svc.RenameVideo(ctx, "vid-rename", "   "); err != domain.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput for empty name, got %v", err)
	}
	if err := svc.RenameVideo(ctx, "non-existent", "name.mp4"); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestVideoService_DeleteVideo(t *testing.T) {
	svc, repo, videoDir, cacheDir := setupTestService(t)
	ctx := context.Background()

	filePath := filepath.Join(videoDir, "test_file.mp4")
	thumbPath := filepath.Join(cacheDir, "thumb_vid-delete.jpg")

	_ = os.WriteFile(filePath, []byte("video file content"), 0644)
	_ = os.WriteFile(thumbPath, []byte("thumbnail content"), 0644)

	v := &domain.Video{
		ID:           "vid-delete",
		UserID:       "user-1",
		OriginalName: "test.mp4",
		FilePath:     filePath,
		FileSize:     1000,
		CreatedAt:    time.Now(),
	}
	_ = repo.Create(ctx, v)

	// Execute Delete
	err := svc.DeleteVideo(ctx, "vid-delete")
	if err != nil {
		t.Fatalf("DeleteVideo failed: %v", err)
	}

	// Verifikasi record terhapus dari repository
	if _, err := repo.GetByID(ctx, "vid-delete"); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound from repo, got %v", err)
	}

	// Verifikasi file fisik video terhapus dari disk
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected video file to be removed from disk")
	}

	// Verifikasi file fisik thumbnail terhapus dari disk
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Errorf("expected thumbnail file to be removed from disk")
	}

	// Delete non-existent
	if err := svc.DeleteVideo(ctx, "non-existent"); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for non-existent video, got %v", err)
	}

	// Delete empty ID
	if err := svc.DeleteVideo(ctx, ""); err != domain.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput for empty ID, got %v", err)
	}
}

func TestVideoService_GetStorageStats(t *testing.T) {
	svc, repo, _, _ := setupTestService(t)
	ctx := context.Background()

	_ = repo.Create(ctx, &domain.Video{ID: "v1", FileSize: 1024 * 1024 * 5})
	_ = repo.Create(ctx, &domain.Video{ID: "v2", FileSize: 1024 * 1024 * 15})

	totalBytes, count, err := svc.GetStorageStats(ctx)
	if err != nil {
		t.Fatalf("GetStorageStats failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	expectedBytes := int64((1024 * 1024 * 5) + (1024 * 1024 * 15))
	if totalBytes != expectedBytes {
		t.Errorf("expected totalBytes %d, got %d", expectedBytes, totalBytes)
	}
}
