package domain

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidInput      = errors.New("invalid input data")
	ErrSlotBusy          = errors.New("stream slot is already running")
	ErrSlotNotRunning    = errors.New("stream slot is not running")
	ErrVideoNotCompliant = errors.New("video file is not compliant with passthrough streaming")
	ErrThermalThrottled  = errors.New("device temperature exceeded safe threshold")
	ErrTunnelInactive    = errors.New("tunnel service is not active")
)
