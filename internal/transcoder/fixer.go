package transcoder

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-streamer/internal/domain"
)

// Regex untuk mendeteksi 'time=HH:MM:SS.ms' pada output progress ffmpeg
var timeRegex = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d+)`)

type TranscoderRepo interface {
	CreateJob(ctx context.Context, job *domain.TranscodeJob) error
	GetJobByID(ctx context.Context, id string) (*domain.TranscodeJob, error)
	UpdateJob(ctx context.Context, job *domain.TranscodeJob) error
	ListJobs(ctx context.Context) ([]*domain.TranscodeJob, error)
	GetActiveOrPendingJob(ctx context.Context) (*domain.TranscodeJob, error)
}

type VideoRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Video, error)
	Update(ctx context.Context, v *domain.Video) error
}

type CodecFixer struct {
	ffmpegPath  string
	jobRepo     TranscoderRepo
	videoRepo   VideoRepo
	queueChan   chan string // menyalurkan job ID
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.Mutex
	activeJobID string
	activeCmd   *exec.Cmd
}

func NewCodecFixer(ffmpegPath string, jobRepo TranscoderRepo, videoRepo VideoRepo) *CodecFixer {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &CodecFixer{
		ffmpegPath: ffmpegPath,
		jobRepo:    jobRepo,
		videoRepo:  videoRepo,
		queueChan:  make(chan string, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start menjalankan background worker dengan single concurrency.
func (f *CodecFixer) Start() {
	f.wg.Add(1)
	go f.workerLoop()
}

// Stop menghentikan worker secara graceful dan membatalkan proses ffmpeg yang sedang aktif.
func (f *CodecFixer) Stop() {
	f.cancel()

	f.mu.Lock()
	if f.activeCmd != nil && f.activeCmd.Process != nil {
		_ = f.activeCmd.Process.Kill()
	}
	f.mu.Unlock()

	f.wg.Wait()
}

// Enqueue memasukkan job ID ke antrean worker.
func (f *CodecFixer) Enqueue(jobID string) {
	select {
	case f.queueChan <- jobID:
	case <-f.ctx.Done():
	}
}

// workerLoop menjalankan pemrosesan antrean satu demi satu (single concurrency).
func (f *CodecFixer) workerLoop() {
	defer f.wg.Done()

	// Cek apakah ada job yang pending atau processing dari run sebelumnya
	if f.jobRepo != nil {
		if pending, err := f.jobRepo.GetActiveOrPendingJob(f.ctx); err == nil && pending != nil {
			f.processJob(pending.ID)
		}
	}

	for {
		select {
		case <-f.ctx.Done():
			return
		case jobID := <-f.queueChan:
			if jobID != "" {
				f.processJob(jobID)
			}
		}
	}
}

func (f *CodecFixer) processJob(jobID string) {
	ctx := f.ctx
	job, err := f.jobRepo.GetJobByID(ctx, jobID)
	if err != nil || job == nil {
		return
	}

	// Hanya proses jika status queued atau processing
	if job.Status != domain.TranscodeQueued && job.Status != domain.TranscodeProcessing {
		return
	}

	f.mu.Lock()
	f.activeJobID = jobID
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		f.activeJobID = ""
		f.activeCmd = nil
		f.mu.Unlock()
	}()

	// Update status ke processing
	job.Status = domain.TranscodeProcessing
	_ = f.jobRepo.UpdateJob(ctx, job)

	sourceVideo := job.SourceVideo
	if sourceVideo == nil && f.videoRepo != nil {
		sourceVideo, _ = f.videoRepo.GetByID(ctx, job.SourceVideoID)
	}

	if sourceVideo == nil {
		f.failJob(ctx, job, "source video not found")
		return
	}

	sourcePath := sourceVideo.FilePath
	if _, err := os.Stat(sourcePath); err != nil {
		f.failJob(ctx, job, fmt.Sprintf("source video file not found: %s", sourcePath))
		return
	}

	targetPath := job.TargetFilePath
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		f.failJob(ctx, job, fmt.Sprintf("failed to create target directory: %v", err))
		return
	}

	// Args FFmpeg standardisasi ke passthrough-ready
	args := []string{
		"-i", sourcePath,
		"-c:v", "libx264",
		"-profile:v", "high",
		"-level:v", "4.1",
		"-preset", "veryfast",
		"-b:v", "2500k",
		"-maxrate", "2500k",
		"-bufsize", "5000k",
		"-g", "60",
		"-keyint_min", "60",
		"-sc_threshold", "0",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		"-y",
		targetPath,
	}

	cmd := exec.CommandContext(ctx, f.ffmpegPath, args...)
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		f.failJob(ctx, job, fmt.Sprintf("failed to get stderr pipe: %v", err))
		return
	}

	f.mu.Lock()
	f.activeCmd = cmd
	f.mu.Unlock()

	if err := cmd.Start(); err != nil {
		f.failJob(ctx, job, fmt.Sprintf("failed to start ffmpeg: %v", err))
		return
	}

	// Baca progress dari stderr
	totalDuration := sourceVideo.DurationSeconds
	var stderrLog strings.Builder
	doneRead := make(chan struct{})

	go func() {
		defer close(doneRead)
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Split(scanLinesOrCarriageReturns)

		lastUpdate := time.Now()

		for scanner.Scan() {
			line := scanner.Text()
			if len(stderrLog.String()) < 2048 {
				stderrLog.WriteString(line + "\n")
			}

			sec := ParseFFmpegTime(line)
			if sec > 0 && totalDuration > 0 {
				percent := (sec / totalDuration) * 100.0
				if percent > 99.0 {
					percent = 99.0
				}
				percent = math.Round(percent*10) / 10

				// Throttle DB updates minimal tiap 1 detik atau kenaikan signifikan
				if time.Since(lastUpdate) >= 1*time.Second && percent > job.ProgressPercent {
					job.ProgressPercent = percent
					_ = f.jobRepo.UpdateJob(ctx, job)
					lastUpdate = time.Now()
				}
			}
		}
	}()

	waitErr := cmd.Wait()
	<-doneRead

	if waitErr != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			job.Status = domain.TranscodeFailed
			job.ErrorMessage = "transcode canceled"
			_ = f.jobRepo.UpdateJob(ctx, job)
			return
		}
		errMsg := waitErr.Error()
		if stderrLog.Len() > 0 {
			errMsg = fmt.Sprintf("%s: %s", waitErr, strings.TrimSpace(stderrLog.String()))
		}
		f.failJob(ctx, job, errMsg)
		return
	}

	// Saat selesai: update status job ke completed dan progress 100.0
	job.Status = domain.TranscodeCompleted
	job.ProgressPercent = 100.0
	job.ErrorMessage = ""
	_ = f.jobRepo.UpdateJob(ctx, job)

	// Update record video sumber: IsPassthroughReady = true, VideoCodec = "h264", AudioCodec = "aac", FilePath = targetPath
	if f.videoRepo != nil {
		sourceVideo.IsPassthroughReady = true
		sourceVideo.VideoCodec = "h264"
		sourceVideo.AudioCodec = "aac"
		sourceVideo.FilePath = targetPath
		if fi, err := os.Stat(targetPath); err == nil {
			sourceVideo.FileSize = fi.Size()
		}
		_ = f.videoRepo.Update(ctx, sourceVideo)
	}
}

func (f *CodecFixer) failJob(ctx context.Context, job *domain.TranscodeJob, errMsg string) {
	job.Status = domain.TranscodeFailed
	job.ErrorMessage = errMsg
	_ = f.jobRepo.UpdateJob(ctx, job)
}

// ParseFFmpegTime mengekstrak nilai detik dari string output ffmpeg yang memuat time=HH:MM:SS.ms.
func ParseFFmpegTime(line string) float64 {
	matches := timeRegex.FindStringSubmatch(line)
	if len(matches) < 5 {
		return 0
	}

	h, _ := strconv.ParseFloat(matches[1], 64)
	m, _ := strconv.ParseFloat(matches[2], 64)
	s, _ := strconv.ParseFloat(matches[3], 64)
	msStr := matches[4]
	ms, _ := strconv.ParseFloat(msStr, 64)
	msFrac := ms / math.Pow10(len(msStr))

	return (h * 3600) + (m * 60) + s + msFrac
}

// scanLinesOrCarriageReturns memecah stream berdasarkan newline atau carriage return (\r)
// yang lazim digunakan ffmpeg untuk mengupdate progress bar di stderr.
func scanLinesOrCarriageReturns(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' || data[i] == '\n' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
