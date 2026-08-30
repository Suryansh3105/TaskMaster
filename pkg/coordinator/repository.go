package coordinator

import (
	"context"
	"fmt"

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
func (r *Repository) MarkStarted(ctx context.Context, taskID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET started_at = NOW() WHERE id = $1`, taskID)
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
