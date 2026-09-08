# Implementation Roadmap & Milestones — Go-Streamer

Roadmap ini merinci tahapan pengembangan **Go-Streamer** dari fondasi awal hingga siap produksi pada berbagai target platform (Android Termux, Modul Magisk Root, Linux ARM / PostmarketOS, dan Server Linux x86_64).

---

## 🗺️ Tahapan Eksekusi (Phase Overview)

```
[ Phase 0: Fondasi & Setup ]
         │
         ▼
[ Phase 1: FFmpeg Supervisor & Engine ] ──► (Stream Copy, Concat Playlist, Snapshot 30s, Anti-Zombie)
         │
         ▼
[ Phase 2: Telemetri & Smart Scheduler ] ─► (Procfs Linux, In-Memory Cron, Overlap Guard)
         │
         ▼
[ Phase 3: Media Tools & Codec Fixer ] ───► (Upload, ffprobe, In-App Offline Transcoder)
         │
         ▼
[ Phase 4: Remote Access & Alerts ] ──────► (Cloudflare Quick Tunnel, Discord & Telegram Webhooks)
         │
         ▼
[ Phase 5: Neobrutalist UI & Packaging ] ─► (React SPA, Go Embed, Cross-Compile, Modul Magisk)
```

---

## Phase 0: Fondasi Proyek & Scaffolding (Inisialisasi)

### Tujuan:
Membangun kerangka dasar proyek, struktur direktori terstandar, layer database SQLite tanpa dependensi CGO, dan mekanisme konfigurasi.

### Rincian Tugas:
- [x] **0.1 Inisialisasi Modul Go**:
  - `go mod init go-streamer` (Go 1.22+).
  - Tambahkan dependensi inti: `modernc.org/sqlite`, `github.com/go-chi/chi/v5`, `github.com/gorilla/websocket`, `github.com/robfig/cron/v3`.
- [x] **0.2 Scaffolding Folder Standard Go**:
  - Membentuk folder `cmd/server`, `internal/{api,domain,repository,runner,scheduler,service,sysinfo,tunnel,alert,transcoder}`, `scripts/magisk`, `web`.
- [x] **0.3 Database Engine & DDL Migration**:
  - Inisialisasi `internal/repository/sqlite/db.go` dengan pragma WAL mode (`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`).
  - Implementasi migrasi otomatis 8 tabel: `users`, `videos`, `stream_slots`, `playlist_items`, `schedules`, `transcode_jobs`, `tunnel_settings`, dan `alert_settings`.
- [x] **0.4 Platform & Environment Detector**:
  - Implementasi `internal/sysinfo/platform.go` untuk mendeteksi:
    - Root privilege (`os.Geteuid() == 0`).
    - Environment Termux (`/data/data/com.termux`).
    - Arsitektur CPU (`arm64`, `arm`, `amd64`) dan OS host.
- [x] **0.5 Konfigurasi & CLI Flags**:
  - Flag port `-port` (default `8080`), path data `-data` (default `./data`), dan debug mode `-debug`.

### Kriteria Selesai (Definition of Done):
- Binary dapat di-compile tanpa CGO (`CGO_ENABLED=0 go build ./cmd/server`).
- Database SQLite berhasil terinisiasi dengan seluruh tabel saat pertama kali dijalankan.

---

## Phase 1: FFmpeg Supervisor & Pipeline Engine

### Tujuan:
Membangun engine eksekusi multimedia yang tahan banting, kebal terhadap orphan process, mendukung playlist multi-video (concat demuxer), live snapshot, dan pembaca telemetri realtime.

### Rincian Tugas:
- [x] **1.1 Command Builder & Execution Pipeline**:
  - Implementasi `internal/runner/command.go`:
    - Mode Stream Copy / Passthrough: `-re -stream_loop -1 -i <file> -c copy -flvflags no_duration_filesize -f flv <rtmp_url>`.
    - Mode Transcode Software / Hardware: Parameter libx264 ultrafast, resolusi (480p/720p/1080p), bitrate & buffer size.
    - Dynamic Stream Overlay: Generator filtergraph `-vf drawtext` untuk jam digital WIB realtime dan watermark teks.
- [ ] **1.2 Playlist Sequential Looping (Concat Demuxer)** *(Parsial: engine runner/concat.go selesai, repositori & API tabel playlist_items belum di-wire)*:
  - Implementasi `internal/runner/concat.go`:
    - Generator file manifest `ffconcat version 1.0` dari tabel `playlist_items`.
    - Integrasi argumen `-f concat -safe 0` untuk pemutaran multi-video berkesinambungan tanpa jeda (gapless) dan zero-CPU overhead.
