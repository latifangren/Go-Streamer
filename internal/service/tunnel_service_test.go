package service

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"go-streamer/internal/domain"
	"go-streamer/internal/tunnel"
)

type mockTunnelRepo struct {
	mu       sync.Mutex
	settings *domain.TunnelSettings
}

func newMockTunnelRepo() *mockTunnelRepo {
	return &mockTunnelRepo{
		settings: &domain.TunnelSettings{
			ID:         1,
			Provider:   domain.TunnelProviderCloudflare,
			Mode:       domain.TunnelModeQuick,
			IsActive:   false,
			PublicURL:  "",
			LastStatus: "offline",
		},
	}
}

func (m *mockTunnelRepo) GetSettings(ctx context.Context) (*domain.TunnelSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := *m.settings
	return &clone, nil
}

func (m *mockTunnelRepo) UpdateSettings(ctx context.Context, settings *domain.TunnelSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := *settings
	m.settings = &clone
	return nil
}

// createMockCloudflared membuat script mock executable untuk testing quick tunnel.
func createMockCloudflared(t *testing.T) string {
	tmpDir := t.TempDir()
	var binPath string
	if runtime.GOOS == "windows" {
		binPath = filepath.Join(tmpDir, "mock_cloudflared.bat")
		script := "@echo off\r\necho INF Your quick Tunnel: https://test-streamer-abc.trycloudflare.com 1>&2\r\nping -n 3 127.0.0.1 >nul\r\n"
		if err := os.WriteFile(binPath, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	} else {
		binPath = filepath.Join(tmpDir, "mock_cloudflared.sh")
		script := "#!/bin/sh\necho 'INF Your quick Tunnel: https://test-streamer-abc.trycloudflare.com' >&2\nsleep 2\n"
		if err := os.WriteFile(binPath, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return binPath
}

func TestTunnelService_GetStatus(t *testing.T) {
	repo := newMockTunnelRepo()
	cf := tunnel.NewCloudflareTunnel("mock_fake")
	ts := tunnel.NewTailscaleDetector()
	svc := NewTunnelService(repo, cf, ts, 8080)

	ctx := context.Background()
	settings, tsIP, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if settings == nil {
		t.Fatal("expected non-nil settings")
	}
	if settings.ID != 1 {
		t.Errorf("expected ID 1, got %d", settings.ID)
	}
	if settings.IsActive {
		t.Errorf("expected initially inactive")
	}
	_ = tsIP // tailscale IP can be empty if daemon is not installed on test host
}

func TestTunnelService_StartAndStopCloudflareQuickTunnel(t *testing.T) {
	mockBin := createMockCloudflared(t)
	repo := newMockTunnelRepo()
	cf := tunnel.NewCloudflareTunnel(mockBin)
	ts := tunnel.NewTailscaleDetector()
	svc := NewTunnelService(repo, cf, ts, 8080)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Start Quick Tunnel
	settings, err := svc.StartTunnel(ctx, domain.TunnelProviderCloudflare, domain.TunnelModeQuick, "")
	if err != nil {
		t.Fatalf("StartTunnel failed: %v", err)
	}

	if !settings.IsActive {
		t.Errorf("expected tunnel to be active")
	}
	if settings.PublicURL != "https://test-streamer-abc.trycloudflare.com" {
		t.Errorf("expected public URL parsed from mock, got %s", settings.PublicURL)
	}
	if settings.LastStatus != "online" {
		t.Errorf("expected status online, got %s", settings.LastStatus)
	}

	// Verifikasi sinkronisasi di GetStatus
	liveSettings, _, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if !liveSettings.IsActive || liveSettings.PublicURL != settings.PublicURL {
		t.Errorf("GetStatus mismatch with active tunnel: %+v", liveSettings)
	}

	// 2. Stop Tunnel
	stoppedSettings, err := svc.StopTunnel(ctx)
	if err != nil {
		t.Fatalf("StopTunnel failed: %v", err)
	}

	if stoppedSettings.IsActive {
		t.Errorf("expected tunnel to be inactive after StopTunnel")
	}
	if stoppedSettings.PublicURL != "" {
		t.Errorf("expected empty publicURL after StopTunnel, got %s", stoppedSettings.PublicURL)
	}
	if stoppedSettings.LastStatus != "offline" {
		t.Errorf("expected offline status, got %s", stoppedSettings.LastStatus)
	}

	// Verifikasi repositori terupdate
	repoSettings, _ := repo.GetSettings(ctx)
	if repoSettings.IsActive || repoSettings.LastStatus != "offline" {
		t.Errorf("repository was not updated to offline: %+v", repoSettings)
	}
}

func TestTunnelService_StartNamedTunnel_Validation(t *testing.T) {
	repo := newMockTunnelRepo()
	cf := tunnel.NewCloudflareTunnel("mock_cf")
	svc := NewTunnelService(repo, cf, nil, 8080)

	ctx := context.Background()

	// Named tunnel tanpa token harus mengembalikan error validasi
	_, err := svc.StartTunnel(ctx, domain.TunnelProviderCloudflare, domain.TunnelModeNamed, "")
	if err == nil {
		t.Fatalf("expected error for named tunnel without token")
	}
	if !strings.Contains(err.Error(), "token cannot be empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestTunnelService_UnsupportedProvider(t *testing.T) {
	repo := newMockTunnelRepo()
	svc := NewTunnelService(repo, nil, nil, 8080)

	ctx := context.Background()
	_, err := svc.StartTunnel(ctx, "unknown-provider", "quick", "")
	if err == nil {
		t.Fatalf("expected error for unsupported provider")
	}
}
