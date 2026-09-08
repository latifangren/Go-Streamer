package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go-streamer/internal/domain"
	"go-streamer/internal/tunnel"
)

// TunnelRepository mendefinisikan interface persistensi konfigurasi tunnel.
type TunnelRepository interface {
	GetSettings(ctx context.Context) (*domain.TunnelSettings, error)
	UpdateSettings(ctx context.Context, settings *domain.TunnelSettings) error
}

// TunnelService mengelola orkestrasi remote access via Cloudflare Tunnel dan Tailscale.
type TunnelService struct {
	repo      TunnelRepository
	cf        *tunnel.CloudflareTunnel
	ts        *tunnel.TailscaleDetector
	localPort int
	mu        sync.Mutex
}

// NewTunnelService membuat instans baru TunnelService.
func NewTunnelService(repo TunnelRepository, cf *tunnel.CloudflareTunnel, ts *tunnel.TailscaleDetector, localPort int) *TunnelService {
	if localPort <= 0 {
		localPort = 8080
	}
	return &TunnelService{
		repo:      repo,
		cf:        cf,
		ts:        ts,
		localPort: localPort,
	}
}

// GetStatus mengambil pengaturan tunnel terkini, menyinkronkan status proses aktif, dan mendeteksi IP Tailscale.
func (s *TunnelService) GetStatus(ctx context.Context) (*domain.TunnelSettings, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		settings = &domain.TunnelSettings{
			ID:         1,
			Provider:   domain.TunnelProviderCloudflare,
			Mode:       domain.TunnelModeQuick,
			LastStatus: "offline",
		}
	}

	tailscaleIP := ""
	if s.ts != nil {
		_, _, ip, _ := s.ts.GetStatus(ctx)
		tailscaleIP = ip
	}

	// Sinkronkan status live dari Cloudflare Tunnel process jika aktif
	if s.cf != nil && settings.Provider == domain.TunnelProviderCloudflare {
		cfActive, cfURL, cfStatus := s.cf.GetStatus()
		settings.IsActive = cfActive
		if cfActive && cfURL != "" {
			settings.PublicURL = cfURL
		}
		if cfStatus != "" {
			settings.LastStatus = cfStatus
		}
	}

	return settings, tailscaleIP, nil
}

// StartTunnel mengaktifkan layanan tunneling sesuai provider dan mode yang dipilih.
func (s *TunnelService) StartTunnel(ctx context.Context, provider, mode, token string) (*domain.TunnelSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = domain.TunnelProviderCloudflare
	}

	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = domain.TunnelModeQuick
	}

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		settings = &domain.TunnelSettings{ID: 1}
	}

	switch provider {
	case domain.TunnelProviderCloudflare:
		if s.cf == nil {
			return nil, fmt.Errorf("cloudflare tunnel controller is not initialized")
		}

		publicURL, err := s.cf.Start(ctx, mode, token, s.localPort)
		if err != nil {
			settings.IsActive = false
			settings.LastStatus = "error"
			_ = s.repo.UpdateSettings(ctx, settings)
			return nil, fmt.Errorf("failed to start cloudflare tunnel: %w", err)
		}

		settings.Provider = domain.TunnelProviderCloudflare
		settings.Mode = mode
		settings.TunnelToken = token
		settings.IsActive = true
		settings.PublicURL = publicURL
		settings.LastStatus = "online"

		if err := s.repo.UpdateSettings(ctx, settings); err != nil {
			return nil, fmt.Errorf("failed to save tunnel settings: %w", err)
		}

		return settings, nil

	case domain.TunnelProviderTailscale:
		if s.ts == nil {
			return nil, fmt.Errorf("tailscale detector is not initialized")
		}

		installed, active, ip, err := s.ts.GetStatus(ctx)
		if err != nil || !installed || !active || ip == "" {
			return nil, fmt.Errorf("tailscale daemon is not running or has no 100.x.y.z IP: %w", domain.ErrTunnelInactive)
		}

		settings.Provider = domain.TunnelProviderTailscale
		settings.IsActive = true
		settings.PublicURL = fmt.Sprintf("http://%s:%d", ip, s.localPort)
		settings.LastStatus = "online"

		if err := s.repo.UpdateSettings(ctx, settings); err != nil {
			return nil, fmt.Errorf("failed to save tunnel settings: %w", err)
		}

		return settings, nil

	default:
		return nil, fmt.Errorf("unsupported tunnel provider %s: %w", provider, domain.ErrInvalidInput)
	}
}

// StopTunnel menghentikan tunnel yang sedang berjalan dan memperbarui status di database ke offline.
func (s *TunnelService) StopTunnel(ctx context.Context) (*domain.TunnelSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		settings = &domain.TunnelSettings{ID: 1}
	}

	if s.cf != nil {
		_ = s.cf.Stop()
	}

	settings.IsActive = false
	settings.PublicURL = ""
	settings.LastStatus = "offline"

	if err := s.repo.UpdateSettings(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to update tunnel settings to offline: %w", err)
	}

	return settings, nil
}
