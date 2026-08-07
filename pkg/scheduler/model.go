package scheduler

import "time"

type Taak struct {
	ID          string
	Command     string
	ScheduledAt time.Time
	PickedAt    *time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	FailedAt    *time.Time
}

type CommandRequest struct {
	Command     string `json:"command"`
	ScheduledAt string `json:"scheduled_at"`
}
