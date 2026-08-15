package scheduler

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/common"
)

func setupTestRepo(t *testing.T) *Repository {
	t.Helper()
	ctx := context.Background()
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@localhost:5432/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)
	pool, err := common.ConnectToDatabase(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return NewRepository(pool)
}

func TestInsertAndGetTask(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	scheduledAt := time.Now().Add(1 * time.Hour).Truncate(time.Second)

	id, err := repo.InsertTask(ctx, "echo hello", scheduledAt)
	if err != nil {
		t.Fatalf("InsertTask failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected a non-empty task ID")
	}

	task, err := repo.GetTask(ctx, id)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.Command != "echo hello" {
		t.Errorf("expected command %q, got %q", "echo hello", task.Command)
	}
	if !task.ScheduledAt.Equal(scheduledAt) {
		t.Errorf("expected scheduled_at %v, got %v", scheduledAt, task.ScheduledAt)
	}
	if task.PickedAt != nil {
		t.Errorf("expected PickedAt to be nil, got %v", task.PickedAt)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetTask(ctx, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected an error for a nonexistent task, got nil")
	}
}
