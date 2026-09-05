package worker

import (
	"context"
	"log"
	"time"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const heartbeatInterval = 5 * time.Second

func SendHeartbeats(ctx context.Context, config Config, server *Server) {
	clients := make(map[string]pb.TaskServiceClient)
	for _, addr := range config.CoordinatorAddresses {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("heartbeat: failed to connect to coordinator %s: %v", addr, err)
			continue
		}
		clients[addr] = pb.NewTaskServiceClient(conn)
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("heartbeat: stopping")
			return
		case <-ticker.C:
			for addr, client := range clients {
				_, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{
					WorkerId:     config.WorkerID,
					Address:      config.ListenAddress,
					RunningTasks: server.RunningTasks(),
					MaxCapacity:  config.MaxCapacity,
				})
				if err != nil {
					log.Printf("heartbeat: failed to reach coordinator %s: %v", addr, err)
				}
			}
		}
	}
}
