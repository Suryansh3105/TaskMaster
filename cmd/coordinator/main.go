package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/common"
	"github.com/Suryansh3105/taskmaster/pkg/coordinator"
	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	connString, err := common.GetDBConnectionString()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	pool, err := common.ConnectToDatabase(ctx, connString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	repo := coordinator.NewRepository(pool)
	workerClient, err := coordinator.NewWorkerClient("localhost:9090")
	if err != nil {
		log.Fatalf("failed to connect to worker: %v", err)
	}
	dispatcher := coordinator.NewDispatcher(repo, workerClient)
	coord := coordinator.NewCoordinator(repo, dispatcher)

	lis, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, coordinator.NewServer(repo))
	reflection.Register(grpcServer)
	go func() {
		log.Println("coordinator gRPC server listening on :9091")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc server error: %v", err)
		}
	}()

	log.Println("coordinator claim loop starting")
	coord.Run(ctx, 2*time.Second)

}
