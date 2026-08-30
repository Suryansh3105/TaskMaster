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
