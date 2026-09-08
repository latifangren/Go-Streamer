package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-streamer/internal/domain"
)

// ActiveSlot merepresentasikan instans proses streaming yang sedang berjalan pada slot tertentu.
type ActiveSlot struct {
	SlotNumber     int
	Slot           *domain.StreamSlot
	InputSource    string
	EffectiveInput string
	Cmd            *exec.Cmd
	Status         string
	LastTelemetry  *domain.StreamTelemetry
	CancelFunc     context.CancelFunc
	DoneChan       chan struct{}
	Stopping       bool
	RestartCount   int
}

// Supervisor mengelola siklus hidup proses FFmpeg streaming pada setiap slot.
type Supervisor struct {
	mu               sync.RWMutex
	ffmpegPath       string
	cacheDir         string
	slots            map[int]*ActiveSlot
	telemetryChan    chan *domain.StreamTelemetry
	snapshotInterval time.Duration
	onCrash          func(slotNumber int, slotName, errMsg string)
}

// NewSupervisor menginisialisasi instans Supervisor baru.
func NewSupervisor(ffmpegPath, cacheDir string) *Supervisor {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}
	_ = os.MkdirAll(cacheDir, 0755)

	return &Supervisor{
		ffmpegPath:       ffmpegPath,
		cacheDir:         cacheDir,
		slots:            make(map[int]*ActiveSlot),
		telemetryChan:    make(chan *domain.StreamTelemetry, 100),
		snapshotInterval: 30 * time.Second,
	}
}

// TelemetryChan mengembalikan receive-only channel untuk mendengarkan telemetri streaming real-time.
func (s *Supervisor) TelemetryChan() <-chan *domain.StreamTelemetry {
	return s.telemetryChan
}

// SetOnCrashCallback mendaftarkan fungsi callback yang dipanggil saat proses FFmpeg keluar dengan status error/crash.
func (s *Supervisor) SetOnCrashCallback(cb func(slotNumber int, slotName, errMsg string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCrash = cb
}

// StartSlot memulai streaming FFmpeg untuk slot tertentu.
func (s *Supervisor) StartSlot(slot *domain.StreamSlot, inputSource string) error {
	if slot == nil {
		return fmt.Errorf("slot cannot be nil: %w", domain.ErrInvalidInput)
	}

	inputSource = strings.TrimSpace(inputSource)
	if inputSource == "" {
		return fmt.Errorf("input source cannot be empty: %w", domain.ErrInvalidInput)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	active, exists := s.slots[slot.SlotNumber]
	if exists && (active.Status == domain.SlotStatusRunning || active.Status == domain.SlotStatusStarting) {
		return domain.ErrSlotBusy
	}

	// 1. Tangani Concat Manifest jika tipe sumber adalah Playlist
	effectiveInput := inputSource
	if slot.SourceType == domain.SourceTypePlaylist {
		if strings.HasSuffix(strings.ToLower(inputSource), ".txt") {
			effectiveInput = inputSource
		} else {
			var videoPaths []string
			if len(slot.Playlist) > 0 {
				for _, item := range slot.Playlist {
					if item != nil && item.Video != nil && item.Video.FilePath != "" {
						videoPaths = append(videoPaths, item.Video.FilePath)
					}
				}
			}
			if len(videoPaths) == 0 && inputSource != "" {
				for _, p := range strings.Split(inputSource, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						videoPaths = append(videoPaths, p)
					}
				}
			}
			if len(videoPaths) > 0 {
				manifest, err := GenerateConcatManifest(slot.ID, s.cacheDir, videoPaths)
				if err != nil {
					return fmt.Errorf("failed to generate concat manifest: %w", err)
				}
				effectiveInput = manifest
			}
		}
	}

	// 2. Bangun argumen baris perintah FFmpeg
	args, err := BuildFFmpegArgs(slot, effectiveInput)
	if err != nil {
		return fmt.Errorf("failed to build ffmpeg args: %w", err)
	}

	// 3. Siapkan context dan proses eksekusi
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	SetupProcessGroup(cmd)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start ffmpeg process: %w", err)
	}

	activeSlot := &ActiveSlot{
		SlotNumber:     slot.SlotNumber,
		Slot:           slot,
		InputSource:    inputSource,
		EffectiveInput: effectiveInput,
		Cmd:            cmd,
		Status:         domain.SlotStatusRunning,
		CancelFunc:     cancel,
		DoneChan:       make(chan struct{}),
		Stopping:       false,
		RestartCount:   0,
	}
	s.slots[slot.SlotNumber] = activeSlot

	go s.superviseSlot(activeSlot, stderrPipe, ctx)

	return nil
}

