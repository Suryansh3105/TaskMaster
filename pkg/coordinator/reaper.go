package coordinator

import (
	"context"
	"log"
	"time"
)

const (
	claimLeaseTimeout = 30 * time.Second
	heartbeatTimeout  = 15 * time.Second
)

type Reaper struct {
	repo     *Repository
	registry *Registry
}

func NewReaper(repo *Repository, registry *Registry) *Reaper {
	return &Reaper{repo: repo, registry: registry}
}

func (r *Reaper) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("reaper: stopping")
			return
		case <-ticker.C:
			r.checkStaleClaims(ctx)
			r.checkStaleWorkers(ctx)
		}
	}
}

func (r *Reaper) checkStaleClaims(ctx context.Context) {
	stale, err := r.repo.FindStaleClaims(ctx, claimLeaseTimeout)
	if err != nil {
		log.Printf("reaper: failed to find stale claims: %v", err)
		return
	}

	for _, task := range stale {
		attempted, err := r.repo.WasDispatchAttempted(ctx, task.ID)
		if err != nil {
			log.Printf("reaper: failed to check dispatch state for task %s: %v", task.ID, err)
			continue
		}

		if !attempted {
			log.Printf("reaper: task %s claim expired, dispatch never attempted — auto-retrying", task.ID)
			if err := r.repo.ClearClaimForRetry(ctx, task.ID); err != nil {
				log.Printf("reaper: failed to clear claim for task %s: %v", task.ID, err)
			}
			continue
		}

		log.Printf("reaper: task %s claim expired, dispatch was in flight — marking needs_review", task.ID)
		if err := r.repo.MarkNeedsReview(ctx, task.ID); err != nil {
			log.Printf("reaper: failed to mark task %s needs_review: %v", task.ID, err)
		}
	}
}

func (r *Reaper) checkStaleWorkers(ctx context.Context) {
	staleWorkers := r.registry.StaleWorkers(heartbeatTimeout)
	for _, workerID := range staleWorkers {
		inProgress, err := r.repo.FindTasksInProgressForWorker(ctx, workerID)
		if err != nil {
			log.Printf("reaper: failed to find in-progress tasks for worker %s: %v", workerID, err)
			continue
		}
		for _, task := range inProgress {
			log.Printf("reaper: task %s was on stale worker %s — marking needs_review", task.ID, workerID)
			if err := r.repo.MarkNeedsReview(ctx, task.ID); err != nil {
				log.Printf("reaper: failed to mark task %s needs_review: %v", task.ID, err)
			}
		}
	}
}
