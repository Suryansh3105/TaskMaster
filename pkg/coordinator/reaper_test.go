package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/common"
	"github.com/Suryansh3105/taskmaster/pkg/scheduler"
	"github.com/jackc/pgx/v5/pgxpool"
)

func cleanTasksTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(context.Background(), "TRUNCATE TABLE tasks")
	if err != nil {
		t.Fatalf("failed to clean tasks table: %v", err)
	}
}

func TestReaper_SafeAutoRetry_WhenDispatchNeverAttempted(t *testing.T) {
	repo := setupCoordinatorTestRepo(t)
	reaper := NewReaper(repo, NewRegistry())

	connString, _ := common.GetDBConnectionString()
	pool, _ := common.ConnectToDatabase(context.Background(), connString)
	defer pool.Close()

	cleanTasksTable(t, pool)

	schedRepo := scheduler.NewRepository(pool)

	taskID, _ := schedRepo.InsertTask(context.Background(), "echo hi", time.Now().Add(-1*time.Minute))
	claimed, _ := repo.ClaimDueTasks(context.Background(), 1)
	if len(claimed) != 1 {
		t.Fatalf("expected to claim 1 task")
	}

	pool.Exec(context.Background(),
		`UPDATE tasks SET claim_renewed_at = NOW() - INTERVAL '60 seconds' WHERE id = $1`, taskID)

	reaper.checkStaleClaims(context.Background())

	task, _ := schedRepo.GetTask(context.Background(), taskID)
	if task.PickedAt != nil {
		t.Error("expected claim to be cleared for auto-retry")
	}
	if task.NeedsReviewAt != nil {
		t.Error("expected NOT needs_review — dispatch was never attempted")
	}
}

func TestReaper_NeedsReview_WhenDispatchWasInFlight(t *testing.T) {
	repo := setupCoordinatorTestRepo(t)
	reaper := NewReaper(repo, NewRegistry())

	connString, _ := common.GetDBConnectionString()
	pool, _ := common.ConnectToDatabase(context.Background(), connString)
	defer pool.Close()

	cleanTasksTable(t, pool)

	schedRepo := scheduler.NewRepository(pool)

	taskID, _ := schedRepo.InsertTask(context.Background(), "echo hi", time.Now().Add(-1*time.Minute))
	repo.ClaimDueTasks(context.Background(), 1)

	pool.Exec(context.Background(),
		`UPDATE tasks SET dispatch_attempted_at = NOW(),
		 claim_renewed_at = NOW() - INTERVAL '60 seconds' WHERE id = $1`, taskID)

	reaper.checkStaleClaims(context.Background())

	task, _ := schedRepo.GetTask(context.Background(), taskID)
	if task.NeedsReviewAt == nil {
		t.Error("expected needs_review — dispatch was in flight")
	}
}

func TestReaper_StaleWorker_OnlyAffectsItsOwnTasks(t *testing.T) {
	repo := setupCoordinatorTestRepo(t)
	registry := NewRegistry()
	reaper := NewReaper(repo, registry)

	connString, _ := common.GetDBConnectionString()
	pool, _ := common.ConnectToDatabase(context.Background(), connString)
	defer pool.Close()

	cleanTasksTable(t, pool)

	schedRepo := scheduler.NewRepository(pool)

	// Two tasks, dispatched to two different workers.
	taskA, _ := schedRepo.InsertTask(context.Background(), "task-a", time.Now().Add(-1*time.Minute))
	taskB, _ := schedRepo.InsertTask(context.Background(), "task-b", time.Now().Add(-1*time.Minute))
	pool.Exec(context.Background(),
		`UPDATE tasks SET started_at = NOW(), worker_id = 'worker-stale' WHERE id = $1`, taskA)
	pool.Exec(context.Background(),
		`UPDATE tasks SET started_at = NOW(), worker_id = 'worker-healthy' WHERE id = $1`, taskB)

	// Heartbeat only the healthy one in; the stale one is simply absent
	// from the registry (never heartbeated, or long expired).
	registry.RecordHeartbeat("worker-healthy", "localhost:9999", 0, 5)
	// Manually seed a stale entry for worker-stale with an old timestamp.
	registry.mu.Lock()
	registry.workers["worker-stale"] = WorkerInfo{
		WorkerID: "worker-stale",
		LastSeen: time.Now().Add(-1 * time.Hour),
	}
	registry.mu.Unlock()

	reaper.checkStaleWorkers(context.Background())

	tA, _ := schedRepo.GetTask(context.Background(), taskA)
	tB, _ := schedRepo.GetTask(context.Background(), taskB)

	if tA.NeedsReviewAt == nil {
		t.Error("expected task on stale worker to be marked needs_review")
	}
	if tB.NeedsReviewAt != nil {
		t.Error("expected task on healthy worker to be untouched")
	}
}
