package runner

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

func TestParseProbeOutput(t *testing.T) {
	jsonSample := []byte(`{
		"streams": [
			{
				"index": 0,
				"codec_name": "h264",
				"codec_type": "video",
				"width": 1920,
				"height": 1080,
				"r_frame_rate": "30/1",
				"avg_frame_rate": "30/1",
				"duration": "120.500"
			},
			{
				"index": 1,
				"codec_name": "aac",
				"codec_type": "audio",
				"duration": "120.500"
			}
		],
		"format": {
			"filename": "/tmp/test.mp4",
			"duration": "120.500000",
			"size": "10485760"
		}
	}`)

	video, err := parseProbeOutput(jsonSample, "/tmp/test.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if video.VideoCodec != "h264" {
		t.Errorf("expected video_codec h264, got %s", video.VideoCodec)
	}
	if video.AudioCodec != "aac" {
		t.Errorf("expected audio_codec aac, got %s", video.AudioCodec)
	}
	if video.Resolution != "1920x1080" {
		t.Errorf("expected resolution 1920x1080, got %s", video.Resolution)
	}
	if video.FPS != 30.0 {
		t.Errorf("expected fps 30.0, got %f", video.FPS)
	}
	if video.DurationSeconds != 120.5 {
		t.Errorf("expected duration 120.5, got %f", video.DurationSeconds)
	}
	if video.FileSize != 10485760 {
		t.Errorf("expected file_size 10485760, got %d", video.FileSize)
	}
	if video.GOPSize != 2.0 {
		t.Errorf("expected gop_size 2.0, got %f", video.GOPSize)
	}
	if !video.IsPassthroughReady {
		t.Errorf("expected IsPassthroughReady to be true for h264+aac")
	}
}

func TestParseProbeOutput_NonPassthrough(t *testing.T) {
	jsonSample := []byte(`{
		"streams": [
			{
				"index": 0,
				"codec_name": "hevc",
				"codec_type": "video",
				"width": 3840,
				"height": 2160,
				"r_frame_rate": "60000/1001"
			},
			{
				"index": 1,
				"codec_name": "opus",
				"codec_type": "audio"
			}
		],
		"format": {
			"duration": "60.0",
			"size": "5000000"
		}
	}`)

	video, err := parseProbeOutput(jsonSample, "sample.mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if video.IsPassthroughReady {
		t.Errorf("expected IsPassthroughReady to be false for hevc+opus")
	}
	if video.Resolution != "3840x2160" {
		t.Errorf("expected resolution 3840x2160, got %s", video.Resolution)
	}
	if video.FPS != 59.94 {
		t.Errorf("expected fps ~59.94, got %f", video.FPS)
	}
}

func TestParseProbeOutput_Avc1Passthrough(t *testing.T) {
	jsonSample := []byte(`{
		"streams": [
			{
				"codec_name": "avc1",
				"codec_type": "video",
				"width": 1280,
				"height": 720,
				"r_frame_rate": "25/1"
			},
			{
				"codec_name": "AAC",
				"codec_type": "audio"
			}
		],
		"format": {
			"duration": "10.0",
			"size": "1000"
		}
	}`)

	video, err := parseProbeOutput(jsonSample, "avc1_sample.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !video.IsPassthroughReady {
		t.Errorf("expected IsPassthroughReady to be true for avc1+AAC")
	}
}

func TestProbeVideo_FileNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := ProbeVideo(ctx, "ffprobe", "non_existent_file_xyz_123.mp4")
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}
}

func TestParseFPS(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"30/1", 30.0},
		{"60/1", 60.0},
		{"24000/1001", 23.98},
		{"25/1", 25.0},
		{"29.97", 29.97},
		{"0/0", 0},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := parseFPS(tt.input)
		if got != tt.expected {
			t.Errorf("parseFPS(%q) = %v, expected %v", tt.input, got, tt.expected)
		}
	}
}

func TestEstimateStreamHealth(t *testing.T) {
	t.Run("nil telemetry", func(t *testing.T) {
		pts, cadence, latency := EstimateStreamHealth(nil)
		if pts != 0.00 {
			t.Errorf("expected pts 0.00, got %f", pts)
		}
		if cadence != "2.00s GOP (STABLE)" {
			t.Errorf("expected STABLE cadence, got %s", cadence)
		}
		if latency < 35 || latency > 45 {
			t.Errorf("expected latency between 35-45, got %d", latency)
		}
	})

	t.Run("healthy stream", func(t *testing.T) {
		tel := &domain.StreamTelemetry{
			FPS:           30.0,
			Speed:         "1.0x",
			BitrateKbps:   4500,
			DroppedFrames: 0,
			Timestamp:     time.Now(),
		}

		pts, cadence, latency := EstimateStreamHealth(tel)
		if pts != 0.00 {
			t.Errorf("expected pts 0.00, got %f", pts)
		}
		if cadence != "2.00s GOP (STABLE)" {
			t.Errorf("expected STABLE, got %s", cadence)
		}
		if latency < 35 || latency > 45 {
			t.Errorf("expected latency 35-45, got %d", latency)
		}
		if tel.PTSSyncMS != pts || tel.KeyframeCadence != cadence || tel.NetLatencyMS != latency {
			t.Errorf("telemetry fields were not updated properly")
		}
	})

	t.Run("lagging stream with dropped frames", func(t *testing.T) {
		tel := &domain.StreamTelemetry{
			FPS:           20.0,
			Speed:         "0.80x",
			BitrateKbps:   2500,
			DroppedFrames: 60,
			Timestamp:     time.Now(),
		}

		pts, cadence, latency := EstimateStreamHealth(tel)
		if pts <= 0.00 {
			t.Errorf("expected pts > 0.00 for lagging stream, got %f", pts)
		}
		if cadence != "2.00s GOP (UNSTABLE)" {
			t.Errorf("expected UNSTABLE for low speed/high drops, got %s", cadence)
		}
		if latency <= 45 {
			t.Errorf("expected latency > 45 for dropped frames, got %d", latency)
		}
	})
}

