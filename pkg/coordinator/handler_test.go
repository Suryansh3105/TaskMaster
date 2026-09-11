package coordinator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/scheduler"
)

func TestHandleRequeue_Success(t *testing.T) {
	pool := testPool(t)
	cleanTasksTable(t, pool)

	repo := NewRepository(pool)
	handler := NewHandler(repo)
	schedRepo := scheduler.NewRepository(pool)

	taskID, _ := schedRepo.InsertTask(context.Background(), "echo hi", time.Now().Add(-1*time.Minute))
	pool.Exec(context.Background(),
		`UPDATE tasks SET picked_at = NOW(), needs_review_at = NOW() WHERE id = $1`, taskID)

	req := httptest.NewRequest(http.MethodPost, "/tasks/"+taskID+"/requeue", nil)
	req.SetPathValue("id", taskID)
	w := httptest.NewRecorder()

	handler.HandleRequeue(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	task, _ := schedRepo.GetTask(context.Background(), taskID)
	if task.NeedsReviewAt != nil {
		t.Error("expected needs_review_at to be cleared")
	}
	if task.PickedAt != nil {
		t.Error("expected picked_at to be cleared, task should be claimable again")
	}
}

func TestHandleRequeue_NotFlaggedForReview(t *testing.T) {
	pool := testPool(t)
	cleanTasksTable(t, pool)

	repo := NewRepository(pool)
	handler := NewHandler(repo)
	schedRepo := scheduler.NewRepository(pool)

	taskID, _ := schedRepo.InsertTask(context.Background(), "echo hi", time.Now().Add(-1*time.Minute))

	req := httptest.NewRequest(http.MethodPost, "/tasks/"+taskID+"/requeue", nil)
	req.SetPathValue("id", taskID)
	w := httptest.NewRecorder()

	handler.HandleRequeue(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestHandleListTasks_ReturnsTasks(t *testing.T) {
	pool := testPool(t)
	cleanTasksTable(t, pool)

	repo := NewRepository(pool)
	handler := NewHandler(repo)
	schedRepo := scheduler.NewRepository(pool)

	schedRepo.InsertTask(context.Background(), "task one", time.Now().Add(1*time.Hour))
	schedRepo.InsertTask(context.Background(), "task two", time.Now().Add(2*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()

	handler.HandleListTasks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var tasks []scheduler.Task
	if err := json.NewDecoder(w.Body).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}
