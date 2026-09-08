package tunnel

import (
	"testing"
)

func TestExtractTryCloudflareURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard cloudflared log line",
			input:    "2026-09-08T12:00:00Z INF |  https://my-quick-tunnel-123.trycloudflare.com  |",
			expected: "https://my-quick-tunnel-123.trycloudflare.com",
		},
		{
			name:     "log line with leading text and formatting",
			input:    "Your quick Tunnel has been created! Visit it at: https://super-cool-domain.trycloudflare.com/welcome",
			expected: "https://super-cool-domain.trycloudflare.com",
		},
		{
			name:     "subdomain with hyphens and numbers",
			input:    "+--------------------------------------------------------------------------------------------+\n|  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |\n|  https://quick-tunnel-99-abc.trycloudflare.com                                            |\n+--------------------------------------------------------------------------------------------+",
			expected: "https://quick-tunnel-99-abc.trycloudflare.com",
		},
		{
			name:     "unrelated log line without trycloudflare URL",
			input:    "2026-09-08T12:00:01Z INF Registered tunnel connection connIndex=0 connection=abc-123",
			expected: "",
		},
		{
			name:     "empty line",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractTryCloudflareURL(tc.input)
			if got != tc.expected {
				t.Errorf("ExtractTryCloudflareURL(%q) = %q, expected %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestCloudflareTunnel_InitialState(t *testing.T) {
	cf := NewCloudflareTunnel("fake_binary")
	isActive, pubURL, status := cf.GetStatus()
	if isActive {
		t.Errorf("expected initially inactive")
	}
	if pubURL != "" {
		t.Errorf("expected empty publicURL initially, got %s", pubURL)
	}
	if status != "offline" {
		t.Errorf("expected offline status initially, got %s", status)
	}

	// Stop when not running should be a safe no-op
	if err := cf.Stop(); err != nil {
		t.Errorf("unexpected error on stopping inactive tunnel: %v", err)
	}
}