// StopSlot menghentikan proses streaming pada slot tertentu secara anggun (graceful).
func (s *Supervisor) StopSlot(slotNumber int) error {
	s.mu.Lock()
	activeSlot, exists := s.slots[slotNumber]
	if !exists || (activeSlot.Status != domain.SlotStatusRunning && activeSlot.Status != domain.SlotStatusStarting) {
		s.mu.Unlock()
		return domain.ErrSlotNotRunning
	}

	activeSlot.Stopping = true
	cmd := activeSlot.Cmd
	cancel := activeSlot.CancelFunc
	doneChan := activeSlot.DoneChan
	s.mu.Unlock()

	// Batalkan context untuk menghentikan goroutine snapshot dan backoff ticker
	if cancel != nil {
		cancel()
	}

	// 1. Graceful stop: kirim sinyal SIGINT
	if cmd != nil {
		_ = KillProcessGroup(cmd)
	}

	// 2. Tunggu proses berhenti hingga 5 detik
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-doneChan:
		// Proses berhenti secara graceful
	case <-timer.C:
		// Jika masih berjalan setelah 5 detik, paksa berhenti (SIGKILL)
		if cmd != nil {
			_ = ForceKillProcessGroup(cmd)
		}
		select {
		case <-doneChan:
		case <-time.After(1 * time.Second):
		}
	}

	s.mu.Lock()
	activeSlot.Status = domain.SlotStatusIdle
	s.mu.Unlock()

	return nil
}

// StopAll menghentikan semua slot streaming yang sedang aktif.
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	var runningSlots []int
	for slotNum, active := range s.slots {
		if active.Status == domain.SlotStatusRunning || active.Status == domain.SlotStatusStarting {
			runningSlots = append(runningSlots, slotNum)
		}
	}
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, slotNum := range runningSlots {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			_ = s.StopSlot(num)
		}(slotNum)
	}
	wg.Wait()
}

// GetSlotStatus mengembalikan status terkini dari slot yang ditentukan.
func (s *Supervisor) GetSlotStatus(slotNumber int) (status string, telemetry *domain.StreamTelemetry, running bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active, exists := s.slots[slotNumber]
	if !exists {
		return domain.SlotStatusIdle, nil, false
	}

	isRunning := active.Status == domain.SlotStatusRunning || active.Status == domain.SlotStatusStarting
	return active.Status, active.LastTelemetry, isRunning
}

