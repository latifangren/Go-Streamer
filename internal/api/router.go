package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"go-streamer/internal/alert"
	"go-streamer/internal/api/handler"
	"go-streamer/internal/api/ws"
	"go-streamer/internal/config"
	"go-streamer/internal/domain"
	"go-streamer/internal/repository/sqlite"
	"go-streamer/internal/runner"
	"go-streamer/internal/scheduler"
	"go-streamer/internal/service"
	"go-streamer/internal/sysinfo"
	"go-streamer/internal/transcoder"
	"go-streamer/internal/tunnel"

	streamer "go-streamer"
)

// Server mengorkestrasi konfigurasi routing HTTP REST, WebSocket Hub, background daemon worker, dan repositori.
type Server struct {
	cfg      *config.Config
	db       *sqlite.DB
	platform *domain.PlatformInfo
	router   *chi.Mux

	// Core daemons & background workers
	wsHub           *ws.Hub
	supervisor      *runner.Supervisor
	collector       *sysinfo.Collector
	scheduler       *scheduler.Scheduler
	fixer           *transcoder.CodecFixer
	cfTunnel        *tunnel.CloudflareTunnel
	tsDetector      *tunnel.TailscaleDetector
	alertDispatcher *alert.AlertDispatcher

	// Repositories
	slotRepo     *sqlite.SlotRepository
	videoRepo    *sqlite.VideoRepository
	scheduleRepo *sqlite.ScheduleRepository
	alertRepo    *sqlite.AlertRepository
	tunnelRepo   *sqlite.TunnelRepository
	jobRepo      *sqlite.TranscoderRepository

	// Services
	videoService      *service.VideoService
	transcoderService *service.TranscoderService
	tunnelService     *service.TunnelService
	alertService      *service.AlertService

	// Handlers
	slotHandler     *handler.SlotHandler
	videoHandler    *handler.VideoHandler
	scheduleHandler *handler.ScheduleHandler
	codecHandler    *handler.CodecHandler
	tunnelHandler   *handler.TunnelHandler
	alertHandler    *handler.AlertHandler
	systemHandler   *handler.SystemHandler

	ctx    context.Context
	cancel context.CancelFunc
}

// NewServer membuat instans Server baru dan menghubungkan seluruh dependensi arsitektural.
func NewServer(cfg *config.Config, db *sqlite.DB, platform *domain.PlatformInfo) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		cfg:      cfg,
		db:       db,
		platform: platform,
		router:   chi.NewRouter(),
		ctx:      ctx,
		cancel:   cancel,
	}

	s.initDependencies()
	s.setupMiddleware()
	s.setupRoutes()
	s.startBackgroundWorkers()

	return s
}

