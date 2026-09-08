package runner

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go-streamer/internal/domain"
)

// ExtractSnapshot mengeksekusi FFmpeg untuk mengambil 1 frame thumbnail JPEG dari inputSource.
// Command: ffmpeg -ss 00:00:01 -i <inputSource> -vframes 1 -q:v 5 -vf "scale=480:-1" -y <outputPath>
func ExtractSnapshot(ctx context.Context, ffmpegPath, inputSource, outputPath string) error {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	outDir := filepath.Dir(outputPath)
	if outDir != "" && outDir != "." {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create snapshot directory: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-ss", "00:00:01",
		"-i", inputSource,
		"-vframes", "1",
		"-q:v", "5",
		"-vf", "scale=480:-1",
		"-y",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("ffmpeg snapshot failed: %w: %s", err, stderr.String())
		}
		return fmt.Errorf("ffmpeg snapshot failed: %w", err)
	}

	return nil
}

// EstimateStreamHealth menghitung estimasi kesehatan stream berdasarkan data telemetri:
// - PTS Sync: default 0.00ms (Excellent) jika speed >= 0.95 dan FPS >= 24, naikkan offset jika speed < 0.95.
// - Keyframe Cadence: default "2.00s GOP (STABLE)".
// - Net Latency: estimasi ms berdasarkan bitrate dan dropped frames (35-45ms jika 0 drop, meningkat jika drop > 0).
func EstimateStreamHealth(telemetry *domain.StreamTelemetry) (ptsSyncMS float64, keyframeCadence string, netLatencyMS int) {
	if telemetry == nil {
		return 0.00, "2.00s GOP (STABLE)", 40
	}

	speedVal := parseSpeed(telemetry.Speed)

	// 1. PTS Sync calculation
	if speedVal >= 0.95 && telemetry.FPS >= 24.0 {
		ptsSyncMS = 0.00
	} else {
		if speedVal < 0.95 {
			ptsSyncMS += (0.95 - speedVal) * 100.0
		}
		if telemetry.FPS > 0 && telemetry.FPS < 24.0 {
			ptsSyncMS += (24.0 - telemetry.FPS) * 1.5
		}
		ptsSyncMS = math.Round(ptsSyncMS*100) / 100
	}

	// 2. Keyframe Cadence calculation
	keyframeCadence = "2.00s GOP (STABLE)"
	if speedVal < 0.85 || telemetry.DroppedFrames > 50 {
		keyframeCadence = "2.00s GOP (UNSTABLE)"
	}

	// 3. Net Latency calculation (35-45ms jika 0 drop, meningkat jika drop > 0)
	baseLatency := 40
	if telemetry.BitrateKbps > 6000 {
		baseLatency = 45
	} else if telemetry.BitrateKbps > 4000 {
		baseLatency = 42
	} else if telemetry.BitrateKbps > 2000 {
		baseLatency = 38
	} else if telemetry.BitrateKbps > 0 {
		baseLatency = 35
	}

	if telemetry.DroppedFrames > 0 {
		netLatencyMS = baseLatency + (telemetry.DroppedFrames * 5)
		if netLatencyMS > 5000 {
			netLatencyMS = 5000
		}
	} else {
		netLatencyMS = baseLatency
	}

	// Sinkronkan kembali ke telemetry struct
	telemetry.PTSSyncMS = ptsSyncMS
	telemetry.KeyframeCadence = keyframeCadence
	telemetry.NetLatencyMS = netLatencyMS

	return ptsSyncMS, keyframeCadence, netLatencyMS
}

func parseSpeed(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "x")
	s = strings.TrimSuffix(s, "X")
	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || val <= 0 {
		return 1.0
	}
	return val
}
