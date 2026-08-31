package coordinator

import (
	"context"
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
