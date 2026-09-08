package domain

import "time"

const (
	OverlapYieldPriority = "yield_priority"
	OverlapTerminatePrev = "terminate_prev"
	OverlapDenyNew       = "deny_new"
)

type Schedule struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	SlotID             int64      `json:"slot_id"`
	CronExpr           string     `json:"cron_expr"`
	DurationMinutes    int        `json:"duration_minutes"`
	IsEnabled          bool       `json:"is_enabled"`
	OverlapGuardPolicy string     `json:"overlap_guard_policy"`
	LastRunAt          *time.Time `json:"last_run_at,omitempty"`
	NextRunAt          *time.Time `json:"next_run_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`

	// Joined slot
	Slot *StreamSlot `json:"slot,omitempty"`
}
