package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	server, err := worker.NewServer(config)
	if err != nil {
		log.Fatalf("failed to create worker server: %v", err)
	}

	lis, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	grpcapi.RegisterTaskServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	heartbeatCtx, stopHeartbeats := context.WithCancel(context.Background())
	go worker.SendHeartbeats(heartbeatCtx, config, server)

	go func() {
		log.Printf("worker %s listening on %s", config.WorkerID, config.ListenAddress)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("worker: shutdown signal received, stopping heartbeats and refusing new tasks")
	stopHeartbeats()
	grpcServer.GracefulStop()

	log.Println("worker: waiting for in-flight task executions to finish")
	done := make(chan struct{})
	go func() {
		server.WaitForInFlight()
		close(done)
	}()

	select {
	case <-done:
		log.Println("worker: all in-flight executions finished, shutting down cleanly")
	case <-time.After(30 * time.Second):
		log.Println("worker: shutdown timeout reached, exiting with tasks still in flight")
	}
}
