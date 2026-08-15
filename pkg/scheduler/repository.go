package scheduler

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

func (r *Repository) InsertTask(ctx context.Context, command string, scheduledAt time.Time) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO tasks (command, scheduled_at) VALUES ($1, $2) RETURNING id`,
		command, scheduledAt,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to insert task: %w", err)
	}
	return id, nil
}

func (r *Repository) GetTask(ctx context.Context, id string) (*Task, error) {
	var t Task
	err := r.pool.QueryRow(ctx,
		`SELECT id, command, scheduled_at, picked_at, started_at, completed_at, failed_at
		 FROM tasks WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Command, &t.ScheduledAt, &t.PickedAt, &t.StartedAt, &t.CompletedAt, &t.FailedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &t, nil
}
