package scheduler

import "time"

type Task struct {
	ID                   string
	Command              string
	ScheduledAt          time.Time
	PickedAt             *time.Time
	StartedAt            *time.Time
	CompletedAt          *time.Time
	FailedAt             *time.Time
	RetryCount           int
	MaxRetries           int
	NextAttemptAt        *time.Time
	DeadLetterAt         *time.Time
	NeedsReviewAt        *time.Time
	DispatchAttemptedAt  *time.Time
	ClaimRenewedAt       *time.Time
}

type CommandRequest struct {
	Command     string `json:"command"`
	ScheduledAt string `json:"scheduled_at"`
}
