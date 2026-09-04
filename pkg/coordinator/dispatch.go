package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const renewInterval = 5 * time.Second

type Dispatcher struct {
	repo     *Repository
	registry *Registry
	worker   *WorkerClient
	clients  map[string]*WorkerClient
	mu       sync.Mutex
}

func NewDispatcher(repo *Repository, registry *Registry) *Dispatcher {
	return &Dispatcher{
		repo:     repo,
		registry: registry,
		clients:  make(map[string]*WorkerClient),
	}
}

func (d *Dispatcher) getClient(address string) (*WorkerClient, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if c, ok := d.clients[address]; ok {
		return c, nil
	}
	c, err := NewWorkerClient(address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to worker at %s: %w", address, err)
	}
	d.clients[address] = c
	return c, nil
}

func (d *Dispatcher) Dispatch(ctx context.Context, task ClaimedTask) {
	worker, err := SelectWorker(d.registry.Snapshot())
	if err != nil {
		log.Printf("dispatch: no available worker for task %s: %v", task.ID, err)
		return
	}

	client, err := d.getClient(worker.Address)
	if err != nil {
		log.Printf("dispatch: %v", err)
		return
	}

	if err := d.repo.MarkDispatchAttempted(ctx, task.ID); err != nil {
		log.Printf("dispatch: failed to mark attempted for task %s: %v", task.ID, err)
		return
	}

	resp, err := client.AssignTask(ctx, task.ID, task.Command)
	if err != nil {
		log.Printf("dispatch: AssignTask failed for task %s on worker %s: %v", task.ID, worker.WorkerID, err)
		return
	}
	if !resp.Accepted {
		log.Printf("dispatch: worker %s rejected task %s", worker.WorkerID, task.ID)
		return
	}

	if err := d.repo.MarkStarted(ctx, task.ID, worker.WorkerID); err != nil {
		log.Printf("dispatch: failed to mark started for task %s: %v", task.ID, err)
	}

	go d.renewLeaseLoop(ctx, task.ID, renewInterval)
}

func (d *Dispatcher) renewLeaseLoop(ctx context.Context, taskID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			renewed, err := d.repo.RenewClaimConditional(ctx, taskID)
			if err != nil {
				log.Printf("renew: failed for task %s: %v", taskID, err)
				continue
			}
			if !renewed {
				log.Printf("renew: task %s no longer renewable, stopping", taskID)
				return
			}
		}
	}
}
