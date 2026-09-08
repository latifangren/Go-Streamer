package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseInitializationAndMigration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_streamer.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Verifikasi Pragma WAL mode
	var journalMode string
	err = db.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected journal_mode 'wal', got '%s'", journalMode)
	}

	// 2. Verifikasi 8 tabel ada
	expectedTables := []string{
		"users",
		"videos",
		"stream_slots",
		"playlist_items",
		"schedules",
		"transcode_jobs",
		"tunnel_settings",
		"alert_settings",
	}

	for _, table := range expectedTables {
		var name string
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?;"
		err := db.QueryRowContext(ctx, query, table).Scan(&name)
		if err != nil {
			t.Errorf("Table '%s' does not exist in database: %v", table, err)
		}
	}

	// 3. Verifikasi Seeding Default
	var slotCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stream_slots").Scan(&slotCount); err != nil || slotCount != 2 {
		t.Errorf("Expected 2 default stream slots, got %d (err: %v)", slotCount, err)
	}

	var userCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username='admin'").Scan(&userCount); err != nil || userCount != 1 {
		t.Errorf("Expected default admin user, got %d (err: %v)", userCount, err)
	}

	var tunnelCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tunnel_settings WHERE id=1").Scan(&tunnelCount); err != nil || tunnelCount != 1 {
		t.Errorf("Expected default tunnel settings row, got %d (err: %v)", tunnelCount, err)
	}

	var alertCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM alert_settings WHERE id=1").Scan(&alertCount); err != nil || alertCount != 1 {
		t.Errorf("Expected default alert settings row, got %d (err: %v)", alertCount, err)
	}
}
