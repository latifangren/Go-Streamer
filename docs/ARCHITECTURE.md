# Architecture & Technical Design Document — Go-Streamer

## 1. High-Level System Architecture

Aplikasi dirancang sebagai **Single Self-Contained Binary** yang menggabungkan backend REST & WebSocket API, frontend SPA terkompilasi melalui Go `embed.FS`, database SQLite tanpa CGO, in-memory scheduler, process supervisor untuk FFmpeg, Linux procfs telemetry collector, controller tunneling (Cloudflare/Tailscale), dan dispatcher webhook alerts.

```
+-----------------------------------------------------------------------------------------------+
|                                  CLIENT (BROWSER / MOBILE WEB)                                |
|                                                                                               |
|       +-------------------------------------------------------------------------------+       |
|       |         Vite + React 18 + Tailwind CSS SPA (Authentic Neobrutalism UI)        |       |
|       |  - Dual-Slot Control Desk & Live Snapshot Ingest Monitor (PTS Sync, GOP, Lat) |       |
|       |  - Playlist Sequential Queue Visualizer ([PLAYING], [NEXT], [QUEUED])          |       |
|       |  - Smart Scheduler with Collision Overlap Guard                               |       |
|       |  - Video Library & In-App Codec Fixer (Offline Transcode Pipeline)            |       |
|       |  - Remote Access (Cloudflare Tunnel) & Alerting Setup (Discord / Telegram)    |       |
|       +---------------------------------------+---------------------------------------+       |
+-----------------------------------------------|-----------------------------------------------+
                                                | (HTTP REST / WebSocket Full-Duplex)
                        +-----------------------+-----------------------+
                        | (Localhost / WiFi)                            | (Remote via CGNAT)
                        v                                               v
+-----------------------------------------------+       +---------------------------------------+
|             LOCAL HTTP LISTENER               |       |         BUILT-IN TUNNEL MANAGER       |
|             (e.g., :8080 or :80)              |       |   (Cloudflare Quick/Named / Tailscale)|
+-----------------------+-----------------------+       +-------------------+-------------------+
                        |                                                   |
                        +-----------------------+---------------------------+
                                                |
+-----------------------------------------------v-----------------------------------------------+
|                                     GO-STREAMER CORE BINARY                                   |
|                                                                                               |
|  +-----------------------------+        +--------------------------------------------------+  |
|  |     Embedded Frontend       |        |                HTTP & WebSocket Layer            |  |
|  |     (//go:embed web/dist)   |        |            (chi Router / Go 1.22+ stdlib)        |  |
|  +-----------------------------+        +-------------------------+------------------------+  |
|                                                                   |                           |
|             +---------------------+-------------------------------+---------------------+     |
|             |                     |                               |                     |     |
|  +----------v---------+  +--------v----------+          +---------v----------+  +-------v---+ |
|  | Service Layer      |  | Scheduler Engine  |          | Process Supervisor |  | Alert Hub | |
|  | (Slot, Video, Auth,|  | (robfig/cron/v3 + |          | (Runner, Concat,   |  | (Telegram | |
|  |  Playlist, Codec)  |  |  Overlap Guard)   |          |  Snapshot & Parse) |  |  & Discord)|
|  +----------+---------+  +--------+----------+          +---------+----------+  +-------+---+ |
|             |                     |                               |                     |     |
|  +----------v---------+           +---------------+---------------+                     |     |
|  |   SQLite Storage   |                           |                                     |     |
|  | (modernc.org/sqlite|                  +--------v--------+                            |     |
|  |   WAL Mode ACID)   |                  | Procfs Collector|                            |     |
|  +--------------------+                  | (Linux /proc)   |----------------------------+     |
|                                          +--------+--------+ (Threshold Triggers:             |
|                                                   |           Temp > 48C, RAM, Storage)       |
+---------------------------------------------------|-------------------------------------------+
                                                    | (os/exec + Process Group + Pdeathsig)
+---------------------------------------------------v-------------------------------------------+
|                                        OS KERNEL & HARDWARE                                   |
|                                                                                               |
|      +-------------------------+                           +-------------------------+        |
|      |      FFmpeg Slot 1      |                           |      FFmpeg Slot 2      |        |
|      | (Concat Playlist / Copy)|                           | (Single File / Transcode|        |
|      | [Snapshot pipe 30s int] |                           |  + Dynamic WIB Overlay) |        |
|      +------------+------------+                           +------------+------------+        |
|                   | (RTMP Ingest)                                       | (RTMP Ingest)       |
|                   v                                                     v                     |
|         [ YouTube / Facebook ]                                 [ Twitch / Custom ]            |
+-----------------------------------------------------------------------------------------------+
```

