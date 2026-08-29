package coordinator

import (
	"context"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WorkerClient struct {
	conn   *grpc.ClientConn
	client pb.TaskServiceClient
}

func NewWorkerClient(address string) (*WorkerClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &WorkerClient{conn: conn, client: pb.NewTaskServiceClient(conn)}, nil
}

func (w *WorkerClient) AssignTask(ctx context.Context, taskID, command string) (*pb.AssignTaskResponse, error) {
	return w.client.AssignTask(ctx, &pb.AssignTaskRequest{TaskId: taskID, Command: command})
}

func (w *WorkerClient) Close() error {
	return w.conn.Close()
}
