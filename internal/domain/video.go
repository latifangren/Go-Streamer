package domain

import (
	"context"
	"time"
)

type Video struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	Filename           string    `json:"filename"`
	OriginalName       string    `json:"original_name"`
	FilePath           string    `json:"file_path"`
	FileSize           int64     `json:"file_size"`
	DurationSeconds    float64   `json:"duration_seconds"`
	Resolution         string    `json:"resolution"`
	VideoCodec         string    `json:"video_codec"`
	AudioCodec         string    `json:"audio_codec"`
	FPS                float64   `json:"fps"`
	GOPSize            float64   `json:"gop_size"`
	IsPassthroughReady bool      `json:"is_passthrough_ready"`
	CreatedAt          time.Time `json:"created_at"`
}

type VideoRepository interface {
	Create(ctx context.Context, video *Video) error
	GetByID(ctx context.Context, id string) (*Video, error)
	GetByUserID(ctx context.Context, userID string) ([]*Video, error)
	ListAll(ctx context.Context) ([]*Video, error)
	Update(ctx context.Context, video *Video) error
	Delete(ctx context.Context, id string) error
	GetTotalStorageBytes(ctx context.Context) (int64, error)
}
