package domain

import "time"

const (
	TranscodeQueued     = "queued"
	TranscodeProcessing = "processing"
	TranscodeCompleted  = "completed"
	TranscodeFailed     = "failed"
)

type TranscodeJob struct {
	ID             string    `json:"id"`
	SourceVideoID  string    `json:"source_video_id"`
	TargetFilePath string    `json:"target_file_path"`
	Status         string    `json:"status"`
	ProgressPercent float64  `json:"progress_percent"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Joined source video
	SourceVideo *Video `json:"source_video,omitempty"`
}
