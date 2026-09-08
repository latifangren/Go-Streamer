package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

func createMockFFmpeg(t *testing.T) string {
	tmpDir := t.TempDir()
	var scriptPath string
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(tmpDir, "mock_ffmpeg.bat")
		content := "@echo off\r\necho frame= 100 fps=30.0 q=-1.0 size= 100kB time=00:00:03.33 bitrate= 2500.0kbits/s speed=1.00x 1>&2\r\nping -n 3 127.0.0.1 >nul\r\n"
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	} else {
		scriptPath = filepath.Join(tmpDir, "mock_ffmpeg.sh")
		content := "#!/bin/sh\necho 'frame= 100 fps=30.0 q=-1.0 size= 100kB time=00:00:03.33 bitrate= 2500.0kbits/s speed=1.00x' >&2\nsleep 2\n"
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return scriptPath
}

func createCrashingMockFFmpeg(t *testing.T) string {
	tmpDir := t.TempDir()
	var scriptPath string
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(tmpDir, "mock_crash.bat")
		content := "@echo off\r\necho ffmpeg error: connection refused 1>&2\r\nexit /b 1\r\n"
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	} else {
		scriptPath = filepath.Join(tmpDir, "mock_crash.sh")
		content := "#!/bin/sh\necho 'ffmpeg error: connection refused' >&2\nexit 1\n"
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return scriptPath
}

func TestSupervisor_InitAndEmptyStatus(t *testing.T) {
	cacheDir := t.TempDir()
	sup := NewSupervisor("", cacheDir)

	if sup.ffmpegPath != "ffmpeg" {
		t.Errorf("expected default ffmpeg, got %s", sup.ffmpegPath)
	}
	if sup.cacheDir != cacheDir {
		t.Errorf("expected cacheDir %s, got %s", cacheDir, sup.cacheDir)
	}

	status, tel, running := sup.GetSlotStatus(99)
	if status != domain.SlotStatusIdle {
		t.Errorf("expected idle status for nonexistent slot, got %s", status)
	}
	if tel != nil {
		t.Errorf("expected nil telemetry, got %+v", tel)
	}
	if running {
		t.Errorf("expected running to be false")
	}
}

func TestSupervisor_Validation(t *testing.T) {
	cacheDir := t.TempDir()
	sup := NewSupervisor("mock", cacheDir)

	if err := sup.StartSlot(nil, "input.mp4"); err == nil {
		t.Errorf("expected error when slot is nil")
	}

	slot := &domain.StreamSlot{SlotNumber: 1, RTMPURL: "rtmp://localhost/live"}
	if err := sup.StartSlot(slot, ""); err == nil {
		t.Errorf("expected error when inputSource is empty")
	}

	if err := sup.StopSlot(1); err != domain.ErrSlotNotRunning {
		t.Errorf("expected ErrSlotNotRunning, got %v", err)
	}
}

