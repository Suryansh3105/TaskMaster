package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Suryansh3105/taskmaster/pkg/common"
	"github.com/Suryansh3105/taskmaster/pkg/scheduler"
)

func main() {
	ctx := context.Background()

	connString, err := common.GetDBConnectionString()
	if err != nil {
		log.Fatalf("failed to get database connection string: %v", err)
	}

	pool, err := common.ConnectToDatabase(ctx, connString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	repo := scheduler.NewRepository(pool)
	handler := scheduler.NewHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /schedule", handler.HandleSchedule)
	mux.HandleFunc("GET /status/{id}", handler.HandleStatus)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("scheduler listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("scheduler stopped")
}