- [x] **1.3 Anti-Zombie & Orphan Process Guard**:
  - Mengonfigurasi `cmd.SysProcAttr`:
    - `Setpgid: true` untuk isolasi process group ID.
    - `Pdeathsig: syscall.SIGKILL` (pada Linux/Android kernel) agar FFmpeg mati otomatis jika parent Go terhenti.
  - Implementasi mekanisme Graceful Kill: Sinyal `SIGINT` ke process group `-cmd.Process.Pid` -> timeout 5 detik -> paksa `SIGKILL`.
- [x] **1.4 Stderr Telemetry Parser**:
  - Implementasi `internal/runner/parser.go` menggunakan regex pre-compiled:
    - Ekstraksi: `frame`, `fps`, `q`, `size`, `time`, `bitrate`, `speed`, dan packet drops.
- [x] **1.5 Live Ingest Snapshot Monitor (Interval 30 Detik)**:
  - Implementasi `internal/runner/snapshot.go`:
    - Ekstraksi berkala 1 frame JPEG (480x270) langsung dari feed aktif setiap 30 detik.
    - Kalkulasi metrik stream health: **PTS Sync** (ms), **Keyframe Cadence** (GOP interval 2.0s), dan **Net Latency** (ms).
    - Penyimpanan file snapshot ke direktori cache lokal untuk disajikan ke UI.
- [x] **1.6 Auto-Restart & State Watcher**:
  - Deteksi crash / koneksi RTMP putus mendadak.
  - Kebijakan auto-restart ber-backoff jika flag `auto_restart = true`.

### Kriteria Selesai (Definition of Done):
- Stream 2 slot dapat berjalan simultan dengan beban CPU mendekati 0% (mode copy) pada file lokal.
- Saat proses Go dihentikan (`Ctrl+C` atau `kill -9`), tidak ada proses FFmpeg yang tertinggal (`orphan`).
- Stderr parser dan snapshot extractor menghasilkan data terstruktur secara periodik.

---

## Phase 2: Telemetri Hardware & Smart Scheduler

### Tujuan:
Membaca metrik sistem langsung dari kernel Linux tanpa library berat, serta mengotomatisasi penayangan stream dengan proteksi tabrakan jadwal.

### Rincian Tugas:
- [x] **2.1 Linux Procfs Metrics Collector**:
  - Implementasi `internal/sysinfo/linux.go`:
    - CPU Load dari `/proc/stat`.
    - RAM Usage (Total, Free, Available) dari `/proc/meminfo`.
    - Storage Usage menggunakan syscall POSIX `Statfs`.
    - SoC Thermal dari `/sys/class/thermal/thermal_zone*/temp`.
  - Fallback stubs di `windows.go` untuk kemudahan local development.
- [x] **2.2 In-Memory Cron Engine**:
  - Integrasi `robfig/cron/v3` di `internal/scheduler/scheduler.go`.
  - Penjadwalan start & stop otomatis per slot berdasarkan ekspresi cron atau durasi tertentu.
- [x] **2.3 Auto-Schedule Overlap Guard**:
  - Implementasi algoritma deteksi bentrok jadwal di `internal/scheduler/overlap_guard.go`.
  - Resolusi kebijakan tabrakan waktu:
    1. *Yield to Highest Priority (Slot 1)*.
    2. *Terminate Preceding Stream Gracefully*.
    3. *Deny New Ingest Until Slot Clears*.
  - Penyediaan endpoint status bentrok untuk menampilkan indikator `OVERLAP GUARD ON` di UI.
- [x] **2.4 Rutinitas Pemeliharaan Otomatis**:
  - Jadwal berkala untuk pembersihan file cache snapshot lama dan log buffer via `CleanExpiredCache`.
  - Perintah vacuum SQLite otomatis (`ScheduleDatabaseVacuum` menjalankan `PRAGMA wal_checkpoint(TRUNCATE)` dan `PRAGMA incremental_vacuum`) untuk menjaga ukuran database tetap ramping.

### Kriteria Selesai (Definition of Done):
- Dashboard menerima broadcast metrik CPU, RAM, Suhu, dan Disk secara akurat setiap 1-2 detik.
- Cron job dapat memulai dan menghentikan slot streaming sesuai jadwal tanpa menggunakan cron OS.

---

## Phase 3: Media Tools & In-App Codec Fixer

### Tujuan:
Menyediakan pengelolaan file video yang aman dan alat konversi otomatis (offline transcoder) untuk memastikan seluruh video memenuhi standar passthrough 24/7.

