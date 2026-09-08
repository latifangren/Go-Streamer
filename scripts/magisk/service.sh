#!/system/bin/sh
# Go-Streamer Magisk Late-Start Service Script
# Runs in background when Android boot process finishes

MODDIR=${0%/*}

# 1. Tunggu hingga sistem Android selesai booting sepenuhnya
until [ "$(getprop sys.boot_completed)" = "1" ]; do
    sleep 3
done

# 2. Setup PATH agar mencakup binary modul Magisk
export PATH="$MODDIR/system/bin:$PATH"

# 3. Buat direktori data penyimpanan internal
DATA_DIR="/data/local/go-streamer-data"
mkdir -p "$DATA_DIR"
chmod 755 "$DATA_DIR"

# 4. Optimasi CPU governor untuk kestabilan termal dan efisiensi 24/7 (schedutil / powersave)
for gov in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
    if [ -f "$gov" ]; then
        avail_gov="${gov%/*}/scaling_available_governors"
        if [ -f "$avail_gov" ] && grep -q "schedutil" "$avail_gov" 2>/dev/null; then
            echo "schedutil" > "$gov" 2>/dev/null
        elif [ -f "$avail_gov" ] && grep -q "powersave" "$avail_gov" 2>/dev/null; then
            echo "powersave" > "$gov" 2>/dev/null
        fi
    fi
done

# 5. Jalankan daemon Go-Streamer
DAEMON_BIN="$MODDIR/system/bin/go-streamer"
if [ -f "$DAEMON_BIN" ]; then
    chmod 755 "$DAEMON_BIN" 2>/dev/null
    "$DAEMON_BIN" -port 80 -data "$DATA_DIR" > "$DATA_DIR/daemon.log" 2>&1 &
    DAEMON_PID=$!

    sleep 1

    # 6. Set prioritas anti-LMK (Low Memory Killer) agar daemon tidak ditutup OS
    if [ -d "/proc/$DAEMON_PID" ]; then
        echo -1000 > "/proc/$DAEMON_PID/oom_score_adj" 2>/dev/null
    fi
fi