---

## 2. Detailed Directory & Package Structure

```
Go-Streamer/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point aplikasi (bootstrapping seluruh modul)
├── internal/
│   ├── alert/                      # Webhook & Remote Alert System
│   │   ├── dispatcher.go           # Worker queue pengiriman alert ber-rate-limit
│   │   ├── discord.go              # Formatter & HTTP client Discord Webhook
│   │   └── telegram.go             # Bot API client Telegram (SendMessage)
│   ├── api/                        # HTTP & WebSocket Layer
│   │   ├── handler/
│   │   │   ├── auth_handler.go
│   │   │   ├── codec_handler.go    # Endpoint trigger offline video transcoder
│   │   │   ├── schedule_handler.go # Endpoint CRUD jadwal & query overlap status
│   │   │   ├── slot_handler.go     # Endpoint manajemen slot & playlist queue
│   │   │   ├── system_handler.go   # Endpoint telemetry, killswitch & system info
│   │   │   ├── tunnel_handler.go   # Endpoint kontrol Cloudflare tunnel & Tailscale
│   │   │   └── video_handler.go    # Upload chunked, probe, delete, rename
│   │   ├── middleware/             # Auth JWT, Rate Limiter, Logger, Panic Recovery
│   │   ├── router.go               # Inisialisasi rute chi/mux
│   │   └── ws/                     # WebSocket Hub, Client connection, Event broadcaster
│   ├── config/                     # Konfigurasi aplikasi (ENV / Flags / Defaults)
│   ├── domain/                     # Domain Entities & Interfaces
│   │   ├── alert.go                # Model webhook settings & trigger events
│   │   ├── errors.go               # Domain custom errors
│   │   ├── metrics.go              # Model struct metrik sistem, stream & snapshot
│   │   ├── playlist.go             # Model playlist items ([PLAYING], [NEXT], [QUEUED])
│   │   ├── schedule.go             # Model jadwal & overlap guard policy
│   │   ├── slot.go                 # Model slot streaming & overlay configuration
│   │   ├── transcoder.go           # Model offline codec fixer jobs
│   │   ├── tunnel.go               # Model tunnel providers & states
│   │   ├── user.go                 # Model user & auth
│   │   └── video.go                # Model media video & ffprobe metadata
│   ├── repository/                 # Data Access Layer (SQLite)
│   │   ├── sqlite/
│   │   │   ├── alert_repo.go
│   │   │   ├── db.go               # Inisialisasi modernc.org/sqlite & DDL migration
│   │   │   ├── playlist_repo.go
│   │   │   ├── schedule_repo.go
│   │   │   ├── slot_repo.go
│   │   │   ├── transcoder_repo.go
│   │   │   ├── tunnel_repo.go
│   │   │   ├── user_repo.go
│   │   │   └── video_repo.go
│   ├── runner/                     # FFmpeg Process Supervisor & Engine
│   │   ├── command.go              # Argument builder (Passthrough, Transcode, Overlays)
│   │   ├── concat.go               # Generator file playlist concat demuxer (-f concat)
│   │   ├── parser.go               # Stderr line parser (frame, fps, bitrate, speed, drop)
│   │   ├── probe.go                # ffprobe wrapper (codec, GOP, PTS sync check)
│   │   ├── snapshot.go             # Ingest frame snapshot extractor (interval 30 detik)
│   │   └── supervisor.go           # Master lifecycle (Start, Stop, Auto-Restart, Anti-Zombie)
│   ├── scheduler/                  # In-Memory Cron Runner
│   │   ├── overlap_guard.go        # Collision detection algorithm & arbitration policies
│   │   └── scheduler.go            # robfig/cron/v3 wrapper + maintenance tasks
│   ├── service/                    # Business Logic Layer
│   │   ├── alert_service.go
│   │   ├── auth_service.go
│   │   ├── schedule_service.go
│   │   ├── stream_service.go
│   │   ├── transcoder_service.go   # Offline video standardization worker
│   │   ├── tunnel_service.go       # Child process cloudflared & tailscale detector
│   │   └── video_service.go
│   ├── sysinfo/                    # Hardware & OS Telemetry Collector
│   │   ├── linux.go                # Membaca /proc/stat, /proc/meminfo, /sys/class/thermal
│   │   ├── platform.go             # Deteksi root (UID 0), Termux, OS, Arsitektur CPU
│   │   └── windows.go              # Stubs untuk environment development Windows
│   └── transcoder/                 # Offline Video Codec Fixer Worker
│       └── fixer.go                # Queue worker transcode H.264 High 4.1 + AAC + 2s GOP
├── web/                            # Frontend SPA (Vite + React 18 + Tailwind CSS)
│   ├── src/
│   │   ├── api/                    # REST client & WebSocket hooks
│   │   ├── components/             # Neobrutalist UI Kit (Cards, Buttons, Modals, Badges)
│   │   ├── pages/                  # ControlDesk, VideoTools, SchedulerTab, RemoteAlerts
│   │   ├── styles/                 # Tailwind theme tokens (Authentic Neobrutalism)
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   ├── vite.config.ts
│   └── dist/                       # Output build frontend (di-embed ke binary Go)
├── embed.go                        # Directive `//go:embed web/dist/*`
├── scripts/
│   ├── build.sh                    # Multi-architecture compile script
│   └── magisk/                     # Modul Magisk untuk Android Root
│       ├── module.prop
│       ├── service.sh
│       └── system/bin/
├── docs/
│   ├── PRD.md
│   ├── ARCHITECTURE.md
│   ├── PLATFORM_GUIDE.md
│   ├── ROADMAP.md
│   └── design/                     # Desain UI Mockup & Style Guide
│       ├── DESIGN.md
│       ├── code.html
│       └── screen.png
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 3. Database Schema (Pure-Go SQLite)

