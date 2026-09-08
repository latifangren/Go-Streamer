package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port            int
	DataDir         string
	VideoDir        string
	CacheDir        string
	DBPath          string
	JWTSecret       string
	Debug           bool
	FFmpegPath      string
	FFprobePath     string
	CloudflaredPath string
}

func Load() (*Config, error) {
	portFlag := flag.Int("port", 8080, "Port server HTTP (contoh: 8080 atau 80 jika root)")
	dataFlag := flag.String("data", "./data", "Direktori penyimpanan data, db, dan video")
	debugFlag := flag.Bool("debug", false, "Aktifkan mode debug log")
	jwtSecretFlag := flag.String("jwt-secret", "", "Secret key untuk enkripsi JWT")
	ffmpegFlag := flag.String("ffmpeg", "", "Path kustom ke executable ffmpeg")
	ffprobeFlag := flag.String("ffprobe", "", "Path kustom ke executable ffprobe")

	// Parse flags jika belum diparse
	if !flag.Parsed() {
		flag.Parse()
	}

	cfg := &Config{
		Port:        *portFlag,
		DataDir:     *dataFlag,
		Debug:       *debugFlag,
		JWTSecret:   *jwtSecretFlag,
		FFmpegPath:  *ffmpegFlag,
		FFprobePath: *ffprobeFlag,
	}

	// Override dari environment variables jika ada
	if envPort := os.Getenv("STREAMER_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = p
		}
	}
	if envData := os.Getenv("STREAMER_DATA"); envData != "" {
		cfg.DataDir = envData
	}
	if envDebug := os.Getenv("STREAMER_DEBUG"); envDebug == "1" || envDebug == "true" {
		cfg.Debug = true
	}
	if envJWT := os.Getenv("STREAMER_JWT_SECRET"); envJWT != "" {
		cfg.JWTSecret = envJWT
	}
	if envFFmpeg := os.Getenv("STREAMER_FFMPEG"); envFFmpeg != "" {
		cfg.FFmpegPath = envFFmpeg
	}
	if envFFprobe := os.Getenv("STREAMER_FFPROBE"); envFFprobe != "" {
		cfg.FFprobePath = envFFprobe
	}

	// Normalisasi path direktori
	cfg.DataDir = filepath.Clean(cfg.DataDir)
	cfg.VideoDir = filepath.Join(cfg.DataDir, "videos")
	cfg.CacheDir = filepath.Join(cfg.DataDir, "cache")
	cfg.DBPath = filepath.Join(cfg.DataDir, "streamer.db")

	// Buat direktori data jika belum ada
	for _, dir := range []string{cfg.DataDir, cfg.VideoDir, cfg.CacheDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("gagal membuat direktori %s: %w", dir, err)
		}
	}

	// Generate JWT Secret jika kosong dan simpan ke file agar persisten
	if cfg.JWTSecret == "" {
		secretFile := filepath.Join(cfg.DataDir, ".jwt_secret")
		if data, err := os.ReadFile(secretFile); err == nil && len(data) >= 32 {
			cfg.JWTSecret = string(data)
		} else {
			bytes := make([]byte, 32)
			if _, err := rand.Read(bytes); err != nil {
				return nil, fmt.Errorf("gagal menghasilkan JWT secret: %w", err)
			}
			cfg.JWTSecret = hex.EncodeToString(bytes)
			_ = os.WriteFile(secretFile, []byte(cfg.JWTSecret), 0600)
		}
	}

	return cfg, nil
}