func (s *Server) initDependencies() {
	// 1. Repositories
	s.slotRepo = sqlite.NewSlotRepository(s.db.DB)
	s.videoRepo = sqlite.NewVideoRepository(s.db.DB)
	s.scheduleRepo = sqlite.NewScheduleRepository(s.db.DB)
	s.alertRepo = sqlite.NewAlertRepository(s.db.DB)
	s.tunnelRepo = sqlite.NewTunnelRepository(s.db.DB)
	s.jobRepo = sqlite.NewTranscoderRepository(s.db.DB)

	// 2. Telemetri & Runner Engine
	s.collector = sysinfo.NewCollector(s.cfg.DataDir)
	s.supervisor = runner.NewSupervisor(s.platform.FFmpegPath, s.cfg.CacheDir)

	// 3. Scheduler & Maintenance
	s.scheduler = scheduler.NewScheduler(s.supervisor, s.cfg.CacheDir)
	s.scheduler.Start()
	_, _ = s.scheduler.ScheduleDatabaseVacuum(s.db.DB)

	// 4. Remote Tunnel
	s.cfTunnel = tunnel.NewCloudflareTunnel("")
	s.tsDetector = tunnel.NewTailscaleDetector()
	s.tunnelService = service.NewTunnelService(s.tunnelRepo, s.cfTunnel, s.tsDetector, s.cfg.Port)

	// 5. Alert Notifications & Proactive Dispatcher
	tgSender := alert.NewTelegramClient()
	dcSender := alert.NewDiscordClient()
	s.alertDispatcher = alert.NewAlertDispatcher(s.alertRepo, tgSender, dcSender)
	s.alertService = service.NewAlertService(s.alertRepo, s.alertDispatcher)

	// Daftarkan callback crash FFmpeg ke alert dispatcher
	s.supervisor.SetOnCrashCallback(func(slotNumber int, slotName, errMsg string) {
		_ = s.alertDispatcher.TriggerCrash(context.Background(), slotNumber, slotName, errMsg)
	})

	// 6. Transcoder & Video Service
	s.fixer = transcoder.NewCodecFixer(s.platform.FFmpegPath, s.jobRepo, s.videoRepo)
	s.fixer.Start()
	s.transcoderService = service.NewTranscoderService(s.jobRepo, s.videoRepo, s.fixer, s.cfg.VideoDir)
	s.videoService = service.NewVideoService(s.videoRepo, s.cfg.VideoDir, s.cfg.CacheDir, s.platform.FFprobePath, s.platform.FFmpegPath)

	// 7. WebSocket Hub dengan Action Handler
	s.wsHub = ws.NewHub(s.handleWSAction)

	// 8. REST Handlers
	s.slotHandler = handler.NewSlotHandler(s.slotRepo, s.videoRepo, s.supervisor, s.wsHub, s.cfg.CacheDir)
	s.videoHandler = handler.NewVideoHandler(s.videoService, s.wsHub, s.cfg.CacheDir)
	s.scheduleHandler = handler.NewScheduleHandler(s.scheduleRepo, s.slotRepo, s.scheduler, s.wsHub)
	s.codecHandler = handler.NewCodecHandler(s.transcoderService, s.wsHub)
	s.tunnelHandler = handler.NewTunnelHandler(s.tunnelService, s.wsHub)
	s.alertHandler = handler.NewAlertHandler(s.alertService, s.wsHub)
	s.systemHandler = handler.NewSystemHandler(s.collector, s.supervisor, s.wsHub)
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))
}

func (s *Server) setupRoutes() {
	// Root Landing Info
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"app":     "Go-Streamer",
			"version": s.platform.Version,
			"status":  "operational",
			"docs":    "/docs",
		})
	})

	// WebSocket Endpoint (/ws)
	s.router.Get("/ws", s.wsHub.ServeWS)

	// API v1 Endpoints
	s.router.Route("/api/v1", func(r chi.Router) {
		// WebSocket Endpoint (/api/v1/ws) untuk frontend useWebSocket
		r.Get("/ws", s.wsHub.ServeWS)

		// Root API Info
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"app":     "Go-Streamer",
				"version": s.platform.Version,
				"status":  "operational",
				"docs":    "/docs",
			})
		})

		// Health & Platform Info
		r.Get("/health", s.handleHealth)
		r.Get("/system/info", s.handlePlatformInfo)

		// System Telemetry & Killswitch
		r.Get("/system/metrics", s.systemHandler.GetMetrics)
		r.Post("/system/killswitch", s.systemHandler.KillSwitch)

		// Slot Streaming Endpoints
		r.Route("/slots", func(sr chi.Router) {
			sr.Get("/", s.slotHandler.ListSlots)
			sr.Get("/{id}", s.slotHandler.GetSlot)
			sr.Put("/{id}", s.slotHandler.UpdateSlot)
			sr.Post("/{id}/start", s.slotHandler.StartSlot)
			sr.Post("/{id}/stop", s.slotHandler.StopSlot)
			sr.Get("/{id}/snapshot", s.slotHandler.GetSnapshot)
		})

		// Media Management Endpoints
		r.Route("/videos", func(vr chi.Router) {
			vr.Get("/", s.videoHandler.ListVideos)
			vr.Post("/upload", s.videoHandler.UploadVideo)
			vr.Post("/upload-chunk", s.videoHandler.UploadChunk)
			vr.Post("/upload-complete", s.videoHandler.UploadComplete)
			vr.Delete("/{id}", s.videoHandler.DeleteVideo)
			vr.Put("/{id}/rename", s.videoHandler.RenameVideo)
			vr.Get("/storage", s.videoHandler.GetStorageStats)
		})

		// Schedule Endpoints
		r.Route("/schedules", func(schr chi.Router) {
			schr.Get("/", s.scheduleHandler.ListSchedules)
			schr.Post("/", s.scheduleHandler.CreateSchedule)
			schr.Delete("/{id}", s.scheduleHandler.DeleteSchedule)
			schr.Get("/overlap-status", s.scheduleHandler.GetOverlapStatus)
		})

		// Codec Standardization Endpoints
		r.Route("/codec", func(cr chi.Router) {
			cr.Post("/fix", s.codecHandler.CreateFixJob)
			cr.Get("/jobs", s.codecHandler.ListJobs)
			cr.Get("/jobs/{id}", s.codecHandler.GetJob)
		})

		// Remote Tunnel Endpoints
		r.Route("/tunnel", func(tr chi.Router) {
			tr.Get("/status", s.tunnelHandler.GetStatus)
			tr.Post("/start", s.tunnelHandler.StartTunnel)
			tr.Post("/stop", s.tunnelHandler.StopTunnel)
		})

		// Alert Notification Endpoints
		r.Route("/alerts", func(ar chi.Router) {
			ar.Get("/settings", s.alertHandler.GetSettings)
			ar.Put("/settings", s.alertHandler.UpdateSettings)
			ar.Post("/test", s.alertHandler.TestAlert)
		})
	})

	// Mount Embedded SPA Frontend (Fallback for all non-API routes)
	fsys := streamer.GetFileSystem()
	s.router.Handle("/*", http.FileServer(fsys))
}

