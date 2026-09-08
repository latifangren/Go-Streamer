# Product Requirements Document (PRD) — Go-Streamer

## 1. Executive Summary & Problem Statement

**Go-Streamer** adalah sistem streaming multimedia mandiri (*self-hosted 24/7 RTMP live-streamer*) yang dirancang khusus untuk berjalan dengan andal pada perangkat bersumber daya terbatas (Android non-root via Termux, Android root via Modul Magisk, PostmarketOS, Linux ARM SBC/Raspberry Pi, dan Server Linux x86_64).

### Masalah pada Sistem Legacy (PHP)
1. **Footprint Berat & Dependensi Eksternal**: Sistem legacy berbasis PHP membutuhkan stack runtime tambahan (Apache/Nginx atau PHP built-in web server), ekstensi C (`posix`), serta database file JSON mentah yang rawan korupsi data saat terjadi crash atau write race condition.
2. **Kerapuhan Background Process & Orphan Process**: Pengelolaan proses FFmpeg melalui `posix_kill` dan shell spawn PHP rentan meninggalkan *orphan process* (FFmpeg tetap berjalan tak terkontrol saat PHP mati, menguras CPU & baterai).
3. **Android Low Memory Killer (LMK) & Phantom Process Killer**: Pada Android 12+, daemon PHP di Termux sering dihentikan paksa tanpa toleransi saat memakan CPU/RAM di background.
4. **Ketergantungan Cron OS**: Penjadwalan bergantung pada crond sistem yang sering tidak aktif atau dibatasi pada environment Android/Termux.
5. **Keterbatasan Jaringan Mobile (CGNAT)**: Mengakses dashboard web dari luar jaringan rumah pada perangkat smartphone Android sangat sulit karena alamat IP seluler berada di balik Carrier-Grade NAT (CGNAT) tanpa IP publik.

### Visi Solusi Go-Streamer
Menggantikan seluruh sistem lama dengan **Single Self-Contained Binary** berbasis bahasa **Go (Golang)**:
- **Zero External Web Server**: Menggunakan server HTTP bawaan Go.
- **Embedded Frontend**: Dashboard SPA (Vite + React + Tailwind CSS) bergaya **Authentic Neobrutalism** di-bundle langsung ke dalam binary executable menggunakan `//go:embed`.
- **Zero-CGO SQLite**: Menggunakan `modernc.org/sqlite` untuk persistensi data ACID tanpa ketergantungan toolchain compiler C eksternal.
- **Robust Process Supervisor**: Membungkus proses FFmpeg dengan Go context, channel, process group, dan sinyal POSIX untuk mencegah orphan process dan membaca log progress secara realtime.
- **In-Memory Scheduling & Overlap Guard**: Engine scheduler mandiri berbasis goroutine internal (`robfig/cron/v3`) dengan proteksi tabrakan jadwal cerdas.
- **Native OS Metrics**: Membaca metrik CPU/RAM/Storage/Network langsung dari kernel Linux (`/proc/stat`, `/proc/meminfo`, `syscall.Statfs`) tanpa dependensi library eksternal berlebih.
- **Built-in CGNAT Bypassing**: Integrasi langsung Cloudflare Quick Tunnel (`cloudflared`) dan status Tailscale untuk akses jarak jauh dari mana saja tanpa port-forwarding.
- **Proactive Remote Webhooks**: Peringatan otomatis ke Telegram & Discord saat stream terganggu, panas ekstrem (>48°C), atau kapasitas penyimpanan kritis.

---

## 2. Target Platform & Lingkungan Eksekusi

| Platform | Mode | Akses Hak | Karakteristik & Limitasi |
| :--- | :--- | :--- | :--- |
| **Android (Termux)** | Non-Root | Userland Sandbox | Port > 1024 (misal: 8080), rentan Phantom Process Killer & LMK jika layar mati, butuh wakelock, encoder stream copy sangat dianjurkan. |
| **Android (Magisk)** | Root | UID 0 (Root System) | Dapat bind port 80/443, kebal LMK (`oom_score_adj = -1000`), autostart saat boot via `service.sh`, akses langsung `/dev/` dan thermal throttle governor. |
| **Linux ARM (PostmarketOS / Pi)** | Root / Sudo | Systemd / OpenRC | Hemat daya, stabil 24/7, encoder software atau hardware (v4l2m2m / VA-API jika didukung). |
| **Linux Server (x86_64 / Cloud)** | Root / Docker | Systemd / Container | Komputasi tinggi, mendukung software encoding (libx264) multi-slot atau NVENC/QuickSync. |

