# Platform & Deployment Guide — Go-Streamer

Panduan ini berisi instruksi teknis, mitigasi limitasi OS, konfigurasi sistem, dan strategi encoding FFmpeg untuk menjalankan **Go-Streamer** di berbagai target platform: Android Termux (Non-Root), Android Magisk Module (Root), PostmarketOS / Linux ARM, dan Server Linux x86_64.

---

## 1. Android Termux (Mode Non-Root)

Menjalankan server streaming di Android tanpa root memiliki tantangan utama berupa manajemen daya agresif Android (*Doze Mode*, *Low Memory Killer*, dan *Phantom Process Killer* pada Android 12+).

### 1.1. Prasyarat & Instalasi
Di dalam aplikasi Termux:
```bash
# Update repository
pkg update && pkg upgrade -y

# Install dependensi multimedia
pkg install ffmpeg -y

# Verifikasi instalasi FFmpeg
ffmpeg -version
```

### 1.2. Mitigasi Phantom Process Killer (Android 12, 13, 14+)
Android 12+ secara otomatis mematikan child process (FFmpeg) jika Termux memiliki lebih dari 32 proses anak atau menggunakan komputasi CPU di background secara terus-menerus.

**Solusi (via ADB dari PC satu kali):**
Jalankan perintah ini via ADB di komputer dengan USB Debugging aktif:
```bash
adb shell "/system/bin/device_config set_sync_disabled_for_tests persistent"
adb shell "/system/bin/device_config put activity_manager max_phantom_processes 2147483647"
```

### 1.3. Mencegah Android Doze (Wakelock & Battery Optimization)
1. **Buka Pengaturan HP**: Masuk ke *App Info* -> *Termux* -> *Battery* -> Pilih **"Unrestricted"** (Tidak dibatasi).
2. **Kunci Wakelock Termux**:
   ```bash
   termux-wake-lock
   ```
   Pastikan notifikasi "Termux wake lock held" muncul di status bar agar CPU tidak *deep sleep* saat layar mati.

### 1.4. Menjalankan Go-Streamer di Termux
1. Letakkan binary `go-streamer` di direktori home Termux (`/data/data/com.termux/files/home/`).
2. Beri hak eksekusi:
   ```bash
   chmod +x go-streamer
   ./go-streamer -port 8080 -data ./data
   ```
3. Buka browser di HP atau perangkat lain di jaringan WiFi yang sama: `http://<IP-HP>:8080`.

### 1.5. Autostart via Termux:Boot (Opsional)
Pasang add-on **Termux:Boot** dari F-Droid, lalu buat script boot:
```bash
mkdir -p ~/.termux/boot
cat << 'EOF' > ~/.termux/boot/start-streamer.sh
#!/data/data/com.termux/files/usr/bin/sh
termux-wake-lock
cd /data/data/com.termux/files/home
./go-streamer -port 8080 -data ./data > streamer.log 2>&1 &
EOF
chmod +x ~/.termux/boot/start-streamer.sh
```

---

## 2. Android Modul Magisk / KernelSU (Mode Root)

Mode Root adalah skenario terbaik untuk Android karena aplikasi dapat berjalan sebagai **system daemon mandiri** tanpa perlu membuka aplikasi Termux, kebal pembunuhan LMK kernel, dan bisa bind port privileged (80/443).

### 2.1. Keuntungan Mode Root
- **OOM Score Adjustment**: Set nilai `oom_score_adj = -1000` pada kernel Linux Android, membuat proses Go-Streamer dan FFmpeg memiliki prioritas setara proses sistem Android (kebal LMK).
- **Direct System Service**: Berjalan otomatis begitu HP menyala (*boot completed*).
- **Akses Langsung Node Hardware**: Bebas mengakses `/dev/` dan mengatur *CPU Governor* ke mode hemat panas.

### 2.2. Struktur Modul Magisk
Struktur folder di `/data/adb/modules/go_streamer/`:
```
/data/adb/modules/go_streamer/
├── module.prop
├── service.sh
└── system/
    └── bin/
        ├── go-streamer
        └── ffmpeg
```

### 2.3. Konfigurasi `module.prop`
```ini
id=go_streamer
name=Go-Streamer Daemon
version=v1.0.0
versionCode=100
author=Go-Streamer Team
description=Standalone 24/7 RTMP live streaming daemon powered by Go and FFmpeg.
```

### 2.4. Konfigurasi `service.sh`
Script ini dijalankan otomatis oleh Magisk pada tahap `late-start service`:
```bash
#!/system/bin/sh
MODDIR=${0%/*}

# 1. Tunggu hingga proses booting Android selesai sepenuhnya
until [ "$(getprop sys.boot_completed)" = "1" ]; do
    sleep 3
done

# 2. Setup environment PATH ke binary bawaan modul
export PATH="$MODDIR/system/bin:$PATH"
export STREAMER_DATA="/data/local/go-streamer-data"
mkdir -p "$STREAMER_DATA"

# 3. Jalankan daemon Go-Streamer di background
$MODDIR/system/bin/go-streamer -port 80 -data "$STREAMER_DATA" > "$STREAMER_DATA/daemon.log" 2>&1 &
PID=$!

# 4. KUNCI UTAMA: Lindungi proses dari Low Memory Killer (LMK)
if [ -n "$PID" ]; then
    echo -1000 > "/proc/$PID/oom_score_adj"
fi

# 5. Optimasi Thermal & CPU Governor (Opsional: atur ke powersave/schedutil agar tidak panas)
for gov in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
    [ -f "$gov" ] && echo "schedutil" > "$gov"
done
```

---

## 3. Linux Server & PostmarketOS (ARM / x86_64)

