package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-streamer/internal/domain"
)

type TranscoderRepository struct {
	db *sql.DB
}

func NewTranscoderRepository(db *sql.DB) *TranscoderRepository {
	return &TranscoderRepository{db: db}
}

func (r *TranscoderRepository) CreateJob(ctx context.Context, job *domain.TranscodeJob) error {
	if job == nil {
		return errors.New("job cannot be nil")
	}

	query := `
		INSERT INTO transcode_jobs (
			id, source_video_id, target_file_path, status, progress_percent, error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}
	if job.Status == "" {
		job.Status = domain.TranscodeQueued
	}

	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.SourceVideoID,
		job.TargetFilePath,
		job.Status,
		job.ProgressPercent,
		job.ErrorMessage,
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert transcode job: %w", err)
	}

	return nil
}

func (r *TranscoderRepository) GetJobByID(ctx context.Context, id string) (*domain.TranscodeJob, error) {
	query := `
		SELECT 
			tj.id, tj.source_video_id, tj.target_file_path, tj.status, tj.progress_percent, tj.error_message, tj.created_at, tj.updated_at,
			v.id, v.user_id, v.filename, v.original_name, v.file_path, v.file_size, v.duration_seconds, v.resolution, v.video_codec, v.audio_codec, v.fps, v.gop_size, v.is_passthrough_ready, v.created_at
		FROM transcode_jobs tj
		LEFT JOIN videos v ON tj.source_video_id = v.id
		WHERE tj.id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)

	var job domain.TranscodeJob
	var (
		vID                 sql.NullString
		vUserID             sql.NullString
		vFilename           sql.NullString
		vOrigName           sql.NullString
		vFilePath           sql.NullString
		vFileSize           sql.NullInt64
		vDuration           sql.NullFloat64
		vResolution         sql.NullString
		vVideoCodec         sql.NullString
		vAudioCodec         sql.NullString
		vFPS                sql.NullFloat64
		vGOPSize            sql.NullFloat64
		vIsPassthroughReady sql.NullBool
		vCreatedAt          sql.NullTime
	)

	err := row.Scan(
		&job.ID,
		&job.SourceVideoID,
		&job.TargetFilePath,
		&job.Status,
		&job.ProgressPercent,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
		&vID,
		&vUserID,
		&vFilename,
		&vOrigName,
		&vFilePath,
		&vFileSize,
		&vDuration,
		&vResolution,
		&vVideoCodec,
		&vAudioCodec,
		&vFPS,
		&vGOPSize,
		&vIsPassthroughReady,
		&vCreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("transcode job not found: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to query transcode job: %w", err)
	}

	if vID.Valid {
		job.SourceVideo = &domain.Video{
			ID:                 vID.String,
			UserID:             vUserID.String,
			Filename:           vFilename.String,
			OriginalName:       vOrigName.String,
			FilePath:           vFilePath.String,
			FileSize:           vFileSize.Int64,
			DurationSeconds:    vDuration.Float64,
			Resolution:         vResolution.String,
			VideoCodec:         vVideoCodec.String,
			AudioCodec:         vAudioCodec.String,
			FPS:                vFPS.Float64,
			GOPSize:            vGOPSize.Float64,
			IsPassthroughReady: vIsPassthroughReady.Bool,
			CreatedAt:          vCreatedAt.Time,
		}
	}

	return &job, nil
}

func (r *TranscoderRepository) UpdateJob(ctx context.Context, job *domain.TranscodeJob) error {
	if job == nil {
		return errors.New("job cannot be nil")
	}

	query := `
		UPDATE transcode_jobs
		SET status = ?, progress_percent = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`

	job.UpdatedAt = time.Now()

	res, err := r.db.ExecContext(ctx, query,
		job.Status,
		job.ProgressPercent,
		job.ErrorMessage,
		job.UpdatedAt,
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update transcode job: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("transcode job %s not found: %w", job.ID, domain.ErrNotFound)
	}

	return nil
}

