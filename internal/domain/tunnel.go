package domain

import (
	"context"
	"time"
)

const (
	TunnelProviderCloudflare = "cloudflare"
	TunnelProviderTailscale  = "tailscale"

	TunnelModeQuick = "quick"
	TunnelModeNamed = "named"
)

type TunnelSettings struct {
	ID          int       `json:"id"`
	Provider    string    `json:"provider"`
	Mode        string    `json:"mode"`
	TunnelToken string    `json:"-"`
	IsActive    bool      `json:"is_active"`
	PublicURL   string    `json:"public_url"`
	LastStatus  string    `json:"last_status"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TunnelRepository interface {
	GetSettings(ctx context.Context) (*TunnelSettings, error)
	UpdateSettings(ctx context.Context, settings *TunnelSettings) error
}