Menggunakan driver CGO-free `modernc.org/sqlite` dengan mode WAL (*Write-Ahead Logging*) untuk performa concurrency tinggi.

```sql
-- 1. Pengguna (Multi-User & Role)
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user', -- 'admin' / 'user'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 2. Media Library (Video Files & Metadata)
CREATE TABLE IF NOT EXISTS videos (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    filename TEXT NOT NULL,
    original_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    duration_seconds REAL DEFAULT 0,
    resolution TEXT,
    video_codec TEXT,
    audio_codec TEXT,
    fps REAL DEFAULT 30.0,
    gop_size REAL DEFAULT 2.0,            -- Keyframe cadence (detik)
    is_passthrough_ready BOOLEAN DEFAULT 0,-- 1 jika H.264 + AAC + 2s GOP
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 3. Slot Streaming & Konfigurasi Overlay
CREATE TABLE IF NOT EXISTS stream_slots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slot_number INTEGER NOT NULL UNIQUE,   -- 1, 2
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'idle',   -- 'idle', 'starting', 'running', 'error'
    source_type TEXT NOT NULL DEFAULT 'single', -- 'single' / 'playlist'
    video_id TEXT,                         -- Digunakan jika source_type = 'single'
    target_platform TEXT NOT NULL,         -- 'youtube', 'facebook', 'twitch', 'custom'
    rtmp_url TEXT NOT NULL,
    stream_key TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'copy',     -- 'copy' (passthrough) / 'transcode'
    quality TEXT DEFAULT '720p',
    preset TEXT DEFAULT 'ultrafast',
    loop_playback BOOLEAN DEFAULT 1,       -- 1 = loop terus menerus
    max_duration_minutes INTEGER DEFAULT 0,
    auto_restart BOOLEAN DEFAULT 1,
    
    -- Fitur Dynamic Overlay
    enable_overlay BOOLEAN DEFAULT 0,
    overlay_clock_wib BOOLEAN DEFAULT 0,   -- Burn-in jam digital WIB
    overlay_watermark TEXT,                -- Teks watermark (opsional)
    overlay_watermark_pos TEXT DEFAULT 'top_right',
    
    -- Health & Telemetry State
    last_pts_sync_ms REAL DEFAULT 0.0,
    last_keyframe_gop_s REAL DEFAULT 2.0,
    last_net_latency_ms INTEGER DEFAULT 0,
    last_snapshot_path TEXT,               -- Path JPEG snapshot terakhir
    last_snapshot_at DATETIME,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(video_id) REFERENCES videos(id) ON DELETE SET NULL
);

-- 4. Playlist Items (Sequential Looping Mode)
CREATE TABLE IF NOT EXISTS playlist_items (
    id TEXT PRIMARY KEY,
    slot_id INTEGER NOT NULL,
    video_id TEXT NOT NULL,
    order_index INTEGER NOT NULL,          -- Urutan putar (0, 1, 2, ...)
    play_status TEXT NOT NULL DEFAULT 'queued', -- 'playing', 'next', 'queued'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(slot_id) REFERENCES stream_slots(id) ON DELETE CASCADE,
    FOREIGN KEY(video_id) REFERENCES videos(id) ON DELETE CASCADE
);

-- 5. Penjadwalan & Collision Overlap Guard
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    slot_id INTEGER NOT NULL,
    cron_expr TEXT NOT NULL,               -- Misal "0 8 * * *"
    duration_minutes INTEGER DEFAULT 120,
    is_enabled BOOLEAN DEFAULT 1,
    overlap_guard_policy TEXT DEFAULT 'yield_priority', 
    -- 'yield_priority' : Kalah dari Slot 1 jika bentrok
    -- 'terminate_prev' : Matikan stream yang sedang aktif secara graceful
    -- 'deny_new'       : Tunda eksekusi hingga slot benar-benar idle
    last_run_at DATETIME,
    next_run_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(slot_id) REFERENCES stream_slots(id) ON DELETE CASCADE
);

-- 6. Offline Transcoder Jobs (In-App Video Codec Fixer)
CREATE TABLE IF NOT EXISTS transcode_jobs (
    id TEXT PRIMARY KEY,
    source_video_id TEXT NOT NULL,
    target_file_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued', -- 'queued', 'processing', 'completed', 'failed'
    progress_percent REAL DEFAULT 0.0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(source_video_id) REFERENCES videos(id) ON DELETE CASCADE
);

-- 7. Remote Access & Tunnel Settings
CREATE TABLE IF NOT EXISTS tunnel_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    provider TEXT NOT NULL DEFAULT 'cloudflare', -- 'cloudflare' / 'tailscale'
    mode TEXT NOT NULL DEFAULT 'quick',         -- 'quick' (trycloudflare) / 'named'
    tunnel_token TEXT,
    is_active BOOLEAN DEFAULT 0,
    public_url TEXT,
    last_status TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 8. Webhook Alerts (Telegram & Discord)
CREATE TABLE IF NOT EXISTS alert_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    telegram_enabled BOOLEAN DEFAULT 0,
    telegram_bot_token TEXT,
    telegram_chat_id TEXT,
    discord_enabled BOOLEAN DEFAULT 0,
    discord_webhook_url TEXT,
    trigger_on_crash BOOLEAN DEFAULT 1,
    trigger_on_thermal BOOLEAN DEFAULT 1,
    thermal_threshold_c REAL DEFAULT 48.0, -- Default 48°C untuk HP mobile
    trigger_on_low_storage BOOLEAN DEFAULT 1,
    low_storage_threshold_gb REAL DEFAULT 5.0,
    last_alert_sent_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. Process Supervisor, FFmpeg Concat & Ingest Snapshot Pipeline

### 4.1. Playlist Sequential Looping Engine (Concat Demuxer)
Ketika slot dikonfigurasi dengan mode playlist:
1. `internal/runner/concat.go` membuat file manifest teks `playlist_slot_N.txt` di disk/tmp:
   ```text
   ffconcat version 1.0
   file '/data/videos/promo_1.mp4'
   file '/data/videos/promo_2.mp4'
   file '/data/videos/bumper.mp4'
   ```
2. Parameter eksekusi FFmpeg:
   ```bash
   ffmpeg -re \
     -stream_loop -1 \
     -f concat \
     -safe 0 \
     -i /data/tmp/playlist_slot_1.txt \
     -c copy \
     -flvflags no_duration_filesize \
     -f flv \
     "rtmp://..."
   ```
   *Keuntungan*: Transisi antar file video mulus tanpa jeda hitam (gapless) dan **0% konsumsi CPU** karena tetap menggunakan `-c copy`.

### 4.2. Dynamic Stream Overlay Pipeline
Jika user mengaktifkan overlay jam digital WIB atau watermark:
- FFmpeg beralih secara otomatis dari Stream Copy ke transcode ringan (`-c:v libx264 -preset ultrafast`):
  ```bash
  -vf "drawtext=fontfile=/system/fonts/Roboto-Regular.ttf:text='%{localtime\:%H\\\:%M\\\:%S WIB}':x=w-tw-20:y=20:fontsize=24:fontcolor=white@0.9:box=1:boxcolor=black@0.6,drawtext=text='LIVE STREAM':x=20:y=20:fontsize=20:fontcolor=yellow@0.9"
  ```

### 4.3. Live Ingest Snapshot Monitor (Interval 30 Detik)
Untuk menyediakan verifikasi visual pada kartu slot dashboard:
1. Setiap 30 detik, goroutine di `internal/runner/snapshot.go` mengambil cuplikan frame 1 frame JPEG beresolusi rendah (480x270):
   ```bash
   ffmpeg -ss 00:00:01 -i /path/to/current_video.mp4 -vframes 1 -q:v 5 -vf "scale=480:-1" -y /data/cache/snapshot_slot_1.jpg
   ```
2. Gambar snapshot disajikan melalui HTTP API `/api/v1/slots/:id/snapshot` atau di-encode ke base64 data URI via WebSocket.
3. Supervisor menghitung estimasi metrik kesehatan stream:
   - **PTS Sync**: Dihitung dari delta timestamp audio vs video pada paket container FLV.
   - **Keyframe Cadence**: Divalidasi dari GOP time header (standar 2.00s).
   - **Net Latency**: Dihitung dari Round-Trip Time socket TCP ke RTMP Ingest server.

---

## 5. Remote Access (CGNAT) & Proactive Webhook Alerts

### 5.1. Built-in Cloudflare Tunnel Controller
Mengatasi limitasi alamat IP operator seluler yang berada di balik CGNAT:
- Go-Streamer memaketkan/mengintegrasikan controller child process untuk `cloudflared`:
  1. **Quick Tunnel**:
     ```bash
     cloudflared tunnel --url http://127.0.0.1:8080 --no-autoupdate
     ```
     Supervisor membaca output `stderr` dari `cloudflared`, mengekstrak URL publik acak `https://[random].trycloudflare.com` menggunakan regex, dan menyimpannya ke tabel `tunnel_settings` serta mem-broadcast status `TUNNEL ONLINE` ke UI.
  2. **Named Tunnel**:
     Jika pengguna memasukkan token Cloudflare permanen, binary mengeksekusi `cloudflared tunnel run --token <TOKEN>`.

