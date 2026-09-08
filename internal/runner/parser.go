package runner

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-streamer/internal/domain"
)

var (
	progressRegex = regexp.MustCompile(`frame=\s*(\d+)\s+fps=\s*([\d\.]+)\s+q=.*size=\s*(\d+)kB\s+time=([\d:\.]+)\s+bitrate=\s*([\d\.]+)kbits/s\s+speed=\s*([\d\.]+)x`)
	dropRegex     = regexp.MustCompile(`drop=\s*(\d+)`)
)

// ParseStderrLine mengekstrak data telemetri stream dari baris log stderr FFmpeg.
// Mengembalikan *domain.StreamTelemetry jika baris adalah progress line, atau nil jika bukan.
func ParseStderrLine(line string, slotNumber int) *domain.StreamTelemetry {
	matches := progressRegex.FindStringSubmatch(line)
	if len(matches) < 7 {
		return nil
	}

	frame, _ := strconv.ParseInt(matches[1], 10, 64)
	fps, _ := strconv.ParseFloat(matches[2], 64)
	bitrate, _ := strconv.ParseFloat(matches[5], 64)
	duration := strings.TrimSpace(matches[4])
	speed := strings.TrimSpace(matches[6]) + "x"

	droppedFrames := 0
	if dropMatches := dropRegex.FindStringSubmatch(line); len(dropMatches) > 1 {
		if d, err := strconv.Atoi(dropMatches[1]); err == nil {
			droppedFrames = d
		}
	}

	telemetry := &domain.StreamTelemetry{
		SlotNumber:    slotNumber,
		Status:        domain.SlotStatusRunning,
		Frame:         frame,
		FPS:           fps,
		BitrateKbps:   bitrate,
		Duration:      duration,
		Speed:         speed,
		DroppedFrames: droppedFrames,
		Timestamp:     time.Now().UTC(),
	}

	// Hitung dan perbarui estimasi metrik kesehatan stream (PTS sync, keyframe cadence, network latency)
	EstimateStreamHealth(telemetry)

	return telemetry
}
