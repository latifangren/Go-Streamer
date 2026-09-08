package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-streamer/internal/domain"
)

// VideoRepository mengimplementasikan domain.VideoRepository menggunakan SQLite.
type VideoRepository struct {
	db *sql.DB
}

// NewVideoRepository membuat instans baru VideoRepository dengan koneksi *sql.DB.
func NewVideoRepository(db *sql.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

// Create menyimpan record video baru ke tabel videos.
func (r *VideoRepository) Create(ctx context.Context, video *domain.Video) error {
	if video == nil {
		return fmt.Errorf("video cannot be nil: %w", domain.ErrInvalidInput)
	}
	if video.ID == "" {
		return fmt.Errorf("video ID cannot be empty: %w", domain.ErrInvalidInput)
	}

	if video.CreatedAt.IsZero() {
		video.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO videos (
			id, user_id, filename, original_name, file_path, file_size,
			duration_seconds, resolution, video_codec, audio_codec,
			fps, gop_size, is_passthrough_ready, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		video.ID,
		video.UserID,
		video.Filename,
		video.OriginalName,
		video.FilePath,
		video.FileSize,
		video.DurationSeconds,
		video.Resolution,
		video.VideoCodec,
		video.AudioCodec,
		video.FPS,
		video.GOPSize,
		video.IsPassthroughReady,
		video.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert video record: %w", err)
	}

	return nil
}

// GetByID mengambil data video berdasarkan ID. Mengembalikan domain.ErrNotFound jika tidak ditemukan.
func (r *VideoRepository) GetByID(ctx context.Context, id string) (*domain.Video, error) {
	if id == "" {
		return nil, domain.ErrNotFound
	}

	query := `
		SELECT id, user_id, filename, original_name, file_path, file_size,
		       duration_seconds, resolution, video_codec, audio_codec,
		       fps, gop_size, is_passthrough_ready, created_at
		FROM videos
		WHERE id = ?
	`

	var v domain.Video
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&v.ID,
		&v.UserID,
		&v.Filename,
		&v.OriginalName,
		&v.FilePath,
		&v.FileSize,
		&v.DurationSeconds,
		&v.Resolution,
		&v.VideoCodec,
		&v.AudioCodec,
		&v.FPS,
		&v.GOPSize,
		&v.IsPassthroughReady,
		&v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query video by ID: %w", err)
	}

	return &v, nil
}

// GetByUserID mengambil seluruh video milik user tertentu, diurutkan descending berdasarkan waktu pembuatan.
func (r *VideoRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Video, error) {
	query := `
		SELECT id, user_id, filename, original_name, file_path, file_size,
		       duration_seconds, resolution, video_codec, audio_codec,
		       fps, gop_size, is_passthrough_ready, created_at
		FROM videos
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos by user ID: %w", err)
	}
	defer rows.Close()

	videos := make([]*domain.Video, 0)
	for rows.Next() {
		var v domain.Video
		if err := rows.Scan(
			&v.ID,
			&v.UserID,
			&v.Filename,
			&v.OriginalName,
			&v.FilePath,
			&v.FileSize,
			&v.DurationSeconds,
			&v.Resolution,
			&v.VideoCodec,
			&v.AudioCodec,
			&v.FPS,
			&v.GOPSize,
			&v.IsPassthroughReady,
			&v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan video row: %w", err)
		}
		videos = append(videos, &v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during video rows iteration: %w", err)
	}

	return videos, nil
}

// ListAll mengambil seluruh daftar video yang tersimpan di sistem.
func (r *VideoRepository) ListAll(ctx context.Context) ([]*domain.Video, error) {
	query := `
		SELECT id, user_id, filename, original_name, file_path, file_size,
		       duration_seconds, resolution, video_codec, audio_codec,
		       fps, gop_size, is_passthrough_ready, created_at
		FROM videos
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all videos: %w", err)
	}
	defer rows.Close()

	videos := make([]*domain.Video, 0)
	for rows.Next() {
		var v domain.Video
		if err := rows.Scan(
			&v.ID,
			&v.UserID,
			&v.Filename,
			&v.OriginalName,
			&v.FilePath,
			&v.FileSize,
			&v.DurationSeconds,
			&v.Resolution,
			&v.VideoCodec,
			&v.AudioCodec,
			&v.FPS,
			&v.GOPSize,
			&v.IsPassthroughReady,
			&v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan video row: %w", err)
		}
		videos = append(videos, &v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during video rows iteration: %w", err)
	}

	return videos, nil
}

// Update memperbarui metadata video yang sudah ada. Mengembalikan domain.ErrNotFound jika video tidak ditemukan.
func (r *VideoRepository) Update(ctx context.Context, video *domain.Video) error {
	if video == nil {
		return fmt.Errorf("video cannot be nil: %w", domain.ErrInvalidInput)
	}
	if video.ID == "" {
		return fmt.Errorf("video ID cannot be empty: %w", domain.ErrInvalidInput)
	}

	query := `
		UPDATE videos SET
			user_id = ?,
			filename = ?,
			original_name = ?,
			file_path = ?,
			file_size = ?,
			duration_seconds = ?,
			resolution = ?,
			video_codec = ?,
			audio_codec = ?,
			fps = ?,
			gop_size = ?,
			is_passthrough_ready = ?
		WHERE id = ?
	`

	res, err := r.db.ExecContext(ctx, query,
		video.UserID,
		video.Filename,
		video.OriginalName,
		video.FilePath,
		video.FileSize,
		video.DurationSeconds,
		video.Resolution,
		video.VideoCodec,
		video.AudioCodec,
		video.FPS,
		video.GOPSize,
		video.IsPassthroughReady,
		video.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete menghapus record video dari database berdasarkan ID. Mengembalikan domain.ErrNotFound jika tidak ditemukan.
func (r *VideoRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return domain.ErrNotFound
	}

	res, err := r.db.ExecContext(ctx, "DELETE FROM videos WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetTotalStorageBytes menghitung total ukuran file (file_size) semua video yang tersimpan dalam database.
func (r *VideoRepository) GetTotalStorageBytes(ctx context.Context) (int64, error) {
	var total int64
	query := "SELECT COALESCE(SUM(file_size), 0) FROM videos"
	err := r.db.QueryRowContext(ctx, query).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total storage bytes: %w", err)
	}

	return total, nil
}
