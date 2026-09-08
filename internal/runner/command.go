package runner

import (
	"fmt"
	"os"
	"strings"

	"go-streamer/internal/domain"
)

// BuildFFmpegArgs membangun parameter baris perintah FFmpeg untuk streaming ke RTMP.
// Mendukung mode passthrough (copy) dan transcode (libx264) serta dynamic overlay (jam WIB & watermark).
func BuildFFmpegArgs(slot *domain.StreamSlot, inputPath string) ([]string, error) {
	if slot == nil {
		return nil, fmt.Errorf("stream slot cannot be nil: %w", domain.ErrInvalidInput)
	}

	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return nil, fmt.Errorf("input path cannot be empty: %w", domain.ErrInvalidInput)
	}

	targetURL, err := buildTargetURL(slot.RTMPURL, slot.StreamKey)
	if err != nil {
		return nil, err
	}

	isTranscode := slot.Mode == domain.StreamModeTranscode || slot.EnableOverlay

	// 1. Mode Passthrough (StreamModeCopy tanpa overlay)
	if !isTranscode && (slot.Mode == domain.StreamModeCopy || slot.Mode == "") {
		args := []string{"-re"}
		if slot.LoopPlayback {
			args = append(args, "-stream_loop", "-1")
		}
		if slot.SourceType == domain.SourceTypePlaylist || strings.HasSuffix(strings.ToLower(inputPath), ".txt") {
			args = append(args, "-f", "concat", "-safe", "0")
		}
		args = append(args,
			"-i", inputPath,
			"-c", "copy",
			"-flvflags", "no_duration_filesize",
			"-f", "flv",
			targetURL,
		)
		return args, nil
	}

	// 2. Mode Transcode (StreamModeTranscode atau EnableOverlay aktif)
	args := []string{"-re"}
	if slot.LoopPlayback {
		args = append(args, "-stream_loop", "-1")
	}
	if slot.SourceType == domain.SourceTypePlaylist || strings.HasSuffix(strings.ToLower(inputPath), ".txt") {
		args = append(args, "-f", "concat", "-safe", "0")
	}
	args = append(args, "-i", inputPath)

	preset := strings.TrimSpace(slot.Preset)
	if preset == "" {
		preset = "ultrafast"
	}

	scaleFilter, bitrate := resolveQuality(slot.Quality)

	// Filtergraph assembly
	var filters []string
	if scaleFilter != "" {
		filters = append(filters, scaleFilter)
	}

	if slot.EnableOverlay {
		if slot.OverlayClockWIB {
			var clockFilter strings.Builder
			clockFilter.WriteString("drawtext=")
			if _, err := os.Stat("/system/fonts/Roboto-Regular.ttf"); err == nil {
				clockFilter.WriteString("fontfile=/system/fonts/Roboto-Regular.ttf:")
			}
			clockFilter.WriteString(`text='%{localtime\:%H\\:%M\\:%S WIB}':x=w-tw-20:y=20:fontsize=24:fontcolor=white@0.9:box=1:boxcolor=black@0.6`)
			filters = append(filters, clockFilter.String())
		}

		if strings.TrimSpace(slot.OverlayWatermark) != "" {
			xPos := "20"
			yPos := "20"
			switch slot.OverlayWatermarkPos {
			case "top_right":
				xPos = "w-tw-20"
				yPos = "20"
				if slot.OverlayClockWIB {
					yPos = "60"
				}
			case "bottom_left":
				xPos = "20"
				yPos = "h-th-20"
			case "bottom_right":
				xPos = "w-tw-20"
				yPos = "h-th-20"
			default:
				xPos = "20"
				yPos = "20"
			}
			watermarkFilter := fmt.Sprintf("drawtext=text='%s':x=%s:y=%s:fontsize=20:fontcolor=yellow@0.9", escapeDrawtext(slot.OverlayWatermark), xPos, yPos)
			filters = append(filters, watermarkFilter)
		}
	}

	args = append(args,
		"-c:v", "libx264",
		"-preset", preset,
		"-tune", "zerolatency",
		"-b:v", bitrate,
	)

	if len(filters) > 0 {
		args = append(args, "-vf", strings.Join(filters, ","))
	}

	args = append(args,
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		"-flvflags", "no_duration_filesize",
		"-f", "flv",
		targetURL,
	)

	return args, nil
}

// buildTargetURL menggabungkan rtmpURL dan streamKey secara aman.
func buildTargetURL(rtmpURL, streamKey string) (string, error) {
	rtmpURL = strings.TrimSpace(rtmpURL)
	streamKey = strings.TrimSpace(streamKey)
	if rtmpURL == "" {
		return "", fmt.Errorf("rtmp url cannot be empty: %w", domain.ErrInvalidInput)
	}
	if streamKey == "" {
		return rtmpURL, nil
	}

	trimmedURL := strings.TrimRight(rtmpURL, "/")
	trimmedKey := strings.TrimLeft(streamKey, "/")

	if strings.HasSuffix(trimmedURL, "/"+trimmedKey) || trimmedURL == trimmedKey {
		return trimmedURL, nil
	}

	return fmt.Sprintf("%s/%s", trimmedURL, trimmedKey), nil
}

// resolveQuality memetakan string resolusi/kualitas ke filter scale dan bitrate video.
func resolveQuality(quality string) (scaleFilter string, bitrate string) {
	bitrate = "2500k"
	q := strings.ToLower(strings.TrimSpace(quality))

	switch q {
	case "1080p":
		scaleFilter = "scale=1920:1080"
		bitrate = "4500k"
	case "720p":
		scaleFilter = "scale=1280:720"
		bitrate = "2500k"
	case "480p":
		scaleFilter = "scale=854:480"
		bitrate = "1200k"
	case "360p":
		scaleFilter = "scale=640:360"
		bitrate = "800k"
	default:
		if q != "" {
			if strings.Contains(q, "x") {
				scaleFilter = fmt.Sprintf("scale=%s", q)
			} else {
				scaleFilter = "scale=1280:720"
			}
		}
	}

	return scaleFilter, bitrate
}

// escapeDrawtext meng-escape karakter khusus pada teks drawtext FFmpeg.
func escapeDrawtext(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", `\'`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, ":", `\:`)
	return s
}
