package worker

import (
	"context"
	"testing"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
)

func TestAssignTask_RejectsWhenAtCapacity(t *testing.T) {
	config := Config{
		WorkerID:             "test-worker",
		CoordinatorAddresses: []string{"localhost:1"}, // won't actually be dialed successfully, fine for this test
		MaxCapacity:          2,
	}
	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	// Fill both slots with slow-running tasks.
	for i := 0; i < 2; i++ {
		resp, err := server.AssignTask(context.Background(), &pb.AssignTaskRequest{
			TaskId:  "slow-task",
			Command: "sleep 2",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Accepted {
			t.Fatalf("expected task %d to be accepted (capacity not yet reached)", i)
		}
	}

	// A third task, while both slots are occupied, should be rejected.
	resp, err := server.AssignTask(context.Background(), &pb.AssignTaskRequest{
		TaskId:  "over-capacity-task",
		Command: "echo should be rejected",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Accepted {
		t.Error("expected task to be rejected — worker should be at capacity")
	}

	server.WaitForInFlight() // let the sleep 2 tasks finish before the test exits
}

func TestAssignTask_AcceptsAgainAfterSlotFrees(t *testing.T) {
	fake := &failFirstNServer{failCount: 0} // always succeeds immediately
	_, client, cleanup := startTestGRPCServer(t, fake)
	defer cleanup()

	config := Config{
		WorkerID:             "test-worker",
		CoordinatorAddresses: []string{"localhost:1"}, // unused directly, since we override the client below
		MaxCapacity:          1,
	}
	server := &Server{
		config:            config,
		coordinatorClient: client, // inject the working test client directly
		slots:             make(chan struct{}, config.MaxCapacity),
	}

	resp, _ := server.AssignTask(context.Background(), &pb.AssignTaskRequest{TaskId: "quick-task", Command: "echo hi"})
	if !resp.Accepted {
		t.Fatal("expected first task to be accepted")
	}

	server.inFlight.Wait() // wait for the FULL goroutine (execution + confirm) to finish, not a guessed sleep

	resp2, _ := server.AssignTask(context.Background(), &pb.AssignTaskRequest{TaskId: "second-task", Command: "echo hi"})
	if !resp2.Accepted {
		t.Error("expected second task to be accepted once the first freed its slot")
	}

	server.WaitForInFlight()
}