---

## 3. Fitur Utama & Spesifikasi Fungsional

### 3.1. Multi-Slot Streaming & Content Management
- Mendukung minimal 2 slot streaming simultan yang berjalan independen.
- **Dua Mode Sumber Media**:
  1. *Single Video File Looping*: Memutar satu video berulang tanpa henti (`-stream_loop -1`).
  2. *Playlist Sequential Looping Mode (Concat Demuxer)*: Memutar antrean multi-video berurutan secara berkesinambungan tanpa jeda hitam (gapless) dan tanpa re-encoding menggunakan FFmpeg Concat Demuxer (`-f concat -safe 0`).
  3. *Playback Queue Visualizer*: Visualisasi status antrean video secara realtime di kartu slot aktif:
     - `[PLAYING]`: Video yang sedang dialirkan ke RTMP.
     - `[NEXT]`: Video berikutnya dalam daftar putar.
     - `[QUEUED]`: Video yang menunggu giliran putar.
- **Dynamic Stream Overlay**:
  - Opsi toggle burn-in teks watermark (misal: "LIVE BROADCAST").
  - Opsi toggle jam digital WIB realtime terintegrasi via filter FFmpeg (`drawtext`).
- **Target RTMP Multivariat**: YouTube Live, Facebook Live, Twitch, TikTok Live (via RTMP key), atau Custom RTMP Ingest.
- **Encoding Strategy**:
  - *Stream Copy (Passthrough)*: Beban CPU < 2%, format H.264 + AAC langsung dialirkan ke ingest.
  - *Software Transcoding (libx264)*: Kontrol resolusi (480p, 720p, 1080p), preset, framerate, dan bitrate.
  - *Hardware Acceleration*: Dukungan otomatis untuk `h264_mediacodec`, `h264_v4l2m2m`, `h264_nvenc`, dan `h264_vaapi`.

### 3.2. Lifecycle & Process Supervisor
- Kontrol penuh: Start, Stop, Restart, dan Status Check per slot.
- **Auto-Restart on Crash**: Rekoneksi otomatis jika koneksi RTMP terputus atau FFmpeg crash mendadak (dengan batasan retry count dan exponential backoff).
- **Anti-Zombie / Orphan Prevention**:
  - Memisahkan FFmpeg ke dalam Process Group independen (`Setpgid: true`).
  - Mengonfigurasi `Pdeathsig: SIGKILL` pada Linux/Android (jika parent Go mati, kernel otomatis mematikan FFmpeg).
  - Graceful stop: Sinyal `SIGINT` dikirim ke grup proses, diberi tenggat waktu 5 detik sebelum dipaksa `SIGKILL`.
- **Live Ingest Snapshot Monitor (Interval 30 Detik)**:
  - Supervisor mengambil cuplikan visual (frame JPEG thumbnail) langsung dari pipeline feed RTMP tanpa mengganggu kontinuitas stream.
  - Disajikan langsung pada slot card di web UI untuk verifikasi visual streaming.
  - Metrik stream health visual: **PTS Sync** (ms), **Keyframe Cadence** (GOP interval), dan **Net Latency** (ms).
- **Realtime Telemetry Parsing**:
  - Ekstraksi kontinu dari `stderr` FFmpeg: frame count, FPS aktual, bitrate realtime (kbps), durasi stream, speed multiplier, dan dropped packets.

### 3.3. Smart Scheduler & Overlap Guard
- Penjadwalan berbasis in-memory cron runner (`robfig/cron/v3`).
- **Auto-Schedule Overlap Guard**:
  - Deteksi bentrok waktu antar jadwal streaming.
  - Kebijakan resolusi bentrok yang dapat dikonfigurasi:
    1. *Yield to Highest Priority (Slot 1)*.
    2. *Terminate Preceding Stream Gracefully* (stream sebelumnya dihentikan secara aman).
    3. *Deny New Ingest Until Slot Clears* (jadwal baru ditunda hingga slot bersih).
  - Indikator status visual pada header dan tab jadwal: `OVERLAP GUARD ON` / `OVERLAP DETECTED`.
- Otomasi tugas pemeliharaan sistem via cron:
  - Daily Stream Re-align & PTS Flush.
  - Clear Log Buffer Cache.
  - Auto Database Vacuum & Storage Health Check.

