package sysinfo

import (
	"sync"
	"time"

	"go-streamer/internal/domain"
)

// Collector mengumpulkan telemetri perangkat keras dan sistem operasi (CPU, RAM, Storage, Thermal, Jaringan, Uptime).
type Collector struct {
	mu         sync.Mutex
	storageDir string

	// State internal untuk menghitung delta CPU usage
	prevCPUTotal uint64
	prevCPUIdle  uint64
	hasPrevCPU   bool

	// State internal untuk menghitung kecepatan transfer Net I/O
	prevNetRx   uint64
	prevNetTx   uint64
	prevNetTime time.Time
	hasPrevNet  bool

	// State internal untuk Windows GetSystemTimes
	prevWinIdle   uint64
	prevWinKernel uint64
	prevWinUser   uint64
	hasPrevWin    bool
}

// NewCollector membuat instans baru pengumpul telemetri sistem untuk storageDir tertentu.
func NewCollector(storageDir string) *Collector {
	if storageDir == "" {
		storageDir = "."
	}
	return &Collector{
		storageDir: storageDir,
	}
}

// Collect mengumpulkan data metrik sistem terkini secara thread-safe.
func (c *Collector) Collect() (*domain.SystemMetrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.collectPlatform()
}
