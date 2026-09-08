package runner

import (
	"sync"
	"testing"
	"time"

	"go-streamer/internal/domain"
)

func TestSupervisor_OnCrashCallback(t *testing.T) {
	crashBinary := createCrashingMockFFmpeg(t)
	cacheDir := t.TempDir()

	sup := NewSupervisor(crashBinary, cacheDir)

	var mu sync.Mutex
	crashedSlotNum := 0
	crashedSlotName := ""
	crashedErrMsg := ""
	crashCalled := false

	sup.SetOnCrashCallback(func(slotNumber int, slotName, errMsg string) {
		mu.Lock()
		defer mu.Unlock()
		crashCalled = true
		crashedSlotNum = slotNumber
		crashedSlotName = slotName
		crashedErrMsg = errMsg
	})

	slot := &domain.StreamSlot{
		ID:          9,
		SlotNumber:  9,
		Name:        "Test Crash Slot",
		RTMPURL:     "rtmp://localhost/live",
		AutoRestart: false,
	}

	if err := sup.StartSlot(slot, "vid.mp4"); err != nil {
		t.Fatalf("failed to start slot: %v", err)
	}

	// Tunggu proses crash dan callback dipanggil
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !crashCalled {
		t.Errorf("expected onCrash callback to be called")
	}
	if crashedSlotNum != 9 {
		t.Errorf("expected slotNumber 9, got %d", crashedSlotNum)
	}
	if crashedSlotName != "Test Crash Slot" {
		t.Errorf("expected slotName 'Test Crash Slot', got %s", crashedSlotName)
	}
	if crashedErrMsg == "" {
		t.Errorf("expected non-empty errMsg")
	}
}
