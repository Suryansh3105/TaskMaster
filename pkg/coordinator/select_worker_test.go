package coordinator

import "testing"

func TestSelectWorker_SkipsFullCapacity(t *testing.T) {
	workers := map[string]WorkerInfo{
		"w1": {WorkerID: "w1", Address: "localhost:9090", RunningTasks: 5, MaxCapacity: 5},
		"w2": {WorkerID: "w2", Address: "localhost:9092", RunningTasks: 1, MaxCapacity: 5},
	}
	selected, err := SelectWorker(workers)
	if err != nil {
		t.Fatalf("expected a worker, got error: %v", err)
	}
	if selected.WorkerID != "w2" {
		t.Errorf("expected w2 (has capacity), got %s", selected.WorkerID)
	}
}

func TestSelectWorker_NoneAvailable(t *testing.T) {
	workers := map[string]WorkerInfo{
		"w1": {WorkerID: "w1", RunningTasks: 5, MaxCapacity: 5},
	}
	_, err := SelectWorker(workers)
	if err != ErrNoAvailableWorker {
		t.Errorf("expected ErrNoAvailableWorker, got %v", err)
	}
}