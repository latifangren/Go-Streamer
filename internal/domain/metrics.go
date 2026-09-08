package domain

import "time"

type SystemMetrics struct {
	CPUPercent         float64   `json:"cpu_percent"`
	RAMUsedMB          uint64    `json:"ram_used_mb"`
	RAMTotalMB         uint64    `json:"ram_total_mb"`
	RAMFreeMB          uint64    `json:"ram_free_mb"`
	DiskFreeGB         float64   `json:"disk_free_gb"`
	DiskTotalGB        float64   `json:"disk_total_gb"`
	TemperatureC       float64   `json:"temperature_c"`
	IsThermalThrottled bool      `json:"is_thermal_throttled"`
	NetRxKBps          float64   `json:"net_rx_kbps"`
	NetTxKBps          float64   `json:"net_tx_kbps"`
	UptimeSeconds      uint64    `json:"uptime_seconds"`
	Timestamp          time.Time `json:"timestamp"`
}

type StreamTelemetry struct {
	SlotNumber      int       `json:"slot_number"`
	Status          string    `json:"status"`
	Frame           int64     `json:"frame"`
	FPS             float64   `json:"fps"`
	BitrateKbps     float64   `json:"bitrate_kbps"`
	Duration        string    `json:"duration"`
	Speed           string    `json:"speed"`
	DroppedFrames   int       `json:"dropped_frames"`
	PTSSyncMS       float64   `json:"pts_sync_ms"`
	KeyframeCadence string    `json:"keyframe_cadence"`
	NetLatencyMS    int       `json:"net_latency_ms"`
	Timestamp       time.Time `json:"timestamp"`
}

type PlatformInfo struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	IsRoot      bool   `json:"is_root"`
	IsTermux    bool   `json:"is_termux"`
	NumCPU      int    `json:"num_cpu"`
	FFmpegPath  string `json:"ffmpeg_path"`
	FFprobePath string `json:"ffprobe_path"`
	Version     string `json:"version"`
}
