package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	pb.UnimplementedTaskServiceServer
	config            Config
	runningTasks      int32
	coordinatorConn   *grpc.ClientConn
	coordinatorClient pb.TaskServiceClient
	inFlight          sync.WaitGroup
	slots             chan struct{} // buffered channel used as a capacity semaphore
}

func NewServer(config Config) (*Server, error) {
	conn, err := grpc.NewClient(config.CoordinatorAddresses[0], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Server{
		config:            config,
		coordinatorConn:   conn,
		coordinatorClient: pb.NewTaskServiceClient(conn),
		slots:             make(chan struct{}, config.MaxCapacity),
	}, nil
}

func (s *Server) RunningTasks() int32 {
	return atomic.LoadInt32(&s.runningTasks)
}

func (s *Server) WaitForInFlight() {
	s.inFlight.Wait()
}

func (s *Server) AssignTask(ctx context.Context, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error) {
	select {
	case s.slots <- struct{}{}:

	default:
		log.Printf("worker %s: rejecting task %s — at capacity", s.config.WorkerID, req.TaskId)
		return &pb.AssignTaskResponse{Accepted: false}, nil
	}

	log.Printf("worker %s: received task %s (command: %q)", s.config.WorkerID, req.TaskId, req.Command)

	atomic.AddInt32(&s.runningTasks, 1)
	s.inFlight.Add(1)

	go func() {
		defer func() {
			<-s.slots
			atomic.AddInt32(&s.runningTasks, -1)
			s.inFlight.Done()
		}()

		result := Execute(context.Background(), req.Command)
		if result.Success {
			log.Printf("worker %s: task %s completed successfully", s.config.WorkerID, req.TaskId)
		} else {
			log.Printf("worker %s: task %s failed: %s", s.config.WorkerID, req.TaskId, result.ErrorMessage)
		}
		ConfirmDone(context.Background(), s.coordinatorClient, req.TaskId, result)
	}()

	return &pb.AssignTaskResponse{Accepted: true}, nil
}
