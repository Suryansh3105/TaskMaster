package coordinator

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) HandleRequeue(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, "task id is required", http.StatusBadRequest)
		return
	}

	needsReview, err := h.repo.IsNeedsReview(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to check task state", http.StatusInternalServerError)
		return
	}
	if !needsReview {
		http.Error(w, "task is not flagged for review", http.StatusConflict)
		return
	}

	if err := h.repo.RequeueTask(r.Context(), taskID); err != nil {
		http.Error(w, "failed to requeue task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "requeued"})
}