### Rincian Tugas:
- [x] **3.1 Chunked Resumable Upload**:
  - Implementasi HTTP upload handler yang mampu menerima file video berukuran gigabyte dalam bentuk chunk (`/api/v1/videos/upload-chunk` & `/api/v1/videos/upload-complete`) untuk koneksi jaringan yang tidak stabil.
  - Validasi MIME type dan format container (MP4, MKV, MOV).
  - Sanitasi path file untuk mencegah *path traversal*.
- [x] **3.2 ffprobe Video Inspector**:
  - Implementasi `internal/runner/probe.go`:
    - Ekstraksi: durasi, resolusi, framerate, video codec (`h264`, `hevc`, `vp9`), audio codec (`aac`, `mp3`, `opus`), dan keyframe GOP interval.
    - Evaluasi kepatuhan passthrough: Tandai `is_passthrough_ready = 1` jika video berformat H.264 High Profile 4.1 + AAC + GOP 2 detik konstan.
- [x] **3.3 In-App Video Codec Fixer (Offline Transcoder)**:
  - Implementasi queue worker `internal/transcoder/fixer.go`:
    - Berjalan dengan *single concurrency* (1 antrean) agar tidak membebani CPU saat sistem sedang streaming.
    - Perintah transcode:
      ```bash
      ffmpeg -i input.mp4 -c:v libx264 -profile:v high -level:v 4.1 -preset slow -crf 20 -g 60 -keyint_min 60 -sc_threshold 0 -c:a aac -b:a 128k -ar 44100 fixed_output.mp4
      ```
    - Pelaporan persentase progress konversi ke tabel `transcode_jobs` dan disalurkan via WebSocket ke tab Video & Codec Tools.
- [x] **3.4 Manajemen File Video**:
  - Operasi rename, hapus file video beserta thumbnail cache-nya, dan kalkulasi total kuota penyimpanan.

### Kriteria Selesai (Definition of Done):
- Pengguna dapat mengunggah video dari UI, melihat badge status passthrough, dan menekan tombol *Fix Codec* untuk menstandarkan video yang tidak kompatibel.

---

## Phase 4: Remote Access (CGNAT) & Proactive Webhook Alerts

### Tujuan:
Memungkinkan pengguna mengakses dashboard dari luar jaringan rumah tanpa port-forwarding (solusi CGNAT seluler) serta menerima peringatan dini via Telegram dan Discord.

### Rincian Tugas:
- [x] **4.1 Built-in Cloudflare Tunnel Controller**:
  - Implementasi `internal/tunnel/cloudflare.go`:
    - Deteksi ketersediaan binary `cloudflared` di sistem/PATH.
    - **Quick Tunnel Mode**: Menjalankan `cloudflared tunnel --url http://127.0.0.1:8080`, membaca output stderr, mengekstrak URL publik `https://*.trycloudflare.com`, dan menyimpannya ke memori/database.
    - **Named Tunnel Mode**: Mendukung token Cloudflare permanen untuk domain kustom.
    - Fungsi Start, Stop, dan Status Check tunnel.
- [x] **4.2 Tailscale Integration**:
  - Mendeteksi apakah perangkat terhubung ke Tailscale (`tailscale status --json` atau cek interface jaringan `tailscale0`).
  - Menampilkan IP Tailscale 100.x.y.z pada menu remote dashboard.
- [x] **4.3 Telegram Bot Alert Client**:
  - Implementasi `internal/alert/telegram.go`:
    - Mengirim pesan terformat Markdown via Telegram Bot API (`/sendMessage`).
    - Format pesan memuat: Nama Slot, Status Error, Suhu Perangkat (WIB), dan Kode Exit FFmpeg.
- [x] **4.4 Discord Webhook Alert Client**:
  - Implementasi `internal/alert/discord.go`:
    - Mengirim pesan rich embed Discord dengan status warna (Merah = Fatal/Crash, Kuning = Peringatan Suhu/Memori, Hijau = Recovery).
- [x] **4.5 Alert Dispatcher & Anti-Flood Cooldown**:
  - Implementasi worker queue dengan debounce/cooldown 5 menit untuk jenis peringatan yang sama agar tidak terjadi *spamming*.
  - Trigger otomatis:
    - Stream Crash / Error exit FFmpeg (dihubungkan via `supervisor.SetOnCrashCallback`).
    - Suhu SoC prosesor > 48°C (ambang batas aman perangkat smartphone, dipicu via loop metrik background).
    - Kapasitas penyimpanan video < 5 GB (dipicu via loop metrik background).

