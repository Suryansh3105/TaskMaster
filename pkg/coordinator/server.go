package coordinator

import (
	"context"
	"log"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
)

type Server struct {
	pb.UnimplementedTaskServiceServer
	repo     *Repository
	registry *Registry
}

func NewServer(repo *Repository, registry *Registry) *Server {
	return &Server{repo: repo, registry: registry}
}

func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.registry.RecordHeartbeat(req.WorkerId, req.RunningTasks, req.MaxCapacity)
	return &pb.HeartbeatResponse{Acknowledged: true}, nil
}

func (s *Server) ConfirmDone(ctx context.Context, req *pb.ConfirmDoneRequest) (*pb.ConfirmDoneResponse, error) {
	if !req.Success {
		log.Printf("coordinator: task %s reported failure: %s", req.TaskId, req.ErrorMessage)
		return &pb.ConfirmDoneResponse{Acknowledged: true}, nil
	}

	recorded, err := s.repo.MarkCompleted(ctx, req.TaskId)
	if err != nil {
		log.Printf("coordinator: failed to mark task %s completed: %v", req.TaskId, err)
		return &pb.ConfirmDoneResponse{Acknowledged: false}, err
	}
	if !recorded {
		log.Printf("coordinator: task %s completion already recorded (duplicate confirm)", req.TaskId)
	}
	return &pb.ConfirmDoneResponse{Acknowledged: true}, nil
}