// superviseSlot mengawasi output stderr, eksekusi snapshot rutin, dan pemulihan crash proses FFmpeg.
func (s *Supervisor) superviseSlot(activeSlot *ActiveSlot, stderr io.ReadCloser, ctx context.Context) {
	// Goroutine 1: Pembaca stderr line by line untuk telemetri
	go func() {
		defer stderr.Close()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			tel := ParseStderrLine(line, activeSlot.SlotNumber)
			if tel != nil {
				s.mu.Lock()
				activeSlot.LastTelemetry = tel
				s.mu.Unlock()

				select {
				case s.telemetryChan <- tel:
				default:
					// Drop jika antrean telemetri penuh
				}
			}
		}
	}()

	// Goroutine 2: Pengambil cuplikan snapshot rutin (default interval 30 detik)
	go func() {
		interval := s.snapshotInterval
		if interval <= 0 {
			interval = 30 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		snapshotPath := filepath.Join(s.cacheDir, fmt.Sprintf("snapshot_slot_%d.jpg", activeSlot.SlotNumber))

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = ExtractSnapshot(ctx, s.ffmpegPath, activeSlot.EffectiveInput, snapshotPath)
				s.mu.Lock()
				now := time.Now().UTC()
				activeSlot.Slot.LastSnapshotPath = snapshotPath
				activeSlot.Slot.LastSnapshotAt = &now
				s.mu.Unlock()
			}
		}
	}()

	// Menunggu proses FFmpeg selesai
	waitErr := activeSlot.Cmd.Wait()

	close(activeSlot.DoneChan)

	s.mu.Lock()
	if activeSlot.Stopping {
		activeSlot.Status = domain.SlotStatusIdle
		s.mu.Unlock()
		return
	}

	// Panggil callback onCrash jika FFmpeg exit dengan error / crash
	if waitErr != nil && s.onCrash != nil {
		slotNum := activeSlot.SlotNumber
		slotName := ""
		if activeSlot.Slot != nil {
			slotName = activeSlot.Slot.Name
		}
		cb := s.onCrash
		go cb(slotNum, slotName, waitErr.Error())
	}

	// Tangani AutoRestart jika proses crash atau terhenti secara mendadak
	if activeSlot.Slot.AutoRestart && activeSlot.RestartCount < 5 {
		activeSlot.RestartCount++
		activeSlot.Status = domain.SlotStatusStarting
		currentRestart := activeSlot.RestartCount
		slotData := activeSlot.Slot
		inputSource := activeSlot.InputSource
		effectiveInput := activeSlot.EffectiveInput
		s.mu.Unlock()

		log.Printf("[runner] Slot %d exited (%v), auto-restarting (attempt %d/5)...", activeSlot.SlotNumber, waitErr, currentRestart)

		// Exponential backoff: 1s, 2s, 4s, 8s, 16s
		backoff := time.Duration(1<<uint(currentRestart-1)) * time.Second
		if backoff > 16*time.Second {
			backoff = 16 * time.Second
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		s.mu.Lock()
		if activeSlot.Stopping {
			activeSlot.Status = domain.SlotStatusIdle
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		s.restartSlot(activeSlot, slotData, inputSource, effectiveInput, currentRestart)
		return
	}

	if waitErr != nil {
		activeSlot.Status = domain.SlotStatusError
	} else {
		activeSlot.Status = domain.SlotStatusIdle
	}
	s.mu.Unlock()
}

// restartSlot mengeksekusi ulang proses FFmpeg setelah terjadinya crash dengan exponential backoff.
func (s *Supervisor) restartSlot(activeSlot *ActiveSlot, slot *domain.StreamSlot, inputSource, effectiveInput string, restartCount int) {
	args, err := BuildFFmpegArgs(slot, effectiveInput)
	if err != nil {
		s.mu.Lock()
		activeSlot.Status = domain.SlotStatusError
		s.mu.Unlock()
		return
	}

	newCtx, newCancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(newCtx, s.ffmpegPath, args...)
	SetupProcessGroup(cmd)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		newCancel()
		s.mu.Lock()
		activeSlot.Status = domain.SlotStatusError
		s.mu.Unlock()
		return
	}

	if err := cmd.Start(); err != nil {
		newCancel()
		s.mu.Lock()
		activeSlot.Status = domain.SlotStatusError
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	if activeSlot.Stopping {
		s.mu.Unlock()
		newCancel()
		_ = ForceKillProcessGroup(cmd)
		return
	}

	activeSlot.Cmd = cmd
	activeSlot.CancelFunc = newCancel
	activeSlot.DoneChan = make(chan struct{})
	activeSlot.Status = domain.SlotStatusRunning
	activeSlot.RestartCount = restartCount
	s.mu.Unlock()

	go s.superviseSlot(activeSlot, stderrPipe, newCtx)
}
