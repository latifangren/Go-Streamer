package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"go-streamer/internal/domain"
	"go-streamer/internal/runner"
)

// VideoRepository mendefinisikan abstraksi penyimpanan data video yang dibutuhkan VideoService.
type VideoRepository interface {
	Create(ctx context.Context, video *domain.Video) error
	GetByID(ctx context.Context, id string) (*domain.Video, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.Video, error)
	ListAll(ctx context.Context) ([]*domain.Video, error)
	Update(ctx context.Context, video *domain.Video) error
	Delete(ctx context.Context, id string) error
	GetTotalStorageBytes(ctx context.Context) (int64, error)
}

// VideoService mengelola file media video fisik, probing codec, snapshot thumbnail, dan penyimpanan database.
type VideoService struct {
	repo        VideoRepository
	videoDir    string
	cacheDir    string
	ffprobePath string
	ffmpegPath  string
}

// NewVideoService membuat instans baru VideoService dan memastikan direktori video dan cache tersedia.
func NewVideoService(repo VideoRepository, videoDir, cacheDir, ffprobePath, ffmpegPath string) *VideoService {
	if videoDir == "" {
		videoDir = "data/videos"
	}
	if cacheDir == "" {
		cacheDir = "data/cache"
	}

	_ = os.MkdirAll(videoDir, 0755)
	_ = os.MkdirAll(cacheDir, 0755)

	return &VideoService{
		repo:        repo,
		videoDir:    videoDir,
		cacheDir:    cacheDir,
		ffprobePath: ffprobePath,
		ffmpegPath:  ffmpegPath,
	}
}

// SaveUploadedVideo menyimpan video yang diupload ke storage fisik, melakukan analisis ffprobe, ekstraksi snapshot thumbnail, dan menyimpan metadata ke database.
func (s *VideoService) SaveUploadedVideo(ctx context.Context, userID, originalName string, src io.Reader) (*domain.Video, error) {
	userID = strings.TrimSpace(userID)
	originalName = strings.TrimSpace(originalName)

	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty: %w", domain.ErrInvalidInput)
	}
	if originalName == "" {
		return nil, fmt.Errorf("original name cannot be empty: %w", domain.ErrInvalidInput)
	}
	if src == nil {
		return nil, fmt.Errorf("source reader cannot be nil: %w", domain.ErrInvalidInput)
	}

	// 1. Generate UUID v4
	id := uuid.New().String()

	// 2. Tentukan nama file dan path tujuan penyimpanan
	sanitizedName := sanitizeFilename(originalName)
	filename := fmt.Sprintf("%s_%s", id, sanitizedName)
	destPath := filepath.Join(s.videoDir, filename)

	if err := os.MkdirAll(s.videoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create video directory: %w", err)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination video file: %w", err)
	}

	fileSize, err := io.Copy(outFile, src)
	_ = outFile.Close()
	if err != nil {
		_ = os.Remove(destPath)
		return nil, fmt.Errorf("failed to save video content: %w", err)
	}

	// 3. Panggil runner.ProbeVideo untuk memeriksa metadata codec video/audio
	duration := 0.0
	resolution := "1920x1080"
	videoCodec := "h264"
	audioCodec := "aac"
	fps := 30.0
	gopSize := 2.0
	isPassthroughReady := false

	if s.ffprobePath != "" {
		probed, probeErr := runner.ProbeVideo(ctx, s.ffprobePath, destPath)
		if probeErr == nil && probed != nil {
			duration = probed.DurationSeconds
			if probed.Resolution != "" {
				resolution = probed.Resolution
			}
			if probed.VideoCodec != "" {
				videoCodec = probed.VideoCodec
			}
			if probed.AudioCodec != "" {
				audioCodec = probed.AudioCodec
			}
			if probed.FPS > 0 {
				fps = probed.FPS
			}
			if probed.GOPSize > 0 {
				gopSize = probed.GOPSize
			}
			isPassthroughReady = probed.IsPassthroughReady
		}
	}

	// 4. Panggil runner.ExtractSnapshot untuk menghasilkan thumbnail JPEG
	thumbPath := filepath.Join(s.cacheDir, fmt.Sprintf("thumb_%s.jpg", id))
	if s.ffmpegPath != "" {
		_ = os.MkdirAll(s.cacheDir, 0755)
		_ = runner.ExtractSnapshot(ctx, s.ffmpegPath, destPath, thumbPath)
	}

	// 5. Simpan entitas video ke database via repository
	video := &domain.Video{
		ID:                 id,
		UserID:             userID,
		Filename:           filename,
		OriginalName:       originalName,
		FilePath:           destPath,
		FileSize:           fileSize,
		DurationSeconds:    duration,
		Resolution:         resolution,
		VideoCodec:         videoCodec,
		AudioCodec:         audioCodec,
		FPS:                fps,
		GOPSize:            gopSize,
		IsPassthroughReady: isPassthroughReady,
		CreatedAt:          time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, video); err != nil {
		_ = os.Remove(destPath)
		_ = os.Remove(thumbPath)
		return nil, fmt.Errorf("failed to insert video record: %w", err)
	}

	return video, nil
}

// ListVideos mengambil daftar video. Jika userID diberikan, hanya mengambil video milik user tersebut.
func (s *VideoService) ListVideos(ctx context.Context, userID string) ([]*domain.Video, error) {
	if userID != "" {
		return s.repo.GetByUserID(ctx, userID)
	}
	return s.repo.ListAll(ctx)
}

// GetVideo mengambil informasi detail sebuah video berdasarkan ID.
func (s *VideoService) GetVideo(ctx context.Context, id string) (*domain.Video, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.GetByID(ctx, id)
}

// RenameVideo mengubah original_name dari video.
func (s *VideoService) RenameVideo(ctx context.Context, id, newName string) error {
	id = strings.TrimSpace(id)
	newName = strings.TrimSpace(newName)

	if id == "" || newName == "" {
		return domain.ErrInvalidInput
	}

	video, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	video.OriginalName = newName
	return s.repo.Update(ctx, video)
}

// DeleteVideo menghapus file fisik di disk (video & thumbnail) dan record di database.
func (s *VideoService) DeleteVideo(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.ErrInvalidInput
	}

	video, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Hapus file fisik di storage (abaikan error jika file sudah tidak ada)
	if video.FilePath != "" {
		_ = os.Remove(video.FilePath)
	}
	thumbPath := filepath.Join(s.cacheDir, fmt.Sprintf("thumb_%s.jpg", id))
	_ = os.Remove(thumbPath)

	return nil
}

// GetStorageStats mengembalikan total ukuran bytes dan jumlah total video yang tersimpan.
func (s *VideoService) GetStorageStats(ctx context.Context) (totalBytes int64, count int, err error) {
	totalBytes, err = s.repo.GetTotalStorageBytes(ctx)
	if err != nil {
		return 0, 0, err
	}

	allVideos, err := s.repo.ListAll(ctx)
	if err != nil {
		return 0, 0, err
	}

	return totalBytes, len(allVideos), nil
}

// sanitizeFilename membersihkan nama file dari karakter ilegal path.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	res := b.String()
	if res == "" || res == "." {
		res = "video.mp4"
	}
	return res
}