// startBackgroundWorkers menjalankan tugas latar belakang: telemetri sistem (2s), proactive alert triggers, dan telemetri streaming FFmpeg.
func (s *Server) startBackgroundWorkers() {
	// 1. Pengumpulan metrik sistem setiap 2 detik & Proactive Alert Triggers
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				if s.collector != nil && s.wsHub != nil {
					metrics, err := s.collector.Collect()
					if err == nil && metrics != nil {
						s.wsHub.Broadcast("system_metrics", metrics)

						// Proactive Alert Triggers: Thermal & Low Storage
						if s.alertRepo != nil && s.alertDispatcher != nil {
							if alertSettings, aErr := s.alertRepo.GetSettings(s.ctx); aErr == nil && alertSettings != nil {
								if alertSettings.TriggerOnThermal && metrics.TemperatureC > alertSettings.ThermalThresholdC {
									_ = s.alertDispatcher.TriggerThermal(s.ctx, metrics.TemperatureC, alertSettings.ThermalThresholdC)
								}
								if alertSettings.TriggerOnLowStorage && metrics.DiskFreeGB < alertSettings.LowStorageThresholdGB {
									_ = s.alertDispatcher.TriggerLowStorage(s.ctx, metrics.DiskFreeGB, alertSettings.LowStorageThresholdGB)
								}
							}
						}
					}
				}
			}
		}
	}()

	// 2. Pendengar telemetri streaming FFmpeg dari supervisor
	go func() {
		if s.supervisor == nil || s.wsHub == nil {
			return
		}
		telemetryChan := s.supervisor.TelemetryChan()
		for {
			select {
			case <-s.ctx.Done():
				return
			case tel, ok := <-telemetryChan:
				if !ok {
					return
				}
				if tel != nil {
					s.wsHub.Broadcast("stream_telemetry", tel)
				}
			}
		}
	}()
}

