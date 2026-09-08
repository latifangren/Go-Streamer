package domain

import "time"

const (
	SlotStatusIdle     = "idle"
	SlotStatusStarting = "starting"
	SlotStatusRunning  = "running"
	SlotStatusError    = "error"

	SourceTypeSingle   = "single"
	SourceTypePlaylist = "playlist"

	StreamModeCopy      = "copy"
	StreamModeTranscode = "transcode"

	PlatformYouTube  = "youtube"
	PlatformFacebook = "facebook"
	PlatformTwitch   = "twitch"
	PlatformCustom   = "custom"
)

type StreamSlot struct {
	ID                  int64      `json:"id"`
	SlotNumber          int        `json:"slot_number"`
	Name                string     `json:"name"`
	Status              string     `json:"status"`
	SourceType          string     `json:"source_type"`
	VideoID             *string    `json:"video_id,omitempty"`
	TargetPlatform      string     `json:"target_platform"`
	RTMPURL             string     `json:"rtmp_url"`
	StreamKey           string     `json:"-"` // sensitive
	Mode                string     `json:"mode"`
	Quality             string     `json:"quality"`
	Preset              string     `json:"preset"`
	LoopPlayback        bool       `json:"loop_playback"`
	MaxDurationMinutes  int        `json:"max_duration_minutes"`
	AutoRestart         bool       `json:"auto_restart"`
	EnableOverlay       bool       `json:"enable_overlay"`
	OverlayClockWIB     bool       `json:"overlay_clock_wib"`
	OverlayWatermark    string     `json:"overlay_watermark"`
	OverlayWatermarkPos string     `json:"overlay_watermark_pos"`
	LastPTSSyncMS       float64    `json:"last_pts_sync_ms"`
	LastKeyframeGOPS    float64    `json:"last_keyframe_gop_s"`
	LastNetLatencyMS    int        `json:"last_net_latency_ms"`
	LastSnapshotPath    string     `json:"last_snapshot_path"`
	LastSnapshotAt      *time.Time `json:"last_snapshot_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// Loaded relations (optional)
	Video    *Video          `json:"video,omitempty"`
	Playlist []*PlaylistItem `json:"playlist,omitempty"`
}
