package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-streamer/internal/domain"
)

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	Index        int    `json:"index"`
	CodecName    string `json:"codec_name"`
	CodecType    string `json:"codec_type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	RFrameRate   string `json:"r_frame_rate"`
	AvgFrameRate string `json:"avg_frame_rate"`
	Duration     string `json:"duration"`
}

type ffprobeFormat struct {
	Filename string `json:"filename"`
	Duration string `json:"duration"`
	Size     string `json:"size"`
}

// ProbeVideo menginspeksi file video menggunakan ffprobe dan mengembalikan metadata domain.Video.
func ProbeVideo(ctx context.Context, ffprobePath, filePath string) (*domain.Video, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("video file not found: %w", err)
	}

	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}

	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("ffprobe failed: %w: %s", err, stderr.String())
		}
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	return parseProbeOutput(stdout.Bytes(), filePath)
}

// parseProbeOutput mem-parsing JSON ffprobe menjadi struct domain.Video.
func parseProbeOutput(data []byte, filePath string) (*domain.Video, error) {
	var out ffprobeOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe json output: %w", err)
	}

	var (
		videoCodec          string
		audioCodec          string
		width               int
		height              int
		fps                 float64
		videoStreamDuration float64
	)

	for _, stream := range out.Streams {
		switch strings.ToLower(stream.CodecType) {
		case "video":
			if videoCodec == "" {
				videoCodec = strings.ToLower(stream.CodecName)
				width = stream.Width
				height = stream.Height

				fps = parseFPS(stream.RFrameRate)
				if fps <= 0 {
					fps = parseFPS(stream.AvgFrameRate)
				}

				if d, err := strconv.ParseFloat(stream.Duration, 64); err == nil && d > 0 {
					videoStreamDuration = d
				}
			}
		case "audio":
			if audioCodec == "" {
				audioCodec = strings.ToLower(stream.CodecName)
			}
		}
	}

	// Format resolution
	resolution := ""
	if width > 0 && height > 0 {
		resolution = fmt.Sprintf("%dx%d", width, height)
	}

	// Duration in seconds
	var duration float64
	if d, err := strconv.ParseFloat(out.Format.Duration, 64); err == nil && d > 0 {
		duration = math.Round(d*100) / 100
	} else if videoStreamDuration > 0 {
		duration = math.Round(videoStreamDuration*100) / 100
	}

	// File size in bytes
	var fileSize int64
	if s, err := strconv.ParseInt(out.Format.Size, 10, 64); err == nil && s > 0 {
		fileSize = s
	} else if fi, err := os.Stat(filePath); err == nil {
		fileSize = fi.Size()
	}

	// GOP / keyframe cadence default 2.0s
	gopSize := 2.0

	// IsPassthroughReady: video_codec == "h264" (atau "avc1") DAN audio_codec == "aac"
	vCodec := strings.ToLower(videoCodec)
	aCodec := strings.ToLower(audioCodec)
	isPassthroughReady := (vCodec == "h264" || vCodec == "avc1") && aCodec == "aac"

	filename := filepath.Base(filePath)

	return &domain.Video{
		Filename:           filename,
		OriginalName:       filename,
		FilePath:           filePath,
		FileSize:           fileSize,
		DurationSeconds:    duration,
		Resolution:         resolution,
		VideoCodec:         videoCodec,
		AudioCodec:         audioCodec,
		FPS:                fps,
		GOPSize:            gopSize,
		IsPassthroughReady: isPassthroughReady,
		CreatedAt:          time.Now(),
	}, nil
}

// parseFPS mengubah string frame rate pecahan (seperti "30/1" atau "29.97/1") menjadi float64.
func parseFPS(fpsStr string) float64 {
	fpsStr = strings.TrimSpace(fpsStr)
	if fpsStr == "" || fpsStr == "0/0" {
		return 0
	}

	if strings.Contains(fpsStr, "/") {
		parts := strings.Split(fpsStr, "/")
		if len(parts) == 2 {
			num, err1 := strconv.ParseFloat(parts[0], 64)
			den, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil && den > 0 {
				return math.Round((num/den)*100) / 100
			}
		}
	}

	if val, err := strconv.ParseFloat(fpsStr, 64); err == nil && val > 0 {
		return math.Round(val*100) / 100
	}

	return 0
}
