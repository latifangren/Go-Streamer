package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

func TestTunnelRepository_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tunnel.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}
	defer db.Close()

	repo := NewTunnelRepository(db.DB)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Ambil setting default yang di-seed (id = 1, provider = cloudflare, mode = quick, is_active = false)
	settings, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.ID != 1 {
		t.Errorf("expected ID 1, got %d", settings.ID)
	}
	if settings.Provider != domain.TunnelProviderCloudflare {
		t.Errorf("expected provider cloudflare, got %s", settings.Provider)
	}
	if settings.IsActive {
		t.Errorf("expected is_active to be false initially")
	}

	// 2. Update status tunnel menjadi aktif dengan public URL
	settings.IsActive = true
	settings.PublicURL = "https://random-subdomain.trycloudflare.com"
	settings.LastStatus = "online"
	settings.Mode = domain.TunnelModeQuick

	if err := repo.UpdateSettings(ctx, settings); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// 3. Verifikasi update tersimpan
	updated, err := repo.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings after update failed: %v", err)
	}
	if !updated.IsActive {
		t.Errorf("expected is_active to be true")
	}
	if updated.PublicURL != "https://random-subdomain.trycloudflare.com" {
		t.Errorf("expected public_url updated, got %s", updated.PublicURL)
	}
	if updated.LastStatus != "online" {
		t.Errorf("expected last_status 'online', got %s", updated.LastStatus)
	}

	// 4. Update dengan nil harus mengembalikan error
	if err := repo.UpdateSettings(ctx, nil); err == nil {
		t.Errorf("expected error when updating with nil settings")
	}
}
