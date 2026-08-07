package scheduler

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyCommand    = errors.New("command cannot be empty")
	ErrInvalidTime     = errors.New("scheduled_at must be a valid RFC3339 timestamp")
	ErrScheduledInPast = errors.New("scheduled_at cannot be in the past")
)

func ValidateCommandRequest(req CommandRequest) (time.Time, error) {
	if strings.TrimSpace(req.Command) == "" {
		return time.Time{}, ErrEmptyCommand
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return time.Time{}, ErrInvalidTime
	}

	if scheduledAt.Before(time.Now()) {
		return time.Time{}, ErrScheduledInPast
	}

	return scheduledAt, nil
}