package worker

import (
	"fmt"
	"os"
)

type Config struct {
	WorkerID             string
	ListenAddress        string
	CoordinatorAddresses []string
	MaxCapacity          int32
}

func LoadConfig() (Config, error) {
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		return Config{}, fmt.Errorf("WORKER_ID must be set")
	}
	listenAddr := os.Getenv("WORKER_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":9092"
	}
	coordAddr := os.Getenv("COORDINATOR_ADDR")
	if coordAddr == "" {
		coordAddr = "localhost:9091"
	}
	return Config{
		WorkerID:             workerID,
		ListenAddress:        listenAddr,
		CoordinatorAddresses: []string{coordAddr},
		MaxCapacity:          5,
	}, nil
}
