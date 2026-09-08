package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"go-streamer/internal/domain"
	"go-streamer/internal/sysinfo"
)

// TryCloudflareRegex mengekstrak URL public dari output stderr cloudflared Quick Tunnel.
var TryCloudflareRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

// ExtractTryCloudflareURL mencari dan mengembalikan URL trycloudflare dari sebaris teks log.
func ExtractTryCloudflareURL(text string) string {
	return TryCloudflareRegex.FindString(text)
}

// CloudflareTunnel mengelola siklus hidup child process daemon cloudflared.
type CloudflareTunnel struct {
	mu         sync.RWMutex
	binaryPath string
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	isActive   bool
	publicURL  string
	statusText string
	doneChan   chan struct{}
}

// NewCloudflareTunnel membuat instans baru CloudflareTunnel dengan path binary custom atau otomatis.
func NewCloudflareTunnel(customPath string) *CloudflareTunnel {
	bin := sysinfo.FindCloudflared(customPath)
	if bin == "" {
		bin = "cloudflared"
	}
	return &CloudflareTunnel{
		binaryPath: bin,
		statusText: "offline",
	}
}

// Start menjalankan cloudflared sesuai mode yang dipilih (quick atau named).
func (t *CloudflareTunnel) Start(ctx context.Context, mode, token string, localPort int) (string, error) {
	t.mu.Lock()
	if t.isActive {
		url := t.publicURL
		t.mu.Unlock()
		return url, fmt.Errorf("cloudflare tunnel is already running")
	}

	if localPort <= 0 {
		localPort = 8080
	}

	var args []string
	if mode == domain.TunnelModeNamed {
		token = strings.TrimSpace(token)
		if token == "" {
			t.mu.Unlock()
			return "", fmt.Errorf("tunnel token cannot be empty for named tunnel: %w", domain.ErrInvalidInput)
		}
		args = []string{"tunnel", "run", "--token", token}
	} else {
		mode = domain.TunnelModeQuick
		args = []string{"tunnel", "--url", fmt.Sprintf("http://127.0.0.1:%d", localPort), "--no-autoupdate"}
	}

	cmdCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(cmdCtx, t.binaryPath, args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		t.mu.Unlock()
		return "", fmt.Errorf("failed to create stderr pipe for cloudflared: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		t.mu.Unlock()
		return "", fmt.Errorf("failed to start cloudflared process: %w", err)
	}

	doneChan := make(chan struct{})
	t.cmd = cmd
	t.cancel = cancel
	t.doneChan = doneChan
	t.isActive = true
	t.statusText = "starting"
	t.publicURL = ""
	t.mu.Unlock()

	urlChan := make(chan string, 1)

	// Goroutine pembaca stderr untuk mendeteksi URL trycloudflare.com
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		found := false
		for scanner.Scan() {
			line := scanner.Text()
			if !found {
				if match := ExtractTryCloudflareURL(line); match != "" {
					found = true
					t.mu.Lock()
					t.publicURL = match
					t.statusText = "online"
					t.mu.Unlock()
					select {
					case urlChan <- match:
					default:
					}
				}
			}
		}
	}()

	// Goroutine pemantau selesai/crashes child process
	go func() {
		_ = cmd.Wait()
		t.mu.Lock()
		t.isActive = false
		if t.statusText != "offline" {
			t.statusText = "offline"
		}
		t.cmd = nil
		t.mu.Unlock()
		close(doneChan)
	}()

	// Untuk Named Tunnel, tidak ada trycloudflare URL yang dihasilkan di stderr
	if mode == domain.TunnelModeNamed {
		t.mu.Lock()
		t.statusText = "online"
		t.mu.Unlock()
		return "", nil
	}

	// Untuk Quick Tunnel, tunggu penangkapan URL dengan batas waktu (timeout)
	select {
	case pubURL := <-urlChan:
		return pubURL, nil
	case <-time.After(30 * time.Second):
		_ = t.Stop()
		return "", fmt.Errorf("timeout waiting for trycloudflare URL")
	case <-ctx.Done():
		_ = t.Stop()
		return "", ctx.Err()
	}
}

// Stop menghentikan proses daemon cloudflared yang sedang berjalan.
func (t *CloudflareTunnel) Stop() error {
	t.mu.Lock()
	if !t.isActive && t.cmd == nil {
		t.mu.Unlock()
		return nil
	}

	cancel := t.cancel
	cmd := t.cmd
	doneChan := t.doneChan

	t.isActive = false
	t.statusText = "offline"
	t.publicURL = ""
	t.cmd = nil
	t.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	if doneChan != nil {
		select {
		case <-doneChan:
		case <-time.After(3 * time.Second):
		}
	}

	return nil
}

// GetStatus mengembalikan status operasional terkini cloudflared tunnel.
func (t *CloudflareTunnel) GetStatus() (isActive bool, publicURL string, statusText string) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isActive, t.publicURL, t.statusText
}
