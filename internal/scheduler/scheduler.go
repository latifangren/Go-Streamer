package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go-streamer/internal/domain"
)

// StreamSupervisor mendefinisikan interface pengontrol lifecycle proses stream slot.
type StreamSupervisor interface {
	StartSlot(slot *domain.StreamSlot, inputSource string) error
	StopSlot(slotNumber int) error
}

// ScheduleProvider mendefinisikan interface opsional untuk persistensi data jadwal.
type ScheduleProvider interface {
	GetEnabledSchedules(ctx context.Context) ([]*domain.Schedule, error)
	GetScheduleByID(ctx context.Context, id string) (*domain.Schedule, error)
	UpdateScheduleRunTimes(ctx context.Context, id string, lastRun, nextRun time.Time) error
}

// Scheduler membungkus in-memory cron runner untuk manajemen jadwal streaming.
type Scheduler struct {
	cron       *cron.Cron
	supervisor StreamSupervisor
	cacheDir   string
	entries    map[string]cron.EntryID
	stopTimers map[string]*time.Timer
	mu         sync.RWMutex
	running    bool
}

// NewScheduler membuat instance baru Scheduler.
func NewScheduler(supervisor StreamSupervisor, cacheDir string) *Scheduler {
	return &Scheduler{
		cron:       cron.New(),
		supervisor: supervisor,
		cacheDir:   cacheDir,
		entries:    make(map[string]cron.EntryID),
		stopTimers: make(map[string]*time.Timer),
	}
}

// Start mengaktifkan cron runner dan mendaftarkan background cleanup berkala.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true

	// Tambahkan background job otomatis setiap 6 jam membersihkan cache snapshot > 24 jam
	if s.cacheDir != "" {
		_, _ = s.cron.AddFunc("0 */6 * * *", func() {
			_, _ = CleanExpiredCache(s.cacheDir, 24*time.Hour)
		})
	}

	s.cron.Start()
	s.mu.Unlock()
}

// Stop menghentikan cron runner dan membatalkan timer stop yang masih aktif.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false

	ctx := s.cron.Stop()

	// Batalkan semua duration timer yang sedang berjalan
	for id, timer := range s.stopTimers {
		timer.Stop()
		delete(s.stopTimers, id)
	}
	s.mu.Unlock()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
	}
}

// AddStreamJob menambahkan task cron untuk jadwal streaming tertentu.
// Ketika cron terpicu, supervisor.StartSlot() akan dipanggil.
// Jika schedule.DurationMinutes > 0, supervisor.StopSlot() dijadwalkan via time.AfterFunc.
func (s *Scheduler) AddStreamJob(schedule *domain.Schedule, slot *domain.StreamSlot, inputSource string) error {
	if schedule == nil {
		return errors.New("schedule cannot be nil")
	}
	if slot == nil {
		return errors.New("slot cannot be nil")
	}
	if schedule.ID == "" {
		return errors.New("schedule ID cannot be empty")
	}
	if schedule.CronExpr == "" {
		return errors.New("cron expression cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Jika job untuk schedule ID ini sudah ada, hapus entri lama
	if entryID, exists := s.entries[schedule.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, schedule.ID)
	}
	if oldTimer, exists := s.stopTimers[schedule.ID]; exists {
		oldTimer.Stop()
		delete(s.stopTimers, schedule.ID)
	}

	jobFunc := func() {
		// Jalankan streaming
		if err := s.supervisor.StartSlot(slot, inputSource); err != nil {
			return
		}

		// Jika ada durasi batas siaran, jadwalkan penghentian otomatis
		if schedule.DurationMinutes > 0 {
			dur := time.Duration(schedule.DurationMinutes) * time.Minute
			s.mu.Lock()
			if prevTimer, ok := s.stopTimers[schedule.ID]; ok {
				prevTimer.Stop()
			}
			s.stopTimers[schedule.ID] = time.AfterFunc(dur, func() {
				_ = s.supervisor.StopSlot(slot.SlotNumber)
				s.mu.Lock()
				delete(s.stopTimers, schedule.ID)
				s.mu.Unlock()
			})
			s.mu.Unlock()
		}
	}

	entryID, err := s.cron.AddFunc(schedule.CronExpr, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to register cron job: %w", err)
	}

	s.entries[schedule.ID] = entryID
	return nil
}

// RemoveStreamJob menghapus task cron streaming berdasarkan scheduleID.
func (s *Scheduler) RemoveStreamJob(scheduleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.entries[scheduleID]
	if !exists {
		return fmt.Errorf("schedule job %s not found", scheduleID)
	}

	s.cron.Remove(entryID)
	delete(s.entries, scheduleID)

	if timer, ok := s.stopTimers[scheduleID]; ok {
		timer.Stop()
		delete(s.stopTimers, scheduleID)
	}

	return nil
}

// HasJob mengecek apakah scheduleID terdaftar di cron runner.
func (s *Scheduler) HasJob(scheduleID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.entries[scheduleID]
	return exists
}

// ActiveJobCount mengembalikan jumlah job streaming yang sedang aktif.
func (s *Scheduler) ActiveJobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// CleanExpiredCache membersihkan file snapshot thumbnail (.jpg / .jpeg) di cacheDir
// yang berusia lebih tua dari maxAge. Mengembalikan jumlah file yang dibersihkan.
func CleanExpiredCache(cacheDir string, maxAge time.Duration) (int, error) {
	if cacheDir == "" {
		return 0, nil
	}

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return 0, nil
	}

	cleanedCount := 0
	cutoff := time.Now().Add(-maxAge)

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jpg" || ext == ".jpeg" {
			if info.ModTime().Before(cutoff) {
				if removeErr := os.Remove(path); removeErr == nil {
					cleanedCount++
				}
			}
		}
		return nil
	})

	return cleanedCount, err
}

// ScheduleDatabaseVacuum mendaftarkan cron job berkala (setiap 12 jam) untuk pemeliharaan SQLite:
// menjalankan PRAGMA wal_checkpoint(TRUNCATE) dan PRAGMA incremental_vacuum agar ukuran file database tetap terkelola dengan baik.
func (s *Scheduler) ScheduleDatabaseVacuum(db *sql.DB) (cron.EntryID, error) {
	if db == nil {
		return 0, errors.New("db cannot be nil")
	}

	// Jalankan langsung sekali saat startup
	RunDatabaseMaintenance(db)

	// Jalankan setiap 12 jam ("0 */12 * * *")
	entryID, err := s.cron.AddFunc("0 */12 * * *", func() {
		RunDatabaseMaintenance(db)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to schedule database vacuum: %w", err)
	}

	return entryID, nil
}

// ScheduleDatabaseVacuum adalah helper package-level untuk mengeksekusi pemeliharaan database SQLite.
func ScheduleDatabaseVacuum(db *sql.DB) error {
	if db == nil {
		return errors.New("db cannot be nil")
	}
	RunDatabaseMaintenance(db)
	return nil
}

// RunDatabaseMaintenance mengeksekusi PRAGMA wal_checkpoint(TRUNCATE) dan PRAGMA incremental_vacuum pada database SQLite.
func RunDatabaseMaintenance(db *sql.DB) {
	if db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE);"); err != nil {
		log.Printf("[scheduler] WAL checkpoint error: %v", err)
	}

	if _, err := db.ExecContext(ctx, "PRAGMA incremental_vacuum;"); err != nil {
		log.Printf("[scheduler] Incremental vacuum error: %v", err)
	}
}
