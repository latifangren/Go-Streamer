//go:build windows

package sysinfo

import (
	"math"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"go-streamer/internal/domain"
)

var (
	modKernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modKernel32.NewProc("GlobalMemoryStatusEx")
	procGetSystemTimes       = modKernel32.NewProc("GetSystemTimes")
	procGetTickCount64       = modKernel32.NewProc("GetTickCount64")
)

type memoryStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// collectPlatform mengumpulkan metrik sistem spesifik Windows.
func (c *Collector) collectPlatform() (*domain.SystemMetrics, error) {
	now := time.Now().UTC()

	metrics := &domain.SystemMetrics{
		TemperatureC:       37.0,
		IsThermalThrottled: false,
		Timestamp:          now,
	}

	// 1. RAM (GlobalMemoryStatusEx)
	var mem memoryStatusEx
	mem.cbSize = uint32(unsafe.Sizeof(mem))
	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	if ret != 0 {
		metrics.RAMTotalMB = mem.ullTotalPhys / (1024 * 1024)
		metrics.RAMFreeMB = mem.ullAvailPhys / (1024 * 1024)
		if metrics.RAMTotalMB >= metrics.RAMFreeMB {
			metrics.RAMUsedMB = metrics.RAMTotalMB - metrics.RAMFreeMB
		}
	} else {
		// Fallback menggunakan runtime.ReadMemStats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		metrics.RAMUsedMB = m.Alloc / (1024 * 1024)
		metrics.RAMTotalMB = metrics.RAMUsedMB * 2
		metrics.RAMFreeMB = metrics.RAMTotalMB - metrics.RAMUsedMB
	}

	// 2. Storage (windows.GetDiskFreeSpaceEx)
	targetDir := c.storageDir
	if targetDir == "" {
		targetDir = "."
	}
	absPath, err := filepath.Abs(targetDir)
	if err == nil {
		targetDir = absPath
	}
	dirPtr, err := windows.UTF16PtrFromString(targetDir)
	if err == nil {
		var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
		if err := windows.GetDiskFreeSpaceEx(dirPtr, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes); err == nil {
			metrics.DiskTotalGB = math.Round(float64(totalNumberOfBytes)/(1024*1024*1024)*100) / 100
			metrics.DiskFreeGB = math.Round(float64(freeBytesAvailable)/(1024*1024*1024)*100) / 100
		}
	}

	// 3. CPU (GetSystemTimes)
	var idleTime, kernelTime, userTime windows.Filetime
	retTime, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if retTime != 0 {
		idle := filetimeToUint64(idleTime)
		kernel := filetimeToUint64(kernelTime)
		user := filetimeToUint64(userTime)

		if c.hasPrevWin {
			dKernel := kernel - c.prevWinKernel
			dUser := user - c.prevWinUser
			dIdle := idle - c.prevWinIdle

			// kernelTime di GetSystemTimes sudah mencakup idleTime
			dTotal := dKernel + dUser
			if dTotal > 0 && dTotal >= dIdle {
				cpuPercent := float64(dTotal-dIdle) / float64(dTotal) * 100.0
				metrics.CPUPercent = math.Round(cpuPercent*100) / 100
			}
		}

		c.prevWinIdle = idle
		c.prevWinKernel = kernel
		c.prevWinUser = user
		c.hasPrevWin = true
	}

	// 4. Uptime (GetTickCount64)
	if procGetTickCount64.Find() == nil {
		retTick, _, _ := procGetTickCount64.Call()
		metrics.UptimeSeconds = uint64(retTick / 1000)
	}

	return metrics, nil
}

func filetimeToUint64(ft windows.Filetime) uint64 {
	return (uint64(ft.HighDateTime) << 32) | uint64(ft.LowDateTime)
}
