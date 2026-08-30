package coordinator

import "errors"

var ErrNoAvailableWorker = errors.New("no worker with available capacity")

func SelectWorker(workers map[string]WorkerInfo) (WorkerInfo, error) {
	var best WorkerInfo
	found := false

	for _, w := range workers {
		if w.RunningTasks >= w.MaxCapacity {
			continue // at capacity, skip
		}
		if !found || w.RunningTasks < best.RunningTasks {
			best = w
			found = true
		}
	}

	if !found {
		return WorkerInfo{}, ErrNoAvailableWorker
	}
	return best, nil
}
