package worker

import (
	"context"
	"net"
	"testing"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// failFirstNServer accepts ConfirmDone calls, failing the first N,
// then succeeding — lets the test observe retry behavior directly.
type failFirstNServer struct {
	pb.UnimplementedTaskServiceServer
	failCount int
	calls     int
}

func (f *failFirstNServer) ConfirmDone(ctx context.Context, req *pb.ConfirmDoneRequest) (*pb.ConfirmDoneResponse, error) {
	f.calls++
	if f.calls <= f.failCount {
		return nil, context.DeadlineExceeded
	}
	return &pb.ConfirmDoneResponse{Acknowledged: true}, nil
}

func TestConfirmDone_RetriesOnFailure(t *testing.T) {
	fake := &failFirstNServer{failCount: 2} // fails twice, succeeds on the 3rd

	lis, client, cleanup := startTestGRPCServer(t, fake)
	defer cleanup()
	_ = lis

	ConfirmDone(context.Background(), client, "test-task", ExecutionResult{Success: true})

	if fake.calls != 3 {
		t.Errorf("expected 3 attempts (2 failures + 1 success), got %d", fake.calls)
	}
}

func TestConfirmDone_GivesUpAfterMaxRetries(t *testing.T) {
	fake := &failFirstNServer{failCount: 100} // always fails

	_, client, cleanup := startTestGRPCServer(t, fake)
	defer cleanup()

	ConfirmDone(context.Background(), client, "test-task", ExecutionResult{Success: true})

	if fake.calls != confirmMaxRetries {
		t.Errorf("expected exactly %d attempts, got %d", confirmMaxRetries, fake.calls)
	}
}

// startTestGRPCServer is a small test helper — starts a real gRPC
// server on a random local port backed by the given implementation.
func startTestGRPCServer(t *testing.T, impl pb.TaskServiceServer) (net.Listener, pb.TaskServiceClient, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterTaskServiceServer(s, impl)
	go s.Serve(lis)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	client := pb.NewTaskServiceClient(conn)

	cleanup := func() {
		conn.Close()
		s.Stop()
	}
	return lis, client, cleanup
}
