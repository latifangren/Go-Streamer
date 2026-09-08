package sysinfo

import (
	"sync"
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	tempDir := t.TempDir()
	c := NewCollector(tempDir)
	if c == nil {
		t.Fatal("expected non-nil Collector")
	}
	if c.storageDir != tempDir {
		t.Errorf("expected storageDir %s, got %s", tempDir, c.storageDir)
	}

	cDefault := NewCollector("")
	if cDefault.storageDir != "." {
		t.Errorf("expected default storageDir '.', got %s", cDefault.storageDir)
	}
}

func TestCollector_Collect(t *testing.T) {
	tempDir := t.TempDir()
	c := NewCollector(tempDir)

	metrics, err := c.Collect()
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if metrics == nil {
		t.Fatal("expected non-nil SystemMetrics")
	}

	// RAM verification
	if metrics.RAMTotalMB == 0 {
		t.Errorf("expected RAMTotalMB > 0, got %d", metrics.RAMTotalMB)
	}
	if metrics.RAMUsedMB > metrics.RAMTotalMB {
		t.Errorf("RAMUsedMB (%d) cannot exceed RAMTotalMB (%d)", metrics.RAMUsedMB, metrics.RAMTotalMB)
	}

	// Storage verification
	if metrics.DiskTotalGB <= 0 {
		t.Errorf("expected DiskTotalGB > 0, got %f", metrics.DiskTotalGB)
	}
	if metrics.DiskFreeGB < 0 || metrics.DiskFreeGB > metrics.DiskTotalGB {
		t.Errorf("invalid DiskFreeGB (%f) for DiskTotalGB (%f)", metrics.DiskFreeGB, metrics.DiskTotalGB)
	}

	// Timestamp verification
	if metrics.Timestamp.IsZero() {
		t.Errorf("expected non-zero Timestamp")
	}
	if time.Since(metrics.Timestamp) > 5*time.Second {
		t.Errorf("timestamp is too old: %v", metrics.Timestamp)
	}

	// Thermal verification
	if metrics.TemperatureC <= 0 {
		t.Errorf("expected positive TemperatureC, got %f", metrics.TemperatureC)
	}

	// CPUPercent range verification (0.0 - 100.0)
	if metrics.CPUPercent < 0 || metrics.CPUPercent > 100.0 {
		t.Errorf("CPUPercent out of range [0, 100]: %f", metrics.CPUPercent)
	}
}

func TestCollector_Concurrency(t *testing.T) {
	tempDir := t.TempDir()
	c := NewCollector(tempDir)

	var wg sync.WaitGroup
	workers := 10
	errChan := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := c.Collect()
			if err != nil {
				errChan <- err
				return
			}
			if m == nil || m.RAMTotalMB == 0 {
				t.Errorf("invalid metrics received in worker")
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent Collect error: %v", err)
	}
}
