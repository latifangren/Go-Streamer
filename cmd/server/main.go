package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-streamer/internal/api"
	"go-streamer/internal/config"
	"go-streamer/internal/repository/sqlite"
	"go-streamer/internal/scheduler"
	"go-streamer/internal/sysinfo"
)

func main() {
	printBanner()

	// 1. Load Konfigurasi
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Gagal memuat konfigurasi: %v", err)
	}

	// 2. Deteksi Platform & Environment
	platform := sysinfo.DetectPlatform(cfg.FFmpegPath, cfg.FFprobePath)
	log.Printf("[INFO] Platform Terdeteksi: %s/%s | CPUs: %d", platform.OS, platform.Arch, platform.NumCPU)
	log.Printf("[INFO] Status Hak Akses: Root=%v | Termux=%v", platform.IsRoot, platform.IsTermux)

	if platform.FFmpegPath != "" {
		log.Printf("[INFO] FFmpeg Binary: %s", platform.FFmpegPath)
	} else {
		log.Printf("[WARN] FFmpeg tidak ditemukan di sistem! Mode streaming membutuhkan binary FFmpeg.")
	}

	if platform.FFprobePath != "" {
		log.Printf("[INFO] FFprobe Binary: %s", platform.FFprobePath)
	} else {
		log.Printf("[WARN] FFprobe tidak ditemukan! Inspeksi metadata video akan dinonaktifkan.")
	}

	// 3. Inisialisasi Database SQLite (Pure Go / Zero-CGO)
	log.Printf("[INFO] Menginisialisasi SQLite Database: %s", cfg.DBPath)
	db, err := sqlite.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("[FATAL] Gagal menginisialisasi database SQLite: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("[ERROR] Gagal menutup database: %v", err)
		}
	}()
	log.Printf("[INFO] Migrasi database & seeding default berhasil dijalankan (WAL Mode active).")

	// 4. Inisialisasi HTTP Server & Pemeliharaan Berkala
	server := api.NewServer(cfg, db, platform)
	defer server.Close()

	// Daftarkan callback crash FFmpeg ke alert dispatcher
	server.Supervisor().SetOnCrashCallback(func(slotNumber int, slotName, errMsg string) {
		log.Printf("[ALERT] Stream crash terdeteksi pada Slot %d (%s): %s", slotNumber, slotName, errMsg)
		_ = server.AlertDispatcher().TriggerCrash(context.Background(), slotNumber, slotName, errMsg)
	})

	// Jalankan pemeliharaan database vacuum awal
	if err := scheduler.ScheduleDatabaseVacuum(db.DB); err != nil {
		log.Printf("[WARN] Gagal menjalankan vacuum awal database: %v", err)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Jalankan HTTP Server di Goroutine terpisah
	go func() {
		log.Printf("[READY] Go-Streamer Server berjalan pada http://0.0.0.0:%d", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] HTTP server gagal berjalan: %v", err)
		}
	}()

	// 6. Graceful Shutdown Listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[INFO] Menerima sinyal shutdown (%s), mematikan server secara aman...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("[ERROR] Gagal mematikan HTTP server secara graceful: %v", err)
	}

	log.Printf("[INFO] Go-Streamer berhasil dimatikan secara bersih.")
}

func printBanner() {
	banner := `
===================================================================
   ______ ____         _____ _______ _____  ______          __  __ ______ _____  
  / ____// __ \       / ___//_  __// __ \ / ____/   /\     / / / // ____// __ \ 
 / / __ / / / /______ \__ \  / /  / /_/ // __/     / /    / / / // __/  / /_/ / 
/ /_/ // /_/ //_____/___/ / / /  / _, _// /___    / /___ / /_/ // /___ / _, _/  
\____/ \____/        /____/ /_/  /_/ |_|/_____/   /_____/ \____//_____//_/ |_|   
                                                                                 
         Authentic Neobrutalism RTMP Node (Zero-CGO & Mobile Optimized)          
===================================================================`
	fmt.Println(banner)
}