// handleWSAction memproses perintah aksi dari client WebSocket (start_stream, stop_stream).
func (s *Server) handleWSAction(client *ws.Client, action *ws.ClientAction) {
	ctx := context.Background()

	switch action.Action {
	case "start_stream":
		slotNum := action.SlotNumber
		if slotNum == 0 && action.SlotID > 0 {
			if slot, err := s.slotRepo.GetByID(ctx, action.SlotID); err == nil {
				slotNum = slot.SlotNumber
			}
		}

		if slot, err := s.slotRepo.GetBySlotNumber(ctx, slotNum); err == nil {
			var inputSource string
			if slot.SourceType == domain.SourceTypeSingle && slot.VideoID != nil {
				if v, vErr := s.videoRepo.GetByID(ctx, *slot.VideoID); vErr == nil && v != nil {
					inputSource = v.FilePath
				}
			} else if slot.SourceType == domain.SourceTypePlaylist {
				inputSource = filepath.Join(s.cfg.CacheDir, fmt.Sprintf("concat_slot_%d.txt", slot.ID))
			}

			if inputSource != "" && s.supervisor != nil {
				if err := s.supervisor.StartSlot(slot, inputSource); err == nil {
					_ = s.slotRepo.UpdateStatus(ctx, slot.ID, domain.SlotStatusRunning)
					s.wsHub.Broadcast("slot_status_changed", map[string]interface{}{
						"slot_id":     slot.ID,
						"slot_number": slot.SlotNumber,
						"status":      domain.SlotStatusRunning,
					})
				}
			}
		}

	case "stop_stream":
		slotNum := action.SlotNumber
		if slotNum == 0 && action.SlotID > 0 {
			if slot, err := s.slotRepo.GetByID(ctx, action.SlotID); err == nil {
				slotNum = slot.SlotNumber
			}
		}

		if s.supervisor != nil {
			if err := s.supervisor.StopSlot(slotNum); err == nil {
				if slot, err := s.slotRepo.GetBySlotNumber(ctx, slotNum); err == nil {
					_ = s.slotRepo.UpdateStatus(ctx, slot.ID, domain.SlotStatusIdle)
					s.wsHub.Broadcast("slot_status_changed", map[string]interface{}{
						"slot_id":     slot.ID,
						"slot_number": slot.SlotNumber,
						"status":      domain.SlotStatusIdle,
					})
				}
			}
		}
	}
}

// Router mengembalikan handler http.Handler utama server.
func (s *Server) Router() http.Handler {
	return s.router
}

// Supervisor mengembalikan instans runner.Supervisor.
func (s *Server) Supervisor() *runner.Supervisor {
	return s.supervisor
}

// AlertDispatcher mengembalikan instans alert.AlertDispatcher.
func (s *Server) AlertDispatcher() *alert.AlertDispatcher {
	return s.alertDispatcher
}

// AlertRepo mengembalikan instans sqlite.AlertRepository.
func (s *Server) AlertRepo() *sqlite.AlertRepository {
	return s.alertRepo
}

// Collector mengembalikan instans sysinfo.Collector.
func (s *Server) Collector() *sysinfo.Collector {
	return s.collector
}

// Scheduler mengembalikan instans scheduler.Scheduler.
func (s *Server) Scheduler() *scheduler.Scheduler {
	return s.scheduler
}

// DB mengembalikan koneksi SQLite DB.
func (s *Server) DB() *sqlite.DB {
	return s.db
}

// WSHub mengembalikan instans ws.Hub.
func (s *Server) WSHub() *ws.Hub {
	return s.wsHub
}

// Close mematikan background daemons, worker transcoder, supervisor streaming, dan scheduler.
func (s *Server) Close() {
	s.cancel()

	if s.supervisor != nil {
		s.supervisor.StopAll()
	}
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
	if s.cfTunnel != nil {
		_ = s.cfTunnel.Stop()
	}
	if s.fixer != nil {
		s.fixer.Stop()
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"database":  "connected",
	})
}

func (s *Server) handlePlatformInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"platform": s.platform,
		"config": map[string]interface{}{
			"port":      s.cfg.Port,
			"data_dir":  s.cfg.DataDir,
			"video_dir": s.cfg.VideoDir,
			"cache_dir": s.cfg.CacheDir,
			"debug":     s.cfg.Debug,
		},
	})
}