func TestExtractSnapshotDirectoryCreation(t *testing.T) {
	// Test that ExtractSnapshot ensures directory exists even if ffmpeg binary fails
	tmpDir, err := os.MkdirTemp("", "snaptest-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub", "dir")
	outPath := filepath.Join(subDir, "snap.jpg")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = ExtractSnapshot(ctx, "fake_nonexistent_ffmpeg", "nonexistent_input.mp4", outPath)

	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Errorf("expected subDir to be created")
	}
}

func TestBuildFFmpegArgs_Passthrough(t *testing.T) {
	slot := &domain.StreamSlot{
		ID:           1,
		SlotNumber:   1,
		Mode:         domain.StreamModeCopy,
		LoopPlayback: false,
		RTMPURL:      "rtmp://a.rtmp.youtube.com/live2",
		StreamKey:    "abcd-1234-wxyz",
	}

	args, err := BuildFFmpegArgs(slot, "/data/videos/video.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"-re",
		"-i", "/data/videos/video.mp4",
		"-c", "copy",
		"-flvflags", "no_duration_filesize",
		"-f", "flv",
		"rtmp://a.rtmp.youtube.com/live2/abcd-1234-wxyz",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("got %v, expected %v", args, expected)
	}
}

func TestBuildFFmpegArgs_Passthrough_Loop(t *testing.T) {
	slot := &domain.StreamSlot{
		ID:           1,
		SlotNumber:   1,
		Mode:         domain.StreamModeCopy,
		LoopPlayback: true,
		RTMPURL:      "rtmp://live.twitch.tv/app/",
		StreamKey:    "/live_user_123",
	}

	args, err := BuildFFmpegArgs(slot, "/data/videos/loop.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"-re",
		"-stream_loop", "-1",
		"-i", "/data/videos/loop.mp4",
		"-c", "copy",
		"-flvflags", "no_duration_filesize",
		"-f", "flv",
		"rtmp://live.twitch.tv/app/live_user_123",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("got %v, expected %v", args, expected)
	}
}

func TestBuildFFmpegArgs_Transcode_QualityAndPreset(t *testing.T) {
	slot := &domain.StreamSlot{
		ID:             1,
		SlotNumber:     1,
		Mode:           domain.StreamModeTranscode,
		Quality:        "720p",
		Preset:         "veryfast",
		RTMPURL:        "rtmp://localhost:1935/live/stream",
		StreamKey:      "",
		LoopPlayback:   true,
	}

	args, err := BuildFFmpegArgs(slot, "/data/videos/input.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"-re",
		"-stream_loop", "-1",
		"-i", "/data/videos/input.mp4",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-tune", "zerolatency",
		"-b:v", "2500k",
		"-vf", "scale=1280:720",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		"-flvflags", "no_duration_filesize",
		"-f", "flv",
		"rtmp://localhost:1935/live/stream",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Errorf("got %v, expected %v", args, expected)
	}
}

