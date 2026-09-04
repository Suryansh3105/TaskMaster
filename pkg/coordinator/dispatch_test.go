package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/common"
	"github.com/Suryansh3105/taskmaster/pkg/scheduler"
)

func TestDispatch_SuccessfulFlow(t *testing.T) {
	repo := setupCoordinatorTestRepo(t)

	registry := NewRegistry()

	registry.RecordHeartbeat("stub-1", "localhost:9090", 0, 5)

	dispatcher := NewDispatcher(repo, registry)

	connString, _ := common.GetDBConnectionString()
	pool, _ := common.ConnectToDatabase(context.Background(), connString)
	defer pool.Close()
	schedRepo := scheduler.NewRepository(pool)

	taskID, err := schedRepo.InsertTask(context.Background(), "echo hi", time.Now().Add(-1*time.Minute))
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	claimed, err := repo.ClaimDueTasks(context.Background(), 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("expected to claim exactly 1 task, got %d, err: %v", len(claimed), err)
	}

	dispatcher.Dispatch(context.Background(), claimed[0])

	task, err := schedRepo.GetTask(context.Background(), taskID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if task.StartedAt == nil {
		t.Error("expected started_at to be set after successful dispatch")
	}
}

func TestRenewLeaseLoop_StopsWhenNeedsReview(t *testing.T) {
	pool := testPool(t)
	cleanTasksTable(t, pool)

	repo := NewRepository(pool)
	schedRepo := scheduler.NewRepository(pool)

	taskID, _ := schedRepo.InsertTask(context.Background(), "echo hi", time.Now().Add(-1*time.Minute))
	pool.Exec(context.Background(),
		`UPDATE tasks SET picked_at = NOW(), started_at = NOW(),
		 dispatch_attempted_at = NOW() WHERE id = $1`, taskID)

	dispatcher := NewDispatcher(repo, NewRegistry())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go dispatcher.renewLeaseLoop(ctx, taskID, 100*time.Millisecond)

	time.Sleep(150 * time.Millisecond) // let it renew at least once

	// Simulate checkStaleWorkers flagging the task independently.
	pool.Exec(context.Background(),
		`UPDATE tasks SET needs_review_at = NOW() WHERE id = $1`, taskID)

	time.Sleep(150 * time.Millisecond) // give the loop a chance to notice and stop

	before, _ := schedRepo.GetTask(context.Background(), taskID)

	time.Sleep(200 * time.Millisecond) // if the loop is still running, this would renew again

	after, _ := schedRepo.GetTask(context.Background(), taskID)

	if before.ClaimRenewedAt != nil && after.ClaimRenewedAt != nil &&
		!before.ClaimRenewedAt.Equal(*after.ClaimRenewedAt) {
		t.Error("expected renewal to have stopped after needs_review was set, but claim_renewed_at kept changing")
	}
}

func TestRenewLeaseLoop_RenewsWhileHealthy(t *testing.T) {
	pool := testPool(t)
	cleanTasksTable(t, pool)

	repo := NewRepository(pool)
	schedRepo := scheduler.NewRepository(pool)

	taskID, _ := schedRepo.InsertTask(context.Background(), "echo hi", time.Now().Add(-1*time.Minute))
	pool.Exec(context.Background(),
		`UPDATE tasks SET picked_at = NOW(), started_at = NOW(),
		 dispatch_attempted_at = NOW(), claim_renewed_at = NOW() - INTERVAL '25 seconds'
		 WHERE id = $1`, taskID)

	dispatcher := NewDispatcher(repo, NewRegistry())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first, _ := schedRepo.GetTask(context.Background(), taskID)

	go dispatcher.renewLeaseLoop(ctx, taskID, 100*time.Millisecond)
	time.Sleep(250 * time.Millisecond)

	second, _ := schedRepo.GetTask(context.Background(), taskID)

	if second.ClaimRenewedAt == nil || !second.ClaimRenewedAt.After(*first.ClaimRenewedAt) {
		t.Error("expected claim_renewed_at to have advanced while the loop was running healthily")
	}
}