### 5.2. Telegram & Discord Alert Dispatcher
Goroutine `internal/alert/dispatcher.go` menerima sinyal dari procfs collector dan supervisor:
- **Deduplikasi & Cooldown**: Mencegah banjir pesan (*flood*) dengan memberlakukan batas waktu jeda (cooldown) minimal 5 menit untuk jenis alert yang sama.
- **Payload Discord**: Mengirim rich embed dengan warna status (Merah = Crash/Danger, Kuning = Warning, Hijau = Recovered).
- **Payload Telegram**: Mengirim pesan Markdown lengkap dengan timestamp WIB, metrik suhu SoC, dan kode exit FFmpeg.

---

## 6. In-App Video Codec Fixer (Offline Transcoder Worker)

Untuk memastikan bahwa video yang diunggah dapat menggunakan mode **Passthrough (Stream Copy 0% CPU)**:
1. Saat video diunggah, `probe.go` memeriksa kecocokan codec. Jika codec audio bukan AAC atau video bukan H.264 / GOP bukan 2s, tombol **"Fix Codec for 24/7 Passthrough"** muncul di UI.
2. Saat user menekan tombol, sistem memasukkan job ke tabel `transcode_jobs`.
3. Background worker `internal/transcoder/fixer.go` menjalankan proses konversi satu per satu (single concurrency agar CPU tidak kepanasan):
   ```bash
   ffmpeg -i input_raw.mp4 \
     -c:v libx264 -profile:v high -level:v 4.1 \
     -preset slow -crf 20 \
     -g 60 -keyint_min 60 -sc_threshold 0 \
     -c:a aac -b:a 128k -ar 44100 \
     output_passthrough_ready.mp4
   ```