func TestSupervisor_LifecycleAndTelemetry(t *testing.T) {
	mockBinary := createMockFFmpeg(t)
	cacheDir := t.TempDir()

	sup := NewSupervisor(mockBinary, cacheDir)

	slot := &domain.StreamSlot{
		ID:           1,
		SlotNumber:   1,
		Mode:         domain.StreamModeCopy,
		LoopPlayback: false,
		RTMPURL:      "rtmp://localhost:1935/live",
		StreamKey:    "testkey",
		AutoRestart:  false,
	}

	err := sup.StartSlot(slot, "/data/videos/sample.mp4")
	if err != nil {
		t.Fatalf("failed to start slot: %v", err)
	}

	// Slot should now be running
	status, _, running := sup.GetSlotStatus(1)
	if !running || status != domain.SlotStatusRunning {
		t.Errorf("expected slot to be running, got status: %s, running: %v", status, running)
	}

	// Starting again should return ErrSlotBusy
	if err := sup.StartSlot(slot, "/data/videos/sample.mp4"); err != domain.ErrSlotBusy {
		t.Errorf("expected ErrSlotBusy, got: %v", err)
	}

	// Receive telemetry from channel
	select {
	case tel := <-sup.TelemetryChan():
		if tel == nil {
			t.Fatal("expected non-nil telemetry")
		}
		if tel.SlotNumber != 1 {
			t.Errorf("expected slot number 1, got %d", tel.SlotNumber)
		}
		if tel.BitrateKbps != 2500.0 {
			t.Errorf("expected bitrate 2500.0, got %f", tel.BitrateKbps)
		}
		if tel.FPS != 30.0 {
			t.Errorf("expected fps 30.0, got %f", tel.FPS)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for telemetry")
	}

	// Check updated status and telemetry via GetSlotStatus
	status, lastTel, running := sup.GetSlotStatus(1)
	if !running || lastTel == nil {
		t.Errorf("expected running slot with telemetry, status=%s, lastTel=%v", status, lastTel)
	}

	// Graceful stop
	err = sup.StopSlot(1)
	if err != nil {
		t.Fatalf("failed to stop slot: %v", err)
	}

	status, _, running = sup.GetSlotStatus(1)
	if running || status != domain.SlotStatusIdle {
		t.Errorf("expected idle status after stop, got status: %s, running: %v", status, running)
	}

	// Stopping again should return ErrSlotNotRunning
	if err := sup.StopSlot(1); err != domain.ErrSlotNotRunning {
		t.Errorf("expected ErrSlotNotRunning on second stop, got: %v", err)
	}
}

func TestSupervisor_PlaylistAutoConcat(t *testing.T) {
	mockBinary := createMockFFmpeg(t)
	cacheDir := t.TempDir()

	sup := NewSupervisor(mockBinary, cacheDir)

	slot := &domain.StreamSlot{
		ID:           42,
		SlotNumber:   2,
		SourceType:   domain.SourceTypePlaylist,
		Mode:         domain.StreamModeCopy,
		LoopPlayback: true,
		RTMPURL:      "rtmp://localhost:1935/live/playlist",
		AutoRestart:  false,
	}

	// Pass comma-separated video paths
	multiVideos := "/path/to/video1.mp4,/path/to/video2.mp4"
	err := sup.StartSlot(slot, multiVideos)
	if err != nil {
		t.Fatalf("failed to start slot with playlist: %v", err)
	}
	defer sup.StopSlot(2)

	// Verify concat manifest was created
	manifestPath := filepath.Join(cacheDir, "concat_slot_42.txt")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("expected concat manifest at %s, error: %v", manifestPath, err)
	}

	if !strings.Contains(string(content), "ffconcat version 1.0") {
		t.Errorf("manifest missing ffconcat header: %s", string(content))
	}
	if !strings.Contains(string(content), "video1.mp4") || !strings.Contains(string(content), "video2.mp4") {
		t.Errorf("manifest missing video entries: %s", string(content))
	}
}

func TestSupervisor_StopAll(t *testing.T) {
	mockBinary := createMockFFmpeg(t)
	cacheDir := t.TempDir()

	sup := NewSupervisor(mockBinary, cacheDir)

	slot1 := &domain.StreamSlot{
		ID:          1,
		SlotNumber:  1,
		RTMPURL:     "rtmp://localhost/live",
		AutoRestart: false,
	}
	slot2 := &domain.StreamSlot{
		ID:          2,
		SlotNumber:  2,
		RTMPURL:     "rtmp://localhost/live",
		AutoRestart: false,
	}

	if err := sup.StartSlot(slot1, "vid1.mp4"); err != nil {
		t.Fatalf("failed to start slot 1: %v", err)
	}
	if err := sup.StartSlot(slot2, "vid2.mp4"); err != nil {
		t.Fatalf("failed to start slot 2: %v", err)
	}

	_, _, running1 := sup.GetSlotStatus(1)
	_, _, running2 := sup.GetSlotStatus(2)
	if !running1 || !running2 {
		t.Fatalf("expected both slots running")
	}

	sup.StopAll()

	_, _, running1 = sup.GetSlotStatus(1)
	_, _, running2 = sup.GetSlotStatus(2)
	if running1 || running2 {
		t.Errorf("expected both slots stopped after StopAll, got running1=%v, running2=%v", running1, running2)
	}
}

func TestSupervisor_CrashWithoutAutoRestart(t *testing.T) {
	crashBinary := createCrashingMockFFmpeg(t)
	cacheDir := t.TempDir()

	sup := NewSupervisor(crashBinary, cacheDir)

	slot := &domain.StreamSlot{
		ID:          3,
		SlotNumber:  3,
		RTMPURL:     "rtmp://localhost/live",
		AutoRestart: false,
	}

	if err := sup.StartSlot(slot, "vid.mp4"); err != nil {
		t.Fatalf("failed to start slot: %v", err)
	}

	// Give it a moment to crash and update status to error
	time.Sleep(500 * time.Millisecond)

	status, _, running := sup.GetSlotStatus(3)
	if running {
		t.Errorf("expected slot not running after crash")
	}
	if status != domain.SlotStatusError {
		t.Errorf("expected status %s after crash, got %s", domain.SlotStatusError, status)
	}
}
