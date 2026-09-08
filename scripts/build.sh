#!/usr/bin/env bash
set -e

# Pastikan eksekusi dari root direktori proyek
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

echo "=== [1/4] Building Frontend Assets ==="
if [ -f "web/package.json" ]; then
    echo "Building web frontend from source..."
    (cd web && npm install && npm run build)
else
    echo "Notice: web/package.json not found. Using pre-bundled or placeholder assets in web/dist."
fi

# Pastikan web/dist/index.html ada
if [ ! -f "web/dist/index.html" ]; then
    echo "Error: web/dist/index.html is required for Go embed." >&2
    exit 1
fi

BUILD_DIR="$ROOT_DIR/build"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

LDFLAGS="-s -w"

echo "=== [2/4] Cross-Compiling Go Binaries ==="

echo "--> Building Linux ARM64 (Android 64-bit / Raspberry Pi 4/5)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/go-streamer-linux-arm64" ./cmd/server

echo "--> Building Linux AMD64 (VPS Server / Cloud)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/go-streamer-linux-amd64" ./cmd/server

echo "--> Building Windows AMD64..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/go-streamer-windows-amd64.exe" ./cmd/server

echo "=== [3/4] Packaging Magisk Module ==="
MAGISK_STAGING="$BUILD_DIR/magisk_package"
rm -rf "$MAGISK_STAGING"
mkdir -p "$MAGISK_STAGING/system/bin"

cp "$ROOT_DIR/scripts/magisk/module.prop" "$MAGISK_STAGING/module.prop"
cp "$ROOT_DIR/scripts/magisk/service.sh" "$MAGISK_STAGING/service.sh"
cp "$BUILD_DIR/go-streamer-linux-arm64" "$MAGISK_STAGING/system/bin/go-streamer"

chmod 755 "$MAGISK_STAGING/service.sh"
chmod 755 "$MAGISK_STAGING/system/bin/go-streamer"

MAGISK_ZIP="$BUILD_DIR/go-streamer-magisk-v1.0.0.zip"
rm -f "$MAGISK_ZIP"

if command -v zip >/dev/null 2>&1; then
    (cd "$MAGISK_STAGING" && zip -r -9 "$MAGISK_ZIP" .)
else
    python3 -c "
import zipfile, os
with zipfile.ZipFile('$MAGISK_ZIP', 'w', zipfile.ZIP_DEFLATED) as zf:
    for root, dirs, files in os.walk('$MAGISK_STAGING'):
        for file in files:
            path = os.path.join(root, file)
            arcname = os.path.relpath(path, '$MAGISK_STAGING')
            zf.write(path, arcname)
" 2>/dev/null || (cd "$MAGISK_STAGING" && tar -czvf "$MAGISK_ZIP" .)
fi

echo "=== [4/4] Build Complete! ==="
echo "Artifacts generated in $BUILD_DIR:"
ls -lh "$BUILD_DIR"