4. Progress konversi dilaporkan secara realtime ke UI. Setelah selesai, file video ditandai `is_passthrough_ready = 1`.

---

## 7. Frontend Design Architecture (Authentic Neobrutalism)

Mengadopsi spesifikasi desain dari `docs/design/DESIGN.md` dan `docs/design/code.html`:

### 7.1. Design Tokens & Styling Rules
- **Color Palette**:
  - `neoCanvas`: `#daf0fc` (Latar belakang utama bernuansa pastel cyan sejuk)
  - `neoMint`: `#c8f5d0` (Status aktif, tombol sukses, badge OK)
  - `neoBlue`: `#c2e7ff` (Card CPU, tombol info)
  - `neoLavender`: `#e2daf9` (Banner codec optimizer, tab aktif)
  - `neoCoral`: `#ffd5cc` (Tombol killswitch, temperatur tinggi, badge error)
  - `neoYellow`: `#fff0a3` (Badge warning, highlight jam WIB)
  - `neoPink`: `#ffd4e5` (Overlap guard badge, indikator storage)
  - `neoDark`: `#111827` (Warna teks gelap, kontras tinggi)
- **Border & Shadow**:
  - Border tebal: `border-2 border-black` atau `border-[2.5px] border-black`
  - Hard Offset Shadow: `shadow-[3px_3px_0px_#000]`, `shadow-[5px_5px_0px_#000]`
  - Sudut: `rounded-lg` dan `rounded-xl`
  - Interaksi tombol (`.neo-btn`): `active:translate-x-[2px] active:translate-y-[2px] active:shadow-none transition-all`
