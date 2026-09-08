package domain

import "time"

const (
	PlayStatusPlaying = "playing"
	PlayStatusNext    = "next"
	PlayStatusQueued  = "queued"
)

type PlaylistItem struct {
	ID         string    `json:"id"`
	SlotID     int64     `json:"slot_id"`
	VideoID    string    `json:"video_id"`
	OrderIndex int       `json:"order_index"`
	PlayStatus string    `json:"play_status"`
	CreatedAt  time.Time `json:"created_at"`

	// Joined relation
	Video *Video `json:"video,omitempty"`
}
