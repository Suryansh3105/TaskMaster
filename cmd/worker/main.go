package main

import (
	"context"
	"log"
	"net"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type stubServer struct {
	pb.TaskServiceServer
}

func (s *stubServer) AssignTask(ctx context.Context, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error) {
	log.Printf("stub worker: received task %s (command: %q)", req.TaskId, req.Command)
	return &pb.AssignTaskResponse{Accepted: true}, nil
}

func (s *stubServer) ConfirmDone(ctx context.Context, req *pb.ConfirmDoneRequest) (*pb.ConfirmDoneResponse, error) {
	return &pb.ConfirmDoneResponse{Acknowledged: true}, nil
}

func (s *stubServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	return &pb.HeartbeatResponse{Acknowledged: true}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, &stubServer{})

	reflection.Register(grpcServer)

	log.Println("stub worker listening on :9090")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