func (r *TranscoderRepository) ListJobs(ctx context.Context) ([]*domain.TranscodeJob, error) {
	query := `
		SELECT 
			tj.id, tj.source_video_id, tj.target_file_path, tj.status, tj.progress_percent, tj.error_message, tj.created_at, tj.updated_at,
			v.id, v.user_id, v.filename, v.original_name, v.file_path, v.file_size, v.duration_seconds, v.resolution, v.video_codec, v.audio_codec, v.fps, v.gop_size, v.is_passthrough_ready, v.created_at
		FROM transcode_jobs tj
		LEFT JOIN videos v ON tj.source_video_id = v.id
		ORDER BY tj.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list transcode jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*domain.TranscodeJob
	for rows.Next() {
		var job domain.TranscodeJob
		var (
			vID                 sql.NullString
			vUserID             sql.NullString
			vFilename           sql.NullString
			vOrigName           sql.NullString
			vFilePath           sql.NullString
			vFileSize           sql.NullInt64
			vDuration           sql.NullFloat64
			vResolution         sql.NullString
			vVideoCodec         sql.NullString
			vAudioCodec         sql.NullString
			vFPS                sql.NullFloat64
			vGOPSize            sql.NullFloat64
			vIsPassthroughReady sql.NullBool
			vCreatedAt          sql.NullTime
		)

		if err := rows.Scan(
			&job.ID,
			&job.SourceVideoID,
			&job.TargetFilePath,
			&job.Status,
			&job.ProgressPercent,
			&job.ErrorMessage,
			&job.CreatedAt,
			&job.UpdatedAt,
			&vID,
			&vUserID,
			&vFilename,
			&vOrigName,
			&vFilePath,
			&vFileSize,
			&vDuration,
			&vResolution,
			&vVideoCodec,
			&vAudioCodec,
			&vFPS,
			&vGOPSize,
			&vIsPassthroughReady,
			&vCreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transcode job: %w", err)
		}

		if vID.Valid {
			job.SourceVideo = &domain.Video{
				ID:                 vID.String,
				UserID:             vUserID.String,
				Filename:           vFilename.String,
				OriginalName:       vOrigName.String,
				FilePath:           vFilePath.String,
				FileSize:           vFileSize.Int64,
				DurationSeconds:    vDuration.Float64,
				Resolution:         vResolution.String,
				VideoCodec:         vVideoCodec.String,
				AudioCodec:         vAudioCodec.String,
				FPS:                vFPS.Float64,
				GOPSize:            vGOPSize.Float64,
				IsPassthroughReady: vIsPassthroughReady.Bool,
				CreatedAt:          vCreatedAt.Time,
			}
		}

		jobs = append(jobs, &job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return jobs, nil
}

func (r *TranscoderRepository) GetActiveOrPendingJob(ctx context.Context) (*domain.TranscodeJob, error) {
	query := `
		SELECT 
			tj.id, tj.source_video_id, tj.target_file_path, tj.status, tj.progress_percent, tj.error_message, tj.created_at, tj.updated_at,
			v.id, v.user_id, v.filename, v.original_name, v.file_path, v.file_size, v.duration_seconds, v.resolution, v.video_codec, v.audio_codec, v.fps, v.gop_size, v.is_passthrough_ready, v.created_at
		FROM transcode_jobs tj
		LEFT JOIN videos v ON tj.source_video_id = v.id
		WHERE tj.status IN (?, ?)
		ORDER BY CASE WHEN tj.status = ? THEN 1 ELSE 2 END, tj.created_at ASC
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, domain.TranscodeProcessing, domain.TranscodeQueued, domain.TranscodeProcessing)

	var job domain.TranscodeJob
	var (
		vID                 sql.NullString
		vUserID             sql.NullString
		vFilename           sql.NullString
		vOrigName           sql.NullString
		vFilePath           sql.NullString
		vFileSize           sql.NullInt64
		vDuration           sql.NullFloat64
		vResolution         sql.NullString
		vVideoCodec         sql.NullString
		vAudioCodec         sql.NullString
		vFPS                sql.NullFloat64
		vGOPSize            sql.NullFloat64
		vIsPassthroughReady sql.NullBool
		vCreatedAt          sql.NullTime
	)

	err := row.Scan(
		&job.ID,
		&job.SourceVideoID,
		&job.TargetFilePath,
		&job.Status,
		&job.ProgressPercent,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
		&vID,
		&vUserID,
		&vFilename,
		&vOrigName,
		&vFilePath,
		&vFileSize,
		&vDuration,
		&vResolution,
		&vVideoCodec,
		&vAudioCodec,
		&vFPS,
		&vGOPSize,
		&vIsPassthroughReady,
		&vCreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No active or pending job
		}
		return nil, fmt.Errorf("failed to query active/pending job: %w", err)
	}

	if vID.Valid {
		job.SourceVideo = &domain.Video{
			ID:                 vID.String,
			UserID:             vUserID.String,
			Filename:           vFilename.String,
			OriginalName:       vOrigName.String,
			FilePath:           vFilePath.String,
			FileSize:           vFileSize.Int64,
			DurationSeconds:    vDuration.Float64,
			Resolution:         vResolution.String,
			VideoCodec:         vVideoCodec.String,
			AudioCodec:         vAudioCodec.String,
			FPS:                vFPS.Float64,
			GOPSize:            vGOPSize.Float64,
			IsPassthroughReady: vIsPassthroughReady.Bool,
			CreatedAt:          vCreatedAt.Time,
		}
	}

	return &job, nil
}
