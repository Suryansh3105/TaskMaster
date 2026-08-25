package coordinator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/common"
	"github.com/Suryansh3105/taskmaster/pkg/scheduler"
)

func setupCoordinatorTestRepo(t *testing.T) *Repository {
	t.Helper()
	ctx := context.Background()
	connString, err := common.GetDBConnectionString()
	if err != nil {
		t.Fatalf("config error: %v", err)
	}
	pool, err := common.ConnectToDatabase(ctx, connString)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return NewRepository(pool)
}

func TestClaimDueTasks_NoDoubleClaim(t *testing.T) {
	repo := setupCoordinatorTestRepo(t)

	connString, _ := common.GetDBConnectionString()
	pool, _ := common.ConnectToDatabase(context.Background(), connString)
	defer pool.Close()
	schedRepo := scheduler.NewRepository(pool)

	const numTasks = 20
	for i := 0; i < numTasks; i++ {
		_, err := schedRepo.InsertTask(context.Background(), "echo test", time.Now().Add(-1*time.Minute))
		if err != nil {
			t.Fatalf("failed to insert task: %v", err)
		}
	}

	var wg sync.WaitGroup
	results := make([][]ClaimedTask, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimed, err := repo.ClaimDueTasks(context.Background(), numTasks)
			if err != nil {
				t.Errorf("claim %d failed: %v", idx, err)
				return
			}
			results[idx] = claimed
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool)
	total := 0
	for _, r := range results {
		for _, task := range r {
			if seen[task.ID] {
				t.Fatalf("task %s was claimed more than once", task.ID)
			}
			seen[task.ID] = true
			total++
		}
	}

	if total != numTasks {
		t.Errorf("expected %d tasks claimed across both calls, got %d", numTasks, total)
	}
}
