package scheduler

import "time"

type Task struct {
	ID                  string     `json:"id"`
	Command             string     `json:"command"`
	ScheduledAt         time.Time  `json:"scheduled_at"`
	PickedAt            *time.Time `json:"picked_at"`
	StartedAt           *time.Time `json:"started_at"`
	CompletedAt         *time.Time `json:"completed_at"`
	FailedAt            *time.Time `json:"failed_at"`
	RetryCount          int        `json:"retry_count"`
	MaxRetries          int        `json:"max_retries"`
	NextAttemptAt       *time.Time `json:"next_attempt_at"`
	DeadLetterAt        *time.Time `json:"dead_letter_at"`
	NeedsReviewAt       *time.Time `json:"needs_review_at"`
	DispatchAttemptedAt *time.Time `json:"dispatch_attempted_at"`
	ClaimRenewedAt      *time.Time `json:"claim_renewed_at"`
	WorkerID            *string    `json:"worker_id"`
}

type CommandRequest struct {
	Command     string `json:"command"`
	ScheduledAt string `json:"scheduled_at"`
}
