package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-streamer/internal/domain"
)

// SlotRepository mengelola persistensi entitas StreamSlot di SQLite.
type SlotRepository struct {
	db *sql.DB
}

// NewSlotRepository membuat instans baru SlotRepository.
func NewSlotRepository(db *sql.DB) *SlotRepository {
	return &SlotRepository{db: db}
}

// ListAll mengambil seluruh slot streaming yang terdaftar, diurutkan berdasarkan nomor slot.
func (r *SlotRepository) ListAll(ctx context.Context) ([]*domain.StreamSlot, error) {
	query := `
		SELECT s.id, s.slot_number, s.name, s.status, s.source_type, s.video_id,
		       s.target_platform, s.rtmp_url, s.stream_key, s.mode, s.quality, s.preset,
		       s.loop_playback, s.max_duration_minutes, s.auto_restart, s.enable_overlay,
		       s.overlay_clock_wib, s.overlay_watermark, s.overlay_watermark_pos,
		       s.last_pts_sync_ms, s.last_keyframe_gop_s, s.last_net_latency_ms,
		       s.last_snapshot_path, s.last_snapshot_at, s.created_at, s.updated_at,
		       v.id, v.user_id, v.filename, v.original_name, v.file_path, v.file_size,
		       v.duration_seconds, v.resolution, v.video_codec, v.audio_codec, v.fps,
		       v.gop_size, v.is_passthrough_ready, v.created_at
		FROM stream_slots s
		LEFT JOIN videos v ON s.video_id = v.id
		ORDER BY s.slot_number ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query stream slots: %w", err)
	}
	defer rows.Close()

	slots := make([]*domain.StreamSlot, 0)
	for rows.Next() {
		slot, err := scanSlotWithVideo(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stream slot: %w", err)
		}
		slots = append(slots, slot)
	}

	return slots, nil
}

// GetByID mengambil stream slot berdasarkan id numeriknya.
func (r *SlotRepository) GetByID(ctx context.Context, id int64) (*domain.StreamSlot, error) {
	query := `
		SELECT s.id, s.slot_number, s.name, s.status, s.source_type, s.video_id,
		       s.target_platform, s.rtmp_url, s.stream_key, s.mode, s.quality, s.preset,
		       s.loop_playback, s.max_duration_minutes, s.auto_restart, s.enable_overlay,
		       s.overlay_clock_wib, s.overlay_watermark, s.overlay_watermark_pos,
		       s.last_pts_sync_ms, s.last_keyframe_gop_s, s.last_net_latency_ms,
		       s.last_snapshot_path, s.last_snapshot_at, s.created_at, s.updated_at,
		       v.id, v.user_id, v.filename, v.original_name, v.file_path, v.file_size,
		       v.duration_seconds, v.resolution, v.video_codec, v.audio_codec, v.fps,
		       v.gop_size, v.is_passthrough_ready, v.created_at
		FROM stream_slots s
		LEFT JOIN videos v ON s.video_id = v.id
		WHERE s.id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	slot, err := scanSlotRowWithVideo(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query stream slot by ID: %w", err)
	}

	return slot, nil
}

// GetBySlotNumber mengambil stream slot berdasarkan nomor slot (1, 2, dst).
func (r *SlotRepository) GetBySlotNumber(ctx context.Context, slotNumber int) (*domain.StreamSlot, error) {
	query := `
		SELECT s.id, s.slot_number, s.name, s.status, s.source_type, s.video_id,
		       s.target_platform, s.rtmp_url, s.stream_key, s.mode, s.quality, s.preset,
		       s.loop_playback, s.max_duration_minutes, s.auto_restart, s.enable_overlay,
		       s.overlay_clock_wib, s.overlay_watermark, s.overlay_watermark_pos,
		       s.last_pts_sync_ms, s.last_keyframe_gop_s, s.last_net_latency_ms,
		       s.last_snapshot_path, s.last_snapshot_at, s.created_at, s.updated_at,
		       v.id, v.user_id, v.filename, v.original_name, v.file_path, v.file_size,
		       v.duration_seconds, v.resolution, v.video_codec, v.audio_codec, v.fps,
		       v.gop_size, v.is_passthrough_ready, v.created_at
		FROM stream_slots s
		LEFT JOIN videos v ON s.video_id = v.id
		WHERE s.slot_number = ?
	`

	row := r.db.QueryRowContext(ctx, query, slotNumber)
	slot, err := scanSlotRowWithVideo(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query stream slot by slot number: %w", err)
	}

	return slot, nil
}

