package worker

import (
	"context"
	"log"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
)

type Server struct {
	pb.UnimplementedTaskServiceServer
	config Config
}

func NewServer(config Config) *Server {
	return &Server{config: config}
}

func (s *Server) AssignTask(ctx context.Context, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error) {
	log.Printf("worker %s: received task %s (command: %q)", s.config.WorkerID, req.TaskId, req.Command)
	go func() {
		result := Execute(context.Background(), req.Command)
		if result.Success {
			log.Printf("worker %s: task %s completed successfully", s.config.WorkerID, req.TaskId)
		} else {
			log.Printf("worker %s: task %s failed: %s", s.config.WorkerID, req.TaskId, result.ErrorMessage)
		}
	}()
	return &pb.AssignTaskResponse{Accepted: true}, nil
}