Untuk server VPS Ubuntu/Debian atau HP lama yang di-flash dengan **PostmarketOS / Alpine Linux**:

### 3.1. Systemd Service (`/etc/systemd/system/go-streamer.service`)
```ini
[Unit]
Description=Go-Streamer Live Streaming Service
After=network.target

[Service]
Type=simple
User=streamer
Group=streamer
WorkingDirectory=/opt/go-streamer
ExecStart=/opt/go-streamer/go-streamer -port 8080 -data /opt/go-streamer/data
Restart=always
RestartSec=5s

# Proteksi memori & batasan resource
LimitNOFILE=65535
OOMScoreAdjust=-500

# Kemampuan bind port 80/443 tanpa root
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

Aktifkan service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now go-streamer
```

### 3.2. OpenRC Service (PostmarketOS / Alpine Linux)
Buat file `/etc/init.d/go-streamer`:
```sh
#!/sbin/openrc-run

name="go-streamer"
description="Go-Streamer 24/7 RTMP daemon"
command="/usr/local/bin/go-streamer"
command_args="-port 8080 -data /var/lib/go-streamer"
command_background="yes"
pidfile="/run/${RC_SVCNAME}.pid"

depend() {
    need net
    after firewall
}
```
Aktifkan service:
```sh
chmod +x /etc/init.d/go-streamer
rc-update add go-streamer default
rc-service go-streamer start
```

---

## 4. Strategi Encoding FFmpeg & Efisiensi Hardware

### 4.1. Mengapa Mode Passthrough (Stream Copy) Sangat Dianjurkan?
Pada perangkat mobile (Android / ARM SBC):
- **CPU Encoding (libx264)** membutuhkan kalkulasi jutaan operasi matriks per detik. Menjalankan transcode 1080p akan menaikkan temperatur SoC hingga >75°C, menyebabkan *thermal throttling*, FPS drop menjadi <15 fps, dan baterai cepat aus.
- **Stream Copy (`-c copy`)**: FFmpeg hanya membaca container video dan menyalurkannya langsung ke koneksi socket jaringan RTMP. Beban CPU berkisar antara **0.5% - 2%**, temperatur perangkat tetap dingin, dan mampu streaming 24 jam nonstop tanpa kendala.

### 4.2. Standar Ingest Video untuk Passthrough
Agar video lokal bisa di-stream dengan mode Passthrough ke YouTube, Facebook, atau Twitch tanpa re-encoding:
- **Video Codec**: H.264 / AVC (High Profile 4.1 atau 4.0).
- **Audio Codec**: AAC (Stereo, 44.1 kHz atau 48 kHz, bitrate 128 kbps).
- **Framerate (FPS)**: Konstan (CFR) 30 fps atau 60 fps (bukan VFR / variable framerate).
- **GOP / Keyframe Interval**: 2 detik (misal: `-g 60` pada video 30 fps).

*Fitur probe bawaan Go-Streamer (`ffprobe`) akan otomatis memverifikasi kecocokan file video saat diunggah.*

### 4.3. Formula Perintah FFmpeg Resmi Go-Streamer

#### 1. Mode Passthrough (Default & Mobile Friendly)
```bash
ffmpeg -re \
  -stream_loop -1 \
  -i /path/to/source.mp4 \
  -c copy \
  -flvflags no_duration_filesize \
  -f flv \
  "rtmp://a.rtmp.youtube.com/live2/xxxx-xxxx-xxxx"
```

#### 2. Mode Transcode Software (Khusus Server x86 / PC Bertenaga)
```bash
ffmpeg -re \
  -stream_loop -1 \
  -i /path/to/source.mp4 \
  -c:v libx264 \
  -preset ultrafast \
  -tune zerolatency \
  -b:v 2500k -maxrate 2500k -bufsize 5000k \
  -pix_fmt yuv420p \
  -g 60 \
  -c:a aac -b:a 128k -ar 44100 \
  -flvflags no_duration_filesize \
  -f flv \
  "rtmp://a.rtmp.youtube.com/live2/xxxx-xxxx-xxxx"
```

#### 3. Mode Akselerasi Hardware (Jika Tersedia)
- **NVIDIA GPU (Linux Server)**: Ganti `-c:v libx264` dengan `-c:v h264_nvenc -preset p4 -tune ll`.
- **Raspberry Pi / V4L2 M2M**: Ganti `-c:v libx264` dengan `-c:v h264_v4l2m2m -b:v 2500k`.
- **Intel QuickSync (QSV)**: Ganti `-c:v libx264` dengan `-c:v h264_qsv -global_quality 25`.

---

## 5. Troubleshooting & FAQ

**Q: Kenapa stream di YouTube sering buffering atau terputus setelah beberapa jam?**  
A: Periksa bitrate video terhadap kecepatan upload koneksi internet Anda. Pastikan bitrate video tidak melebihi 70% dari bandwidth upload stabil. Selain itu, pastikan mode passthrough aktif dan keyframe interval video adalah 2 detik.

**Q: Bisakah Go-Streamer berjalan tanpa koneksi internet lokal untuk dashboard?**  
A: Bisa. Karena frontend di-embed langsung ke dalam binary Go, dashboard web dapat diakses secara lokal melalui IP hotspot HP (misal `192.168.43.1:8080`) tanpa memerlukan akses internet untuk merender UI.

**Q: Bagaimana jika listrik padam atau handphone restart?**  
A: Pada mode Root Magisk, Go-Streamer akan hidup otomatis saat booting selesai (`sys.boot_completed=1`). Jika slot streaming dikonfigurasi dengan flag `auto_restart: true`, supervisor akan otomatis menyambungkan kembali stream ke server RTMP.
