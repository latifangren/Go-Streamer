package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-streamer/internal/domain"
)

// ScheduleRepository mengelola persistensi entitas Schedule di SQLite.
type ScheduleRepository struct {
	db *sql.DB
}

// NewScheduleRepository membuat instans baru ScheduleRepository.
func NewScheduleRepository(db *sql.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// ListAll mengambil seluruh jadwal streaming yang tersimpan.
func (r *ScheduleRepository) ListAll(ctx context.Context) ([]*domain.Schedule, error) {
	query := `
		SELECT sc.id, sc.title, sc.slot_id, sc.cron_expr, sc.duration_minutes, sc.overlap_guard_policy,
		       sc.is_enabled, sc.last_run_at, sc.next_run_at, sc.created_at,
		       s.slot_number, s.name, s.status, s.rtmp_url
		FROM schedules sc
		LEFT JOIN stream_slots s ON sc.slot_id = s.id
		ORDER BY sc.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*domain.Schedule, 0)
	for rows.Next() {
		var sc domain.Schedule
		var lastRun, nextRun sql.NullTime
		var sSlotNum sql.NullInt64
		var sName, sStatus, sRTMP sql.NullString

		if err := rows.Scan(
			&sc.ID, &sc.Title, &sc.SlotID, &sc.CronExpr, &sc.DurationMinutes, &sc.OverlapGuardPolicy,
			&sc.IsEnabled, &lastRun, &nextRun, &sc.CreatedAt,
			&sSlotNum, &sName, &sStatus, &sRTMP,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule row: %w", err)
		}

		if lastRun.Valid {
			t := lastRun.Time
			sc.LastRunAt = &t
		}
		if nextRun.Valid {
			t := nextRun.Time
			sc.NextRunAt = &t
		}

		if sName.Valid {
			sc.Slot = &domain.StreamSlot{
				ID:             sc.SlotID,
				SlotNumber:     int(sSlotNum.Int64),
				Name:           sName.String,
				Status:         sStatus.String,
				RTMPURL:        sRTMP.String,
			}
		}

		schedules = append(schedules, &sc)
	}

	return schedules, nil
}

// GetByID mengambil satu jadwal streaming berdasarkan ID.
func (r *ScheduleRepository) GetByID(ctx context.Context, id string) (*domain.Schedule, error) {
	query := `
		SELECT sc.id, sc.title, sc.slot_id, sc.cron_expr, sc.duration_minutes, sc.overlap_guard_policy,
		       sc.is_enabled, sc.last_run_at, sc.next_run_at, sc.created_at,
		       s.slot_number, s.name, s.status, s.rtmp_url
		FROM schedules sc
		LEFT JOIN stream_slots s ON sc.slot_id = s.id
		WHERE sc.id = ?
	`

	var sc domain.Schedule
	var lastRun, nextRun sql.NullTime
	var sSlotNum sql.NullInt64
	var sName, sStatus, sRTMP sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sc.ID, &sc.Title, &sc.SlotID, &sc.CronExpr, &sc.DurationMinutes, &sc.OverlapGuardPolicy,
		&sc.IsEnabled, &lastRun, &nextRun, &sc.CreatedAt,
		&sSlotNum, &sName, &sStatus, &sRTMP,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query schedule by ID: %w", err)
	}

	if lastRun.Valid {
		t := lastRun.Time
		sc.LastRunAt = &t
	}
	if nextRun.Valid {
		t := nextRun.Time
		sc.NextRunAt = &t
	}

	if sName.Valid {
		sc.Slot = &domain.StreamSlot{
			ID:             sc.SlotID,
			SlotNumber:     int(sSlotNum.Int64),
			Name:           sName.String,
			Status:         sStatus.String,
			RTMPURL:        sRTMP.String,
		}
	}

	return &sc, nil
}

// Create menyimpan jadwal streaming baru.
func (r *ScheduleRepository) Create(ctx context.Context, sc *domain.Schedule) error {
	if sc == nil {
		return fmt.Errorf("schedule cannot be nil: %w", domain.ErrInvalidInput)
	}
	if sc.ID == "" {
		return fmt.Errorf("schedule ID cannot be empty: %w", domain.ErrInvalidInput)
	}

	if sc.CreatedAt.IsZero() {
		sc.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO schedules (
			id, title, slot_id, cron_expr, duration_minutes, overlap_guard_policy,
			is_enabled, last_run_at, next_run_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		sc.ID,
		sc.Title,
		sc.SlotID,
		sc.CronExpr,
		sc.DurationMinutes,
		sc.OverlapGuardPolicy,
		sc.IsEnabled,
		sc.LastRunAt,
		sc.NextRunAt,
		sc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert schedule record: %w", err)
	}

	return nil
}

// Delete menghapus jadwal streaming berdasarkan ID.
func (r *ScheduleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM schedules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
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
