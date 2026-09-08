package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

// New menginisialisasi koneksi SQLite CGO-free (modernc.org/sqlite) dengan mode WAL.
func New(dbPath string) (*DB, error) {
	// Pastikan direktori database ada
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori db %s: %w", dir, err)
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)", dbPath)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database sqlite: %w", err)
	}

	// Konfigurasi pool koneksi
	sqlDB.SetMaxOpenConns(1) // SQLite single-writer safe pool
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("gagal ping database: %w", err)
	}

	db := &DB{DB: sqlDB}
	if err := db.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrasi database gagal: %w", err)
	}

	if err := db.seedDefaults(ctx); err != nil {
		return nil, fmt.Errorf("seeding database default gagal: %w", err)
	}

	return db, nil
}

func (db *DB) migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS videos (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			original_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			duration_seconds REAL DEFAULT 0,
			resolution TEXT,
			video_codec TEXT,
			audio_codec TEXT,
			fps REAL DEFAULT 30.0,
			gop_size REAL DEFAULT 2.0,
			is_passthrough_ready BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS stream_slots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slot_number INTEGER NOT NULL UNIQUE,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'idle',
			source_type TEXT NOT NULL DEFAULT 'single',
			video_id TEXT,
			target_platform TEXT NOT NULL,
			rtmp_url TEXT NOT NULL,
			stream_key TEXT NOT NULL,
			mode TEXT NOT NULL DEFAULT 'copy',
			quality TEXT DEFAULT '720p',
			preset TEXT DEFAULT 'ultrafast',
			loop_playback BOOLEAN DEFAULT 1,
			max_duration_minutes INTEGER DEFAULT 0,
			auto_restart BOOLEAN DEFAULT 1,
			enable_overlay BOOLEAN DEFAULT 0,
			overlay_clock_wib BOOLEAN DEFAULT 0,
			overlay_watermark TEXT DEFAULT '',
			overlay_watermark_pos TEXT DEFAULT 'top_right',
			last_pts_sync_ms REAL DEFAULT 0.0,
			last_keyframe_gop_s REAL DEFAULT 2.0,
			last_net_latency_ms INTEGER DEFAULT 0,
			last_snapshot_path TEXT DEFAULT '',
			last_snapshot_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(video_id) REFERENCES videos(id) ON DELETE SET NULL
		);`,

		`CREATE TABLE IF NOT EXISTS playlist_items (
			id TEXT PRIMARY KEY,
			slot_id INTEGER NOT NULL,
			video_id TEXT NOT NULL,
			order_index INTEGER NOT NULL,
			play_status TEXT NOT NULL DEFAULT 'queued',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(slot_id) REFERENCES stream_slots(id) ON DELETE CASCADE,
			FOREIGN KEY(video_id) REFERENCES videos(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS schedules (
			id TEXT PRIMARY KEY,
			slot_id INTEGER NOT NULL,
			cron_expr TEXT NOT NULL,
			duration_minutes INTEGER DEFAULT 120,
			is_enabled BOOLEAN DEFAULT 1,
			overlap_guard_policy TEXT DEFAULT 'yield_priority',
			last_run_at DATETIME,
			next_run_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(slot_id) REFERENCES stream_slots(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS transcode_jobs (
			id TEXT PRIMARY KEY,
			source_video_id TEXT NOT NULL,
			target_file_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'queued',
			progress_percent REAL DEFAULT 0.0,
			error_message TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(source_video_id) REFERENCES videos(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS tunnel_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			provider TEXT NOT NULL DEFAULT 'cloudflare',
			mode TEXT NOT NULL DEFAULT 'quick',
			tunnel_token TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT 0,
			public_url TEXT DEFAULT '',
			last_status TEXT DEFAULT 'offline',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS alert_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			telegram_enabled BOOLEAN DEFAULT 0,
			telegram_bot_token TEXT DEFAULT '',
			telegram_chat_id TEXT DEFAULT '',
			discord_enabled BOOLEAN DEFAULT 0,
			discord_webhook_url TEXT DEFAULT '',
			trigger_on_crash BOOLEAN DEFAULT 1,
			trigger_on_thermal BOOLEAN DEFAULT 1,
			thermal_threshold_c REAL DEFAULT 48.0,
			trigger_on_low_storage BOOLEAN DEFAULT 1,
			low_storage_threshold_gb REAL DEFAULT 5.0,
			last_alert_sent_at DATETIME,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) seedDefaults(ctx context.Context) error {
	// 1. Seed Default Admin User jika belum ada
	var userCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount); err == nil && userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err == nil {
			_, _ = db.ExecContext(ctx,
				`INSERT INTO users (id, username, password_hash, role) VALUES ('usr_admin_default', 'admin', ?, 'admin')`,
				string(hash),
			)
		}
	}

	// 2. Seed Default Stream Slots (Slot 1 & Slot 2) jika belum ada
	var slotCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stream_slots").Scan(&slotCount); err == nil && slotCount == 0 {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO stream_slots 
			(slot_number, name, target_platform, rtmp_url, stream_key, mode, quality, preset, loop_playback)
			VALUES 
			(1, 'Slot 1 (YouTube Main Live)', 'youtube', 'rtmp://a.rtmp.youtube.com/live2', '', 'copy', '720p', 'ultrafast', 1),
			(2, 'Slot 2 (Twitch/FB Backup)', 'twitch', 'rtmp://live.twitch.tv/app', '', 'copy', '720p', 'ultrafast', 1)
		`)
	}

	// 3. Seed Default Tunnel Settings jika belum ada
	_, _ = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO tunnel_settings (id, provider, mode, is_active, public_url, last_status)
		VALUES (1, 'cloudflare', 'quick', 0, '', 'offline')
	`)

	// 4. Seed Default Alert Settings jika belum ada
	_, _ = db.ExecContext(ctx, `
		INSERT OR IGNORE INTO alert_settings 
		(id, telegram_enabled, discord_enabled, trigger_on_crash, trigger_on_thermal, thermal_threshold_c, trigger_on_low_storage, low_storage_threshold_gb)
		VALUES (1, 0, 0, 1, 1, 48.0, 1, 5.0)
	`)

	return nil
}