// Update memperbarui konfigurasi stream slot.
func (r *SlotRepository) Update(ctx context.Context, slot *domain.StreamSlot) error {
	if slot == nil {
		return fmt.Errorf("slot cannot be nil: %w", domain.ErrInvalidInput)
	}

	now := time.Now().UTC()
	slot.UpdatedAt = now

	query := `
		UPDATE stream_slots SET
			name = ?,
			source_type = ?,
			video_id = ?,
			target_platform = ?,
			rtmp_url = ?,
			stream_key = ?,
			mode = ?,
			quality = ?,
			preset = ?,
			loop_playback = ?,
			max_duration_minutes = ?,
			auto_restart = ?,
			enable_overlay = ?,
			overlay_clock_wib = ?,
			overlay_watermark = ?,
			overlay_watermark_pos = ?,
			updated_at = ?
		WHERE id = ?
	`

	res, err := r.db.ExecContext(ctx, query,
		slot.Name,
		slot.SourceType,
		slot.VideoID,
		slot.TargetPlatform,
		slot.RTMPURL,
		slot.StreamKey,
		slot.Mode,
		slot.Quality,
		slot.Preset,
		slot.LoopPlayback,
		slot.MaxDurationMinutes,
		slot.AutoRestart,
		slot.EnableOverlay,
		slot.OverlayClockWIB,
		slot.OverlayWatermark,
		slot.OverlayWatermarkPos,
		slot.UpdatedAt,
		slot.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update stream slot: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// UpdateStatus memperbarui status operasional slot (idle, starting, running, error).
func (r *SlotRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	now := time.Now().UTC()
	query := `UPDATE stream_slots SET status = ?, updated_at = ? WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, status, now, id)
	if err != nil {
		return fmt.Errorf("failed to update slot status: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func scanSlotWithVideo(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.StreamSlot, error) {
	var s domain.StreamSlot
	var lastSnapAt sql.NullTime
	var (
		vID, vUserID, vFilename, vOrigName, vFilePath, vRes, vVCodec, vACodec sql.NullString
		vSize                                                                 sql.NullInt64
		vDur, vFPS, vGOP                                                      sql.NullFloat64
		vPassReady                                                            sql.NullBool
		vCreatedAt                                                            sql.NullTime
	)

	err := scanner.Scan(
		&s.ID, &s.SlotNumber, &s.Name, &s.Status, &s.SourceType, &s.VideoID,
		&s.TargetPlatform, &s.RTMPURL, &s.StreamKey, &s.Mode, &s.Quality, &s.Preset,
		&s.LoopPlayback, &s.MaxDurationMinutes, &s.AutoRestart, &s.EnableOverlay,
		&s.OverlayClockWIB, &s.OverlayWatermark, &s.OverlayWatermarkPos,
		&s.LastPTSSyncMS, &s.LastKeyframeGOPS, &s.LastNetLatencyMS,
		&s.LastSnapshotPath, &lastSnapAt, &s.CreatedAt, &s.UpdatedAt,
		&vID, &vUserID, &vFilename, &vOrigName, &vFilePath, &vSize,
		&vDur, &vRes, &vVCodec, &vACodec, &vFPS,
		&vGOP, &vPassReady, &vCreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if lastSnapAt.Valid {
		t := lastSnapAt.Time
		s.LastSnapshotAt = &t
	}

	if vID.Valid {
		s.Video = &domain.Video{
			ID:                 vID.String,
			UserID:             vUserID.String,
			Filename:           vFilename.String,
			OriginalName:       vOrigName.String,
			FilePath:           vFilePath.String,
			FileSize:           vSize.Int64,
			DurationSeconds:    vDur.Float64,
			Resolution:         vRes.String,
			VideoCodec:         vVCodec.String,
			AudioCodec:         vACodec.String,
			FPS:                vFPS.Float64,
			GOPSize:            vGOP.Float64,
			IsPassthroughReady: vPassReady.Bool,
			CreatedAt:          vCreatedAt.Time,
		}
	}

	return &s, nil
}

func scanSlotRowWithVideo(row *sql.Row) (*domain.StreamSlot, error) {
	return scanSlotWithVideo(row)
}
