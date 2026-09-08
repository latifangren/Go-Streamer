package sysinfo

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"go-streamer/internal/domain"
)

const AppVersion = "v1.0.0"

// DetectPlatform mengumpulkan informasi environment perangkat, hak akses, dan ketersediaan binary eksternal.
func DetectPlatform(customFFmpeg, customFFprobe string) *domain.PlatformInfo {
	info := &domain.PlatformInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		NumCPU:   runtime.NumCPU(),
		IsRoot:   IsRunningAsRoot(),
		IsTermux: detectTermux(),
		Version:  AppVersion,
	}

	info.FFmpegPath = findExecutable(customFFmpeg, "ffmpeg", getFFmpegCandidatePaths(info.IsTermux))
	info.FFprobePath = findExecutable(customFFprobe, "ffprobe", getFFprobeCandidatePaths(info.IsTermux))

	return info
}

func detectTermux() bool {
	if prefix := os.Getenv("PREFIX"); strings.Contains(prefix, "com.termux") {
		return true
	}
	if _, err := os.Stat("/data/data/com.termux"); err == nil {
		return true
	}
	return false
}

func findExecutable(customPath, name string, candidates []string) string {
	if customPath != "" {
		if path, err := exec.LookPath(customPath); err == nil {
			return path
		}
		if _, err := os.Stat(customPath); err == nil {
			return customPath
		}
	}

	// Cek PATH standar
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	// Cek kandidat path absolut
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}

func getFFmpegCandidatePaths(isTermux bool) []string {
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			`C:\ffmpeg\bin\ffmpeg.exe`,
			`ffmpeg.exe`,
		)
	} else {
		if isTermux {
			candidates = append(candidates,
				"/data/data/com.termux/files/usr/bin/ffmpeg",
			)
		}
		// Android root Magisk, Linux server, PostmarketOS
		candidates = append(candidates,
			"/data/adb/modules/go_streamer/system/bin/ffmpeg",
			"/system/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
			"/usr/bin/ffmpeg",
		)
	}
	return candidates
}

func getFFprobeCandidatePaths(isTermux bool) []string {
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			`C:\ffmpeg\bin\ffprobe.exe`,
			`ffprobe.exe`,
		)
	} else {
		if isTermux {
			candidates = append(candidates,
				"/data/data/com.termux/files/usr/bin/ffprobe",
			)
		}
		candidates = append(candidates,
			"/data/adb/modules/go_streamer/system/bin/ffprobe",
			"/system/bin/ffprobe",
			"/usr/local/bin/ffprobe",
			"/usr/bin/ffprobe",
		)
	}
	return candidates
}

// FindCloudflared mencari binary cloudflared untuk tunneling
func FindCloudflared(customPath string) string {
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = append(candidates, `C:\cloudflared\cloudflared.exe`, `cloudflared.exe`)
	} else {
		candidates = append(candidates,
			"/data/data/com.termux/files/usr/bin/cloudflared",
			"/data/adb/modules/go_streamer/system/bin/cloudflared",
			"/usr/local/bin/cloudflared",
			"/usr/bin/cloudflared",
		)
	}
	return findExecutable(customPath, "cloudflared", candidates)
}
