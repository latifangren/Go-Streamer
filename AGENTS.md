# Repository Instructions & Agent Guidelines

## 1. Repository Overview
- **Purpose**: Autonomous 24/7 RTMP live-streaming node running on low-power devices (Android Termux, Magisk root, Raspberry Pi, VPS, Windows).
- **Architecture**: Go backend + Embedded React 18 SPA frontend bundled via Go `embed.FS` (`embed.go`) into a single zero-CGO binary.
- **Core Capabilities**:
  - Dual-slot RTMP streaming.
  - FFmpeg supervisor with anti-orphan guarantees.
  - Zero-CGO SQLite database (`modernc.org/sqlite`).
  - Live WebSocket metrics (`/ws`).
  - Cloudflare tunnel support (named and quick tunnels).
  - Telegram and webhook alerts.
  - Cron scheduler (`robfig/cron/v3`).

## 2. Project Structure & Architecture
- `cmd/server/main.go`: Application entrypoint, initializes DB, repositories, services, supervisors, HTTP router, and graceful shutdown.
- `embed.go`: Embeds `web/dist` with SPA fallback.
- `internal/`:
  - `alert/`: Notification delivery (Telegram, Webhook, etc.).
  - `api/`: Chi router, REST handlers, middleware, and `/ws` WebSocket hub.
  - `config/`: CLI flags and environment variables (`STREAMER_*`).
  - `domain/`: Business entities and domain interfaces.
  - `repository/sqlite/`: Zero-CGO SQLite persistence.
  - `runner/`: FFmpeg process runner & anti-orphan supervisor.
  - `scheduler/`: Cron-based scheduled stream execution.
  - `service/`: Core application logic (video, transcoder, alert, tunnel).
  - `sysinfo/`: OS and hardware metrics collector (CPU, RAM, temp, battery/thermal for Android/Linux).
  - `transcoder/`: Video probe and fixer tooling.
  - `tunnel/`: Cloudflare named/quick tunnel orchestration.
- `web/`: Frontend React SPA (TypeScript, Vite, Tailwind CSS with Authentic Neobrutalism design).
- `scripts/`: Cross-compilation, build automation (`build.sh`), and Magisk root module packaging.

## 3. Development & Build Workflows

### Frontend Setup & Build
- Install dependencies and build production bundle:
  ```bash
  cd web && npm install && npm run build
  ```
- Dev server:
  ```bash
  cd web && npm run dev
  ```

### Go Backend Verification
```bash
go mod verify
go vet ./...
go test -v ./...
# Race testing:
go test -v -race ./...
```

### Binary Compilation (Zero-CGO Mandatory)
- Linux / Termux / Server:
  ```bash
  CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/go-streamer ./cmd/server
  ```
- Windows:
  ```bash
  go build -ldflags="-s -w" -o bin/go-streamer.exe ./cmd/server
  ```

### Running Locally
```bash
go run ./cmd/server -port 8080 -data ./data
```

## 4. Strict Engineering Rules & Guardrails
- **Zero CGO**: MUST NEVER introduce dependencies requiring CGO or native C libraries. SQLite must remain pure Go (`modernc.org/sqlite`).
- **Process Safety**: FFmpeg and Cloudflared processes MUST be tracked, supervised, and killed cleanly on shutdown to prevent orphaned streaming processes from exhausting mobile/SBC hardware.
- **Frontend Embedding**: Before building a release binary, frontend assets must be compiled (`npm run build`) so `web/dist` is populated for `embed.go`.
- **UI Design System**: Frontend uses an **Authentic Neobrutalism** aesthetic (`border-2 border-black`, hard drop shadows `shadow-[...px_black]`, crisp bold typography, vibrant accent cards). Preserve this styling consistently; do not flatten or genericize UI elements.
- **Concurrency**: Goroutines must respect `context.Context` cancellation and avoid unbuffered channel deadlocks.

## 5. Agent Delegation & Role Conventions
- **@designer**: UI/UX layout, design polish, and styling in `web/`
- **@fixer**: Mechanical Go/TS code edits, refactors, and test additions
- **@librarian**: External library questions & docs research
- **@oracle**: Architectural decisions, process supervisor design, concurrency debugging
- **@explorer**: Codebase exploration & symbol search
- **@fast-generic**: Routine verification, git commits, and builds
