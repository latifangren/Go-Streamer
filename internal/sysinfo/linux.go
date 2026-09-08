//go:build !windows

package sysinfo

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go-streamer/internal/domain"
)

// collectPlatform mengumpulkan metrik sistem spesifik Linux / Android via /proc dan /sys virtual filesystem.
func (c *Collector) collectPlatform() (*domain.SystemMetrics, error) {
	now := time.Now().UTC()

	metrics := &domain.SystemMetrics{
		TemperatureC:       37.0,
		IsThermalThrottled: false,
		Timestamp:          now,
	}

	// 1. CPU Load (/proc/stat)
	if total, idle, err := readCPUStat(); err == nil {
		if c.hasPrevCPU {
			dTotal := total - c.prevCPUTotal
			dIdle := idle - c.prevCPUIdle
			if dTotal > 0 && dTotal >= dIdle {
				cpuPercent := float64(dTotal-dIdle) / float64(dTotal) * 100.0
				metrics.CPUPercent = math.Round(cpuPercent*100) / 100
			}
		}
		c.prevCPUTotal = total
		c.prevCPUIdle = idle
		c.hasPrevCPU = true
	}

	// 2. RAM (/proc/meminfo)
	if memTotal, memFree, memAvailable, err := readMemInfo(); err == nil {
		metrics.RAMTotalMB = memTotal / 1024
		metrics.RAMFreeMB = memFree / 1024
		if memAvailable > 0 && memTotal >= memAvailable {
			metrics.RAMUsedMB = (memTotal - memAvailable) / 1024
		} else if memTotal >= memFree {
			metrics.RAMUsedMB = (memTotal - memFree) / 1024
		}
	}

	// 3. Storage (syscall.Statfs)
	targetDir := c.storageDir
	if targetDir == "" {
		targetDir = "."
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(targetDir, &stat); err != nil {
		_ = syscall.Statfs(".", &stat)
	}
	if stat.Blocks > 0 && stat.Bsize > 0 {
		totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
		freeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
		metrics.DiskTotalGB = math.Round(float64(totalBytes)/(1024*1024*1024)*100) / 100
		metrics.DiskFreeGB = math.Round(float64(freeBytes)/(1024*1024*1024)*100) / 100
	}

	// 4. Thermal (/sys/class/thermal/thermal_zone*/temp)
	if tempC, ok := readThermal(); ok {
		metrics.TemperatureC = tempC
		if tempC > 48.0 {
			metrics.IsThermalThrottled = true
		}
	}

	// 5. Uptime (/proc/uptime)
	if uptime, err := readUptime(); err == nil {
		metrics.UptimeSeconds = uptime
	}

	// 6. Net I/O (/proc/net/dev)
	if rxBytes, txBytes, err := readNetDev(); err == nil {
		if c.hasPrevNet {
			elapsed := now.Sub(c.prevNetTime).Seconds()
			if elapsed > 0 {
				var dRx, dTx uint64
				if rxBytes >= c.prevNetRx {
					dRx = rxBytes - c.prevNetRx
				}
				if txBytes >= c.prevNetTx {
					dTx = txBytes - c.prevNetTx
				}
				metrics.NetRxKBps = math.Round((float64(dRx)/1024.0)/elapsed*100) / 100
				metrics.NetTxKBps = math.Round((float64(dTx)/1024.0)/elapsed*100) / 100
			}
		}
		c.prevNetRx = rxBytes
		c.prevNetTx = txBytes
		c.prevNetTime = now
		c.hasPrevNet = true
	}

	return metrics, nil
}

// readCPUStat membaca baris pertama /proc/stat untuk menghitung total dan idle ticks.
func readCPUStat() (total uint64, idle uint64, err error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			// fields[0] = "cpu", fields[1]=user, fields[2]=nice, fields[3]=system, fields[4]=idle,
			// fields[5]=iowait, fields[6]=irq, fields[7]=softirq, fields[8]=steal
			if len(fields) < 5 {
				break
			}
			var user, nice, sys, idl, iowait, irq, softirq, steal uint64
			user, _ = strconv.ParseUint(fields[1], 10, 64)
			nice, _ = strconv.ParseUint(fields[2], 10, 64)
			sys, _ = strconv.ParseUint(fields[3], 10, 64)
			idl, _ = strconv.ParseUint(fields[4], 10, 64)
			if len(fields) > 5 {
				iowait, _ = strconv.ParseUint(fields[5], 10, 64)
			}
			if len(fields) > 6 {
				irq, _ = strconv.ParseUint(fields[6], 10, 64)
			}
			if len(fields) > 7 {
				softirq, _ = strconv.ParseUint(fields[7], 10, 64)
			}
			if len(fields) > 8 {
				steal, _ = strconv.ParseUint(fields[8], 10, 64)
			}

			idle = idl + iowait
			nonIdle := user + nice + sys + irq + softirq + steal
			total = idle + nonIdle
			return total, idle, nil
		}
	}

	return 0, 0, os.ErrNotExist
}

// readMemInfo membaca /proc/meminfo dan mengembalikan MemTotal, MemFree, dan MemAvailable dalam satuan kB.
func readMemInfo() (memTotal uint64, memFree uint64, memAvailable uint64, err error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	var buffers, cached uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "kB"))
		val, _ := strconv.ParseUint(strings.Fields(valStr)[0], 10, 64)

		switch key {
		case "MemTotal":
			memTotal = val
		case "MemFree":
			memFree = val
		case "MemAvailable":
			memAvailable = val
		case "Buffers":
			buffers = val
		case "Cached":
			cached = val
		}
	}

	if memAvailable == 0 {
		memAvailable = memFree + buffers + cached
	}

	return memTotal, memFree, memAvailable, nil
}

// readThermal mencari file /sys/class/thermal/thermal_zone*/temp dan mengembalikan suhu Celsius tertinggi.
func readThermal() (float64, bool) {
	files, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil || len(files) == 0 {
		return 37.0, false
	}

	var maxTemp float64
	found := false

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		raw := strings.TrimSpace(string(data))
		milliC, err := strconv.ParseFloat(raw, 64)
		if err != nil || milliC <= 0 || milliC > 150000 {
			continue
		}
		tempC := milliC / 1000.0
		if !found || tempC > maxTemp {
			maxTemp = tempC
			found = true
		}
	}

	if !found {
		return 37.0, false
	}

	return math.Round(maxTemp*10) / 10, true
}

// readUptime membaca /proc/uptime dan mengembalikan total detik sistem berjalan.
func readUptime() (uint64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, os.ErrInvalid
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}
	return uint64(sec), nil
}

// readNetDev membaca /proc/net/dev dan menjumlahkan RX dan TX bytes untuk semua interface fisik (kecuali lo).
func readNetDev() (rxBytes uint64, txBytes uint64, err error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		iface := strings.TrimSpace(line[:colonIdx])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(line[colonIdx+1:])
		// fields[0] = rx_bytes, fields[8] = tx_bytes
		if len(fields) >= 9 {
			rx, _ := strconv.ParseUint(fields[0], 10, 64)
			tx, _ := strconv.ParseUint(fields[8], 10, 64)
			rxBytes += rx
			txBytes += tx
		}
	}

	return rxBytes, txBytes, nil
}