- **Tipografi**:
  - Font sans: `Plus Jakarta Sans` (Judul besar uppercase font-black, label form)
  - Font monospace: `JetBrains Mono` (Angka metrik, telemetri, kode perintah, status log)

### 7.2. 4 Tab Utama Navigation
1. **Control Desk**:
   - 4 Kartu Metrik Wavy: CPU Load, RAM Memory, SoC Thermal (suhu dengan indikator status), dan Storage Buffer.
   - Dual-Slot Interactive Cards: Live Ingest Snapshot (30s interval), stream telemetry (FPS, bitrate, duration, speed), visual playback queue (`[PLAYING]`, `[NEXT]`, `[QUEUED]`), kontrol Start/Stop/Config.
2. **Video & Codec Tools**:
   - Banner Offline Codec Fixer.
   - Video file listing dengan badge kepatuhan passthrough (`PASSTHROUGH READY` vs `NEEDS CONVERSION`).
   - Modal unggah video dengan progress bar chunked upload.
3. **Smart Scheduler**:
   - Overlap Guard status indicator & collision policy selector.
   - Cron task management & visual schedule timeline.
4. **Remote & Tunnel**:
   - Cloudflare Quick Tunnel toggle & generated public URL display.
   - Tailscale IP detection status.
   - Konfigurasi notifikasi Telegram Bot dan Discord Webhook.
