package coordinator

import "testing"

func TestRegistry_RecordAndSnapshot(t *testing.T) {
	reg := NewRegistry()

	reg.RecordHeartbeat("worker-1", 2, 5)
	reg.RecordHeartbeat("worker-2", 0, 5)

	snap := reg.Snapshot()

	if len(snap) != 2 {
		t.Fatalf("expected 2 workers in registry, got %d", len(snap))
	}
	if snap["worker-1"].RunningTasks != 2 {
		t.Errorf("expected worker-1 running_tasks=2, got %d", snap["worker-1"].RunningTasks)
	}

	// A second heartbeat from the same worker updates, doesn't duplicate.
	reg.RecordHeartbeat("worker-1", 3, 5)
	snap = reg.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected still 2 workers after re-heartbeat, got %d", len(snap))
	}
	if snap["worker-1"].RunningTasks != 3 {
		t.Errorf("expected worker-1 running_tasks updated to 3, got %d", snap["worker-1"].RunningTasks)
	}
}