package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/common"
)

func setupTestHandler(t *testing.T) *Handler {
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
	return NewHandler(NewRepository(pool))
}

func TestHandleSchedule_Success(t *testing.T) {
	h := setupTestHandler(t)

	body, _ := json.Marshal(CommandRequest{
		Command:     "echo hi",
		ScheduledAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleSchedule(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp ScheduleResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TaskID == "" {
		t.Error("expected a non-empty task_id")
	}
}

func TestHandleSchedule_InvalidBody(t *testing.T) {
	h := setupTestHandler(t)

	body, _ := json.Marshal(CommandRequest{Command: "", ScheduledAt: "bad-time"})
	req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleSchedule(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestHandleStatus_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/status/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()

	h.HandleStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