### Kriteria Selesai (Definition of Done):
- Menekan tombol *Start Tunnel* di menu remote menghasilkan link publik HTTPS yang langsung dapat dibuka dari internet tanpa setting router.
- Saat stream dimatikan paksa atau suhu melebihi batas, notifikasi alert otomatis masuk ke channel Discord dan bot Telegram.

---

## Phase 5: Authentic Neobrutalism Web UI & Embedded Binary Packaging

### Tujuan:
Membangun frontend SPA interaktif sesuai desain mockup yang telah dibuat (`docs/design/DESIGN.md` & `code.html`), meng-embed frontend ke dalam Go binary (`embed.FS`), dan memaketkan distribusi untuk Android Magisk & Linux.

### Rincian Tugas:
- [x] **5.1 Setup Vite + React + Tailwind CSS**:
  - Konfigurasi TypeScript, Tailwind CSS, dan Lucide Icons di direktori `web/`.
  - Integrasi token tema **Authentic Neobrutalism**:
    - Warna: `neoCanvas`, `neoMint`, `neoBlue`, `neoLavender`, `neoCoral`, `neoYellow`, `neoPink`, `neoDark`.
    - Bayangan: `shadow-[3px_3px_0px_#000]`, `shadow-[5px_5px_0px_#000]`.
    - Font: `Plus Jakarta Sans` dan `JetBrains Mono`.
- [x] **5.2 Implementasi 4 Tab Halaman Dashboard**:
  - **Tab 1: Control Desk & RTMP Telemetry**:
    - 4 Kartu Metrik Wavy: CPU, RAM, Suhu SoC, Storage Buffer.
    - Kartu Dual-Slot dengan Live Ingest Snapshot (refresh 30s), PTS Sync, GOP Cadence, Net Latency, telemetri live, dan Visualizer Antrean Playlist (`[PLAYING]`, `[NEXT]`, `[QUEUED]`).
    - Modal Konfigurasi Slot: Target platform, RTMP URL, Stream Key, toggle Overlay Jam Digital WIB & Watermark.
    - Emergency Kill Switch button di header terhubung ke API backend.
  - **Tab 2: Video Assets & Codec Tools**:
    - Banner Offline Codec Fixer.
    - Daftar video dengan badge kepatuhan passthrough (`PASSTHROUGH READY` vs `NEEDS CONVERSION`).
    - Progress bar konversi offline dan modal upload video terhubung ke API backend.
  - **Tab 3: Smart Scheduler**:
    - Panel Overlap Guard dengan indikator status dan selektor kebijakan resolusi tabrakan waktu.
    - Timeline jadwal dan daftar cron tasks terhubung ke API backend.
  - **Tab 4: Remote Access & Alerts**:
    - Kontrol Cloudflare Quick Tunnel / Token dan Tailscale IP terhubung ke API backend.
    - Form konfigurasi Telegram Bot (Token & Chat ID) dan Discord Webhook URL beserta tombol *Test Alert* terhubung ke API backend.
- [x] **5.3 WebSocket Full-Duplex Client (`useWebSocket`)**:
  - Sinkronisasi realtime dua arah untuk metrik sistem, status slot, telemetri FFmpeg, dan log status.
  - Mendukung dual-path `/api/v1/ws` dan `/ws` dengan normalisasi parsing event (`parsed.event || parsed.type`).
  - Fitur auto-reconnect backoff jika koneksi terputus.
- [x] **5.4 Go Embed Directive (`embed.go`)**:
  - Bundle folder `web/dist` ke binary Go menggunakan `//go:embed web/dist/*`.
  - Menyajikan SPA dengan fallback ke `index.html` untuk client-side routing.
- [x] **5.5 Packaging Modul Magisk (Root Android)**:
  - Menyusun paket modul di `scripts/magisk/`:
    - `module.prop`
    - `service.sh` (menjalankan daemon saat boot selesai, set `oom_score_adj = -1000`, dan set CPU governor).
    - `system/bin/go-streamer`
- [x] **5.6 Multi-Arch Cross Compilation Script**:
  - Script `scripts/build.sh` dan `Makefile` untuk memproduksi binary:
    - `go-streamer-linux-arm64` (Android 64-bit / Pi 4/5)
    - `go-streamer-linux-arm` (Android 32-bit / Pi lama)
    - `go-streamer-linux-amd64` (Server VPS Linux)

### Kriteria Selesai (Definition of Done):
- Binary tunggal `go-streamer` dapat dieksekusi mandiri tanpa aset eksternal.
- Tampilan web UI di browser persis dengan mockup Authentic Neobrutalism yang dirancang pada `docs/design/code.html`.
- Pengujian deployment sukses pada Android Termux, modul Magisk Android, dan Linux Server.
