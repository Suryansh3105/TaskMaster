package coordinator

import (
	"sync"
	"time"
)

type WorkerInfo struct {
	WorkerID     string
	Address      string
	RunningTasks int32
	MaxCapacity  int32
	LastSeen     time.Time
}

type Registry struct {
	mu      sync.RWMutex
	workers map[string]WorkerInfo
}

func NewRegistry() *Registry {
	return &Registry{workers: make(map[string]WorkerInfo)}
}

func (r *Registry) RecordHeartbeat(workerID, address string, runningTasks, maxCapacity int32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workers[workerID] = WorkerInfo{
		WorkerID:     workerID,
		Address:      address,
		RunningTasks: runningTasks,
		MaxCapacity:  maxCapacity,
		LastSeen:     time.Now(),
	}
}

func (r *Registry) Snapshot() map[string]WorkerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	snap := make(map[string]WorkerInfo, len(r.workers))
	for k, v := range r.workers {
		snap[k] = v
	}
	return snap
}

func (r *Registry) StaleWorkers(timeout time.Duration) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var stale []string
	cutoff := time.Now().Add(-timeout)
	for id, w := range r.workers {
		if w.LastSeen.Before(cutoff) {
			stale = append(stale, id)
		}
	}
	return stale
}
