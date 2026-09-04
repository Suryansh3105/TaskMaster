package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := common.ConnectToDatabase(ctx, connString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	repo := coordinator.NewRepository(pool)
	registry := coordinator.NewRegistry()
	dispatcher := coordinator.NewDispatcher(repo, registry)
	coord := coordinator.NewCoordinator(repo, dispatcher)

	lis, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, coordinator.NewServer(repo, registry))
	reflection.Register(grpcServer)
	go func() {
		log.Println("coordinator gRPC server listening on :9091")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc server error: %v", err)
		}
	}()

	log.Println("coordinator claim loop starting")
	go coord.Run(ctx, 2*time.Second)
	reaper := coordinator.NewReaper(repo, registry)
	log.Println("starting reaper")
	go reaper.Run(ctx, 5*time.Second)

	handler := coordinator.NewHandler(repo)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks/{id}/requeue", handler.HandleRequeue)

	httpServer := &http.Server{Addr: ":8082", Handler: mux}
	go func() {
		log.Println("coordinator HTTP server listening on :8082")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("shutdown signal received")
}
