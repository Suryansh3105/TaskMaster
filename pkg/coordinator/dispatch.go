package coordinator

import (
	"context"
	"log"
)

type Dispatcher struct {
	repo   *Repository
	worker *WorkerClient
}

func NewDispatcher(repo *Repository, worker *WorkerClient) *Dispatcher {
	return &Dispatcher{repo: repo, worker: worker}
}

func (d *Dispatcher) Dispatch(ctx context.Context, task ClaimedTask) {
	if err := d.repo.MarkDispatchAttempted(ctx, task.ID); err != nil {
		log.Printf("dispatch: failed to mark attempted for task %s: %v", task.ID, err)
		return
	}

	resp, err := d.worker.AssignTask(ctx, task.ID, task.Command)
	if err != nil {
		log.Printf("dispatch: AssignTask failed for task %s: %v", task.ID, err)
		return
	}

	if !resp.Accepted {
		log.Printf("dispatch: worker rejected task %s", task.ID)
		return
	}

	if err := d.repo.MarkStarted(ctx, task.ID); err != nil {
		log.Printf("dispatch: failed to mark started for task %s: %v", task.ID, err)
	}
}