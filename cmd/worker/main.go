package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Suryansh3105/taskmaster/pkg/grpcapi"
	"github.com/Suryansh3105/taskmaster/pkg/worker"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	config, err := worker.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	lis, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server, err := worker.NewServer(config)
	if err != nil {
		log.Fatalf("failed to create worker server: %v", err)
	}
	grpcServer := grpc.NewServer()
	grpcapi.RegisterTaskServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	ctx, cancel := context.WithCancel(context.Background())
	go worker.SendHeartbeats(ctx, config, server)

	go func() {
		log.Printf("worker %s listening on %s", config.WorkerID, config.ListenAddress)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("worker shutting down")
	cancel() // stops the heartbeat loop
	grpcServer.GracefulStop()
}
