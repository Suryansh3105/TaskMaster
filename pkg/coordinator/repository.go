package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ClaimDueTasks(ctx context.Context, limit int) ([]ClaimedTask, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	rows, err := tx.Query(ctx,
		`SELECT id, command FROM tasks
     WHERE scheduled_at <= NOW() AND picked_at IS NULL
     AND (next_attempt_at IS NULL OR next_attempt_at <= NOW())
     ORDER BY scheduled_at
     LIMIT $1
     FOR UPDATE SKIP LOCKED`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query due tasks: %w", err)
	}

	var claimed []ClaimedTask
	var ids []string
	for rows.Next() {
		var t ClaimedTask
		if err := rows.Scan(&t.ID, &t.Command); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		claimed = append(claimed, t)
		ids = append(ids, t.ID)
	}
	rows.Close()

	if len(ids) == 0 {
		return nil, tx.Commit(ctx) // nothing to claim, commit the empty tx
	}

	_, err = tx.Exec(ctx,
		`UPDATE tasks SET picked_at = NOW(), claim_renewed_at = NOW()
		 WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark tasks claimed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit claim transaction: %w", err)
	}

	return claimed, nil
}

func (r *Repository) MarkDispatchAttempted(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET dispatch_attempted_at = NOW() WHERE id = $1`, taskID)
	if err != nil {
		return fmt.Errorf("failed to mark dispatch attempted: %w", err)
	}
	return nil
}

// MarkStarted is called only after AssignTask returns successfully.
func (r *Repository) MarkStarted(ctx context.Context, taskID, workerID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET started_at = NOW(), worker_id = $2 WHERE id = $1`,
		taskID, workerID)
	if err != nil {
		return fmt.Errorf("failed to mark started: %w", err)
	}
	return nil
}

func (r *Repository) MarkCompleted(ctx context.Context, taskID string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tasks SET completed_at = NOW() WHERE id = $1 AND completed_at IS NULL`,
		taskID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to mark completed: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) RecordFailureAndScheduleRetry(ctx context.Context, taskID string) error {
	var retryCount, maxRetries int
	err := r.pool.QueryRow(ctx,
		`SELECT retry_count, max_retries FROM tasks WHERE id = $1`, taskID,
	).Scan(&retryCount, &maxRetries)
	if err != nil {
		return fmt.Errorf("failed to read retry state for task %s: %w", taskID, err)
	}

	newCount := retryCount + 1

	if newCount >= maxRetries {
		_, err = r.pool.Exec(ctx,
			`UPDATE tasks SET retry_count = $1, dead_letter_at = NOW(), failed_at = NOW()
			 WHERE id = $2`,
			newCount, taskID,
		)
		if err != nil {
			return fmt.Errorf("failed to dead-letter task %s: %w", taskID, err)
		}
		return nil
	}

	delay := NextAttemptDelay(newCount)
	_, err = r.pool.Exec(ctx,
		`UPDATE tasks
		 SET retry_count = $1, failed_at = NOW(), picked_at = NULL,
		     started_at = NULL, dispatch_attempted_at = NULL,
		     next_attempt_at = NOW() + $2 * INTERVAL '1 second'
		 WHERE id = $3`,
		newCount, delay.Seconds(), taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule retry for task %s: %w", taskID, err)
	}
	return nil
}

func (r *Repository) RenewClaim(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET claim_renewed_at = NOW() WHERE id = $1`, taskID)
	if err != nil {
		return fmt.Errorf("failed to renew claim for task %s: %w", taskID, err)
	}
	return nil
}

func (r *Repository) FindStaleClaims(ctx context.Context, leaseTimeout time.Duration) ([]ClaimedTask, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, command FROM tasks
		 WHERE picked_at IS NOT NULL
		   AND completed_at IS NULL
		   AND dead_letter_at IS NULL
		   AND needs_review_at IS NULL
		   AND claim_renewed_at < NOW() - $1 * INTERVAL '1 second'`,
		leaseTimeout.Seconds(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query stale claims: %w", err)
	}
	defer rows.Close()

	var tasks []ClaimedTask
	for rows.Next() {
		var t ClaimedTask
		if err := rows.Scan(&t.ID, &t.Command); err != nil {
			return nil, fmt.Errorf("failed to scan stale claim: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *Repository) WasDispatchAttempted(ctx context.Context, taskID string) (bool, error) {
	var attempted *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT dispatch_attempted_at FROM tasks WHERE id = $1`, taskID,
	).Scan(&attempted)
	if err != nil {
		return false, fmt.Errorf("failed to check dispatch_attempted_at: %w", err)
	}
	return attempted != nil, nil
}

func (r *Repository) ClearClaimForRetry(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET picked_at = NULL, started_at = NULL,
		 dispatch_attempted_at = NULL, claim_renewed_at = NULL, worker_id = NULL
		 WHERE id = $1`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to clear claim for retry on task %s: %w", taskID, err)
	}
	return nil
}

func (r *Repository) MarkNeedsReview(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET needs_review_at = NOW() WHERE id = $1 AND needs_review_at IS NULL`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark task %s needs_review: %w", taskID, err)
	}
	return nil
}

func (r *Repository) FindTasksInProgressForWorker(ctx context.Context, workerID string) ([]ClaimedTask, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, command FROM tasks
		 WHERE started_at IS NOT NULL AND completed_at IS NULL
		   AND dead_letter_at IS NULL AND needs_review_at IS NULL
		   AND worker_id = $1`,
		workerID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query in-progress tasks for worker %s: %w", workerID, err)
	}
	defer rows.Close()

	var tasks []ClaimedTask
	for rows.Next() {
		var t ClaimedTask
		if err := rows.Scan(&t.ID, &t.Command); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *Repository) IsNeedsReview(ctx context.Context, taskID string) (bool, error) {
	var needsReviewAt *time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT needs_review_at FROM tasks WHERE id = $1`, taskID,
	).Scan(&needsReviewAt)
	if err != nil {
		return false, fmt.Errorf("failed to check needs_review state: %w", err)
	}
	return needsReviewAt != nil, nil
}

func (r *Repository) RequeueTask(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET picked_at = NULL, started_at = NULL,
		 dispatch_attempted_at = NULL, claim_renewed_at = NULL,
		 worker_id = NULL, needs_review_at = NULL
		 WHERE id = $1`,
		taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to requeue task %s: %w", taskID, err)
	}
	return nil
}