### 3.4. Remote Access & Alert System (CGNAT Solution)
- **Built-in Cloudflare Tunnel Integration**:
  - Dukungan **Quick Tunnel** otomatis: Menjalankan binary `cloudflared` bawaan untuk menghasilkan URL publik acak (`*.trycloudflare.com`) secara instan tanpa registrasi akun.
  - Dukungan **Named Tunnel**: Menggunakan token tunnel Cloudflare untuk domain kustom permanen pengguna.
  - Status indikator tunnel di sidebar UI: `TUNNEL ONLINE / OFFLINE`.
- **Tailscale Status Detection**:
  - Mendeteksi apakah perangkat terhubung ke mesh VPN Tailscale dan menampilkan IP 100.x.y.z untuk akses privat aman.
- **Telegram & Discord Webhook Alerts**:
  - Pengiriman notifikasi darurat secara otomatis ke channel Discord dan grup Telegram pengguna:
    1. *Stream Crash Event*: Peringatan saat stream terputus tiba-tiba berikut pesan error FFmpeg.
    2. *Thermal SoC Overload*: Peringatan saat sensor suhu prosesor melampaui batas aman (> 48°C pada handphone).
    3. *Low Memory / Storage Pool*: Peringatan saat sisa kapasitas RAM/disk di bawah batas minimum (< 5 GB storage).

### 3.5. Media Library & In-App Codec Fixer (Point 4)
- **Video Library**:
  - Upload file video melalui UI dengan progress bar dan validasi format.
  - Analisis otomatis metadata melalui `ffprobe`: Resolusi, codec video/audio, bitrate, durasi, framerate, dan status kepatuhan passthrough (`is_passthrough_ready`).
- **In-App Video Codec Fixer (Offline Transcoder)**:
  - Tombol aksi otomatis untuk memperbaiki video yang tidak sesuai standar passthrough.
  - Melakukan transcode background offline menjadi H.264 (High Profile 4.1) + AAC + GOP 2 detik konstan.
  - File hasil konversi diberi label `[PASSTHROUGH READY]`, siap distreamingkan 24/7 tanpa menggunakan komputasi CPU sama sekali.
- **Storage Management**: Visualisasi pemakaian disk video library, estimasi sisa jam streaming, dan pembersihan file lama.

### 3.6. Sistem Autentikasi & Multi-User
- Autentikasi berbasis JWT (JSON Web Token) dengan penyimpanan aman di HTTP-only cookie atau localStorage.
- Enkripsi password menggunakan `bcrypt`.
- Role-based Access Control:
  - **Admin**: Akses konfigurasi global, manajemen slot, tunnel, webhook alerts, hardware metrics, dan user management.
  - **Operator**: Akses pemantauan slot dan pengunggahan video.

---

## 4. Kebutuhan Non-Fungsional (Non-Functional Requirements)

1. **Portabilitas & Cross-Compilation**:
   - Single static binary tanpa dependensi CGO (`CGO_ENABLED=0`).
   - Dapat dikompilasi untuk target `linux/arm64`, `linux/arm`, dan `linux/amd64`.
2. **Resource Footprint Minim**:
   - Penggunaan RAM backend Go dalam kondisi idle < 25 MB.
   - Pemanfaatan CPU backend dalam kondisi idle < 0.5%.
   - Pada mode passthrough, FFmpeg mengonsumsi CPU < 2% pada prosesor ARM mobile.
3. **Desain Visual & UX (Authentic Neobrutalism)**:
   - Antarmuka web mengadopsi gaya Neobrutalisme Otentik: garis batas hitam tegas (`border-2 border-black`), bayangan keras (`shadow-[3px_3px_0px_#000]`), sudut `rounded-lg` / `rounded-xl`, tipografi ganda (`Plus Jakarta Sans` untuk judul/label dan `JetBrains Mono` untuk metrik teknis/telemetri).
   - Palet warna pastel fungsional: Mint (`#c8f5d0`), Blue (`#c2e7ff`), Lavender (`#e2daf9`), Coral (`#ffd5cc`), Yellow (`#fff0a3`), Pink (`#ffd4e5`), Canvas (`#daf0fc`), dan Dark (`#111827`).
4. **Thermal Safety & Battery Health**:
   - Fitur emergency kill switch di dashboard utama untuk menghentikan seluruh proses FFmpeg secara instan jika terdeteksi suhu panas berlebih.
5. **Keamanan**:
   - Pencegahan *Command Injection* pada perakitan argumen FFmpeg (menggunakan slice `[]string`, dilarang shell interpolation).
   - Sanitasi path absolut pada manipulasi file video untuk mencegah *Path Traversal*.
