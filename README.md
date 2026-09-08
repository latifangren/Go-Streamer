# GO-STREAMER 🎬⚡

> **Autonomous 24/7 RTMP Live-Streaming Node**  
> Single Self-Contained Binary • Zero-CGO SQLite • Embedded Authentic Neobrutalism Web UI • Mobile & Low-Power Hardware Optimized

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Zero--CGO%20Static-brightgreen.svg)]()
[![Platform](https://img.shields.io/badge/Platform-Android%20%7C%20Linux%20ARM%20%7C%20Linux%20x86%20%7C%20Windows-blue.svg)]()

---

## 📌 Gambaran Umum (Overview)

**Go-Streamer** adalah solusi streaming multimedia mandiri (*self-hosted live streaming server*) yang dirancang khusus untuk berjalan secara stabil 24 jam nonstop pada perangkat bersumber daya terbatas—mulai dari smartphone Android bekas (**Termux non-root** atau **Modul Magisk root**), single-board computer (Raspberry Pi, Orange Pi, Rockchip), **PostmarketOS / Alpine Linux**, hingga server VPS Linux x86_64.

Aplikasi ini menggantikan setup script multimedia lama (seperti script PHP yang rapuh dan boros memori) menjadi **satu file executable tunggal (single binary)** yang telah memaketkan backend API, in-memory scheduler, process supervisor FFmpeg anti-orphan, pengumpul metrik sistem kernel, dan frontend dashboard modern bergaya **Authentic Neobrutalism** (Vite + React 18 + Tailwind CSS) via Go `embed.FS`.

---

## 🚀 Fitur Unggulan (Key Features)

### 1. Dual-Slot RTMP Live Streaming
* **Dua Slot Streaming Simultan**: Jalankan hingga 2 siaran langsung independen ke berbagai platform sekaligus (YouTube Live, Facebook Live, Twitch, TikTok Live via RTMP, atau server RTMP kustom).
* **Mode Passthrough (Stream Copy `-c copy`)**: Menyalurkan stream H.264/AAC langsung ke ingest RTMP tanpa re-encoding. **Beban CPU < 2%** dan perangkat tetap dingin, ideal untuk smartphone dan SBC ARM.
* **Mode Transcoding Fleksibel**: Pilihan resolusi (480p, 720p, 1080p), kontrol bitrate, buffer size, dan preset CPU/GPU jika diperlukan.
* **Dynamic Stream Overlay**: Toggle opsi burn-in jam digital WIB realtime dan watermark teks langsung pada siaran.

### 2. Playlist Sequential Looping (Concat Demuxer)
* **Multi-Video Continuous Playback**: Dukungan pemutaran multi-video berurutan secara berkesinambungan tanpa jeda hitam (*gapless*) dan **zero-CPU overhead** menggunakan FFmpeg Concat Demuxer (`-f concat -safe 0`).
* **Visual Antrean Realtime**: Pantau status urutan video langsung pada kartu slot: `[PLAYING]`, `[NEXT]`, dan `[QUEUED]`.

### 3. Live Ingest Snapshot & Stream Health Monitor
* **Visual Frame Preview (Setiap 30 Detik)**: Tangkapan cuplikan frame JPEG visual langsung dari feed RTMP aktif yang diperbarui secara otomatis.
* **Health Indicators**:
  * **PTS Sync**: Verifikasi sinkronisasi audio/video timestamp FLV container.
  * **Keyframe Cadence**: Pemantauan stabilitas interval keyframe GOP (standar 2.00s).
  * **Net Latency**: Estimasi latensi jaringan transmisi socket RTMP (ms).

### 4. In-App Video Codec Fixer (Offline Transcoder)
* **Verifikasi Kepatuhan Passthrough**: Deteksi otomatis via `ffprobe` apakah video memenuhi standar passthrough (`H.264 High 4.1 + AAC + GOP 2.0s`).
* **Konversi Otomatis Satu Klik**: Background worker berkonkurensi tunggal yang menstandarkan video mentah menjadi passthrough-ready tanpa membebani siaran yang sedang berjalan.

### 5. Smart Scheduler & Overlap Guard
* **In-Memory Cron Engine**: Penjadwalan siaran otomatis internal berbasis `robfig/cron/v3` tanpa ketergantungan cron OS.
* **Auto-Schedule Overlap Guard**: Proteksi otomatis terhadap bentrok waktu siaran dengan kebijakan resolusi yang dapat dipilih:
  * *Yield to Highest Priority* (Prioritas Slot 1)
  * *Terminate Preceding Stream Gracefully*
  * *Deny New Ingest Until Slot Clears*
* **Auto Maintenance**: Rutin pembersihan cache snapshot kadaluarsa (>24 jam) dan vacuum SQLite otomatis.

### 6. Remote Access & Alerting (Solusi CGNAT Seluler)
* **Built-in Cloudflare Tunnel**: Akses dashboard web dari luar jaringan lokal/WiFi secara instan tanpa perlu port-forwarding router. Mendukung **Quick Tunnel** (`*.trycloudflare.com`) maupun **Named Tunnel** dengan token Cloudflare.
* **Tailscale Detection**: Deteksi otomatis antarmuka VPN Tailscale dan menampilkan alamat IP 100.x.y.z pada dashboard.
* **Telegram & Discord Webhook Alerts**: Notifikasi otomatis dengan sistem anti-flood cooldown (5 menit):
  * Peringatan saat stream terputus / crash tiba-tiba.
  * Peringatan suhu prosesor berlebih (*thermal overload* > 48°C).
  * Peringatan kapasitas sisa penyimpanan kritis (< 5 GB).

### 7. Process Supervisor & Anti-Zombie Architecture
* **Process Group Isolation**: Proses FFmpeg diisolasi ke dalam process group terpisah (`Setpgid: true`).
* **Kernel Death Signal (`Pdeathsig: SIGKILL`)**: Memastikan proses FFmpeg mati seketika jika parent Go terhenti, mencegah *orphan/zombie process* yang menguras baterai.
* **Graceful Termination**: Mengirimkan sinyal `SIGINT` ke process group dengan toleransi waktu 5 detik sebelum dipaksa `SIGKILL`.
* **Emergency Kill Switch**: Tombol darurat di header UI untuk mematikan seluruh proses siaran seketika.

### 8. Pure-Go SQLite (Zero-CGO)
* Menggunakan driver `modernc.org/sqlite` dengan mode WAL (*Write-Ahead Logging*).
* Kompilasi silang (*cross-compile*) ke Android ARM64, Linux ARM, atau Linux x86 dapat dilakukan dari sistem operasi mana pun tanpa compiler C (GCC/Clang).

---

## 🏗️ Desain Antarmuka (Authentic Neobrutalism)

Dashboard web Go-Streamer mengadopsi gaya visual **Authentic Neobrutalism**:
* **Border & Shadow Tegas**: Garis batas hitam 2.5px (`border-2 border-black`) dan bayangan keras (`shadow-[3px_3px_0px_#000]`).
* **Palet Warna Pastel Fungsional**:
  * `neoCanvas` (`#daf0fc`): Latar belakang canvas sejuk
  * `neoMint` (`#c8f5d0`): Status aktif, tombol sukses, badge OK
  * `neoBlue` (`#c2e7ff`): Kartu metrik CPU, info
  * `neoLavender` (`#e2daf9`): Banner codec fixer, tab aktif
  * `neoCoral` (`#ffd5cc`): Tombol emergency killswitch, status error
  * `neoYellow` (`#fff0a3`): Indikator warning, highlight waktu WIB
  * `neoPink` (`#ffd4e5`): Overlap guard badge, storage status
* **Tipografi Ganda**: `Plus Jakarta Sans` untuk judul tegas dan `JetBrains Mono` untuk angka metrik teknis/telemetri.

---

## 📱 Panduan Platform & Deployment

### 1. Android Termux (Mode Non-Root)
1. Pasang aplikasi Termux dan install FFmpeg:
   ```bash
   pkg update && pkg upgrade -y
   pkg install ffmpeg -y
   ```
2. Mencegah Android Doze & Phantom Process Killer (Android 12+):
   * Set pengaturan baterai aplikasi Termux ke **Unrestricted**.
   * Jalankan kunci wakelock:
     ```bash
     termux-wake-lock
     ```
   * (Opsional via ADB dari PC satu kali):
     ```bash
     adb shell "/system/bin/device_config set_sync_disabled_for_tests persistent"
     adb shell "/system/bin/device_config put activity_manager max_phantom_processes 2147483647"
     ```
3. Unduh binary `go-streamer-linux-arm64` dan jalankan:
   ```bash
   chmod +x go-streamer-linux-arm64
   ./go-streamer-linux-arm64 -port 8080 -data ./data
   ```

### 2. Android Modul Magisk / KernelSU (Mode Root)
Pada perangkat yang telah di-root, Go-Streamer dapat dipasang sebagai **Modul Magisk**:
* Dijalankan otomatis saat boot sistem Android selesai (`sys.boot_completed=1`).
* Diberikan prioritas anti-LMK: `echo -1000 > /proc/$PID/oom_score_adj`.
* Dapat bind langsung ke port HTTP privileged (`-port 80`).
* File konfigurasi modul tersedia di folder `scripts/magisk/`.

### 3. Linux Server & PostmarketOS (Systemd / OpenRC)
* File unit systemd disediakan di `docs/PLATFORM_GUIDE.md` untuk dijalankan sebagai system service 24/7 dengan restart otomatis.

---

## 🛠️ Kompilasi dari Sumber (Build from Source)

### Prasyarat:
* **Go**: Versi 1.22 atau lebih baru
* **Node.js**: Versi 18+ dan `npm` (untuk kompilasi frontend `web/`)
* **FFmpeg**: Versi 4.4+ terpasang di sistem

### Langkah Kompilasi:
```bash
# 1. Clone repositori
git clone https://github.com/Go-Streamer/Go-Streamer.git
cd Go-Streamer

# 2. Build frontend React SPA
cd web
npm install
npm run build
cd ..

# 3. Kompilasi binary Go (Zero-CGO Standalone)
# Target Host Lokal:
go build -v -o bin/go-streamer ./cmd/server

# Atau Cross-Compile ke Linux ARM64 (Android):
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -o bin/go-streamer-linux-arm64 ./cmd/server
```

Atau cukup gunakan script otomatisasi:
```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

---

## ⚙️ Konfigurasi & Opsi Baris Perintah

| Parameter Flag | Variabel Environment | Nilai Default | Deskripsi |
| :--- | :--- | :--- | :--- |
| `-port` | `STREAMER_PORT` | `8080` | Port HTTP server web |
| `-data` | `STREAMER_DATA` | `./data` | Direktori penyimpanan database, video, dan cache |
| `-debug` | `STREAMER_DEBUG` | `false` | Menampilkan log debug verbose |
| `-jwt-secret` | `STREAMER_JWT_SECRET` | Auto-generated | Kunci enkripsi autentikasi JWT |
| `-ffmpeg` | `STREAMER_FFMPEG` | Auto-detected | Path kustom executable FFmpeg |
| `-ffprobe` | `STREAMER_FFPROBE` | Auto-detected | Path kustom executable FFprobe |

---

## 📡 Ringkasan API Endpoint

### REST Endpoints (`/api/v1/`)
* `GET /api/v1/health` — Status kesehatan aplikasi & database
* `GET /api/v1/system/info` — Informasi arsitektur, OS, root, dan binary
* `GET /api/v1/system/metrics` — Metrik CPU, RAM, Suhu SoC, Storage, Net I/O
* `POST /api/v1/system/killswitch` — Emergency shutdown seluruh proses streaming
* `GET /api/v1/slots` — Daftar slot streaming
* `PUT /api/v1/slots/{id}` — Update konfigurasi slot (overlay, RTMP key, dll.)
* `POST /api/v1/slots/{id}/start` — Mulai siaran slot
* `POST /api/v1/slots/{id}/stop` — Hentikan siaran slot
* `GET /api/v1/slots/{id}/snapshot` — Mengambil file JPEG snapshot visual terbaru
* `GET /api/v1/videos` — Daftar file video di perpustakaan
* `POST /api/v1/videos/upload` — Upload file video baru (multipart)
* `POST /api/v1/codec/fix` — Menjalankan tugas standarisasi video offline
* `GET /api/v1/schedules` — Daftar jadwal cron streaming & status overlap guard
* `GET /api/v1/tunnel/status` — Status Cloudflare Tunnel & IP Tailscale
* `POST /api/v1/tunnel/start` — Mengaktifkan Quick / Named Tunnel
* `POST /api/v1/alerts/test` — Mengirim pesan uji coba ke Telegram & Discord

### WebSocket Endpoint (`/api/v1/ws`)
Mendukung komunikasi duplex realtime:
* **Server Events**: `system_metrics`, `stream_telemetry`, `slot_status_changed`, `job_progress`.
* **Client Actions**: `start_stream`, `stop_stream`, `ping`.

---

## 📄 Lisensi (License)

Proyek ini dilisensikan di bawah [MIT License](LICENSE).
Dokumentasi teknis arsitektur dan rancangan lengkap tersedia di folder [`docs/`](docs/).