func TestBuildFFmpegArgs_Overlay(t *testing.T) {
	slot := &domain.StreamSlot{
		ID:                  1,
		SlotNumber:          1,
		Mode:                domain.StreamModeCopy, // Should switch to transcode
		EnableOverlay:       true,
		OverlayClockWIB:     true,
		OverlayWatermark:    "LIVE BROADCAST",
		OverlayWatermarkPos: "top_left",
		Quality:             "720p",
		RTMPURL:             "rtmp://a.rtmp.youtube.com/live2",
		StreamKey:           "key123",
	}

	args, err := BuildFFmpegArgs(slot, "/data/videos/input.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it switched from copy to libx264
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-c:v libx264") {
		t.Errorf("expected libx264 in transcode with overlay, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-preset ultrafast") {
		t.Errorf("expected preset ultrafast, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-tune zerolatency") {
		t.Errorf("expected tune zerolatency, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-b:v 2500k") {
		t.Errorf("expected bitrate 2500k, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-c:a aac") || !strings.Contains(argsStr, "-b:a 128k") || !strings.Contains(argsStr, "-ar 44100") {
		t.Errorf("expected audio flags, got: %s", argsStr)
	}

	// Verify filtergraph contains scale, clock, and watermark
	var vfVal string
	for i, arg := range args {
		if arg == "-vf" && i+1 < len(args) {
			vfVal = args[i+1]
			break
		}
	}
	if vfVal == "" {
		t.Fatalf("expected -vf flag, not found in %v", args)
	}
	if !strings.Contains(vfVal, "scale=1280:720") {
		t.Errorf("expected scale=1280:720 in -vf, got %s", vfVal)
	}
	if !strings.Contains(vfVal, "localtime") || !strings.Contains(vfVal, "WIB") {
		t.Errorf("expected clock WIB in -vf, got %s", vfVal)
	}
	if !strings.Contains(vfVal, "text='LIVE BROADCAST'") {
		t.Errorf("expected watermark text in -vf, got %s", vfVal)
	}
}

func TestBuildFFmpegArgs_Validation(t *testing.T) {
	_, err := BuildFFmpegArgs(nil, "test.mp4")
	if err == nil {
		t.Errorf("expected error for nil slot")
	}

	slot := &domain.StreamSlot{RTMPURL: "rtmp://localhost/live"}
	_, err = BuildFFmpegArgs(slot, "")
	if err == nil {
		t.Errorf("expected error for empty inputPath")
	}

	slotEmptyURL := &domain.StreamSlot{RTMPURL: ""}
	_, err = BuildFFmpegArgs(slotEmptyURL, "test.mp4")
	if err == nil {
		t.Errorf("expected error for empty RTMPURL")
	}
}

func TestGenerateConcatManifest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concat-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	videos := []string{
		"/data/videos/video1.mp4",
		"/data/videos/video'2.mp4",
	}

	manifestPath, err := GenerateConcatManifest(10, tmpDir, videos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(manifestPath, "concat_slot_10.txt") {
		t.Errorf("unexpected manifest filename: %s", manifestPath)
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest file: %v", err)
	}

	strContent := string(content)
	if !strings.HasPrefix(strContent, "ffconcat version 1.0\n") {
		t.Errorf("manifest must start with header, got:\n%s", strContent)
	}
	if !strings.Contains(strContent, "file '/data/videos/video1.mp4'") {
		t.Errorf("missing video1 in manifest, got:\n%s", strContent)
	}
	if !strings.Contains(strContent, "file '/data/videos/video'\\''2.mp4'") {
		t.Errorf("missing escaped video2 in manifest, got:\n%s", strContent)
	}

	// Empty videos validation
	_, err = GenerateConcatManifest(10, tmpDir, []string{})
	if err == nil {
		t.Errorf("expected error for empty video paths")
	}
}

func TestParseStderrLine(t *testing.T) {
	line := "frame= 724050 fps=30.0 q=-1.0 size=234812kB time=06:42:15 bitrate=2480.2kbits/s speed=1.00x"
	tel := ParseStderrLine(line, 1)
	if tel == nil {
		t.Fatalf("expected parsed telemetry, got nil")
	}

	if tel.SlotNumber != 1 {
		t.Errorf("expected slot 1, got %d", tel.SlotNumber)
	}
	if tel.Status != domain.SlotStatusRunning {
		t.Errorf("expected status %s, got %s", domain.SlotStatusRunning, tel.Status)
	}
	if tel.Frame != 724050 {
		t.Errorf("expected frame 724050, got %d", tel.Frame)
	}
	if tel.FPS != 30.0 {
		t.Errorf("expected fps 30.0, got %f", tel.FPS)
	}
	if tel.BitrateKbps != 2480.2 {
		t.Errorf("expected bitrate 2480.2, got %f", tel.BitrateKbps)
	}
	if tel.Duration != "06:42:15" {
		t.Errorf("expected duration 06:42:15, got %s", tel.Duration)
	}
	if tel.Speed != "1.00x" {
		t.Errorf("expected speed 1.00x, got %s", tel.Speed)
	}
	if tel.Timestamp.IsZero() {
		t.Errorf("expected non-zero timestamp")
	}

	// Test line with dropped frames
	lineWithDrop := "frame=  100 fps=20.0 q=24.0 drop=15 size=   500kB time=00:00:05.00 bitrate= 800.0kbits/s speed=0.85x"
	telDrop := ParseStderrLine(lineWithDrop, 2)
	if telDrop == nil {
		t.Fatalf("expected parsed telemetry for drop line, got nil")
	}
	if telDrop.DroppedFrames != 15 {
		t.Errorf("expected dropped frames 15, got %d", telDrop.DroppedFrames)
	}

	// Test non-progress lines return nil
	nonProgressLines := []string{
		"ffmpeg version 6.0 Copyright (c) 2000-2023 the FFmpeg developers",
		"Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'sample.mp4':",
		"[flv @ 0x55d] Stream #0:0: Video: h264",
		"",
	}
	for _, np := range nonProgressLines {
		if res := ParseStderrLine(np, 1); res != nil {
			t.Errorf("expected nil for non-progress line %q, got %+v", np, res)
		}
	}
}
