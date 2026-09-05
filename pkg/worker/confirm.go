package worker

import (
	"context"
	"log"
	"time"

	pb "github.com/Suryansh3105/taskmaster/pkg/grpcapi"
)

const (
	confirmMaxRetries = 3
	confirmRetryDelay = 2 * time.Second
)

func ConfirmDone(ctx context.Context, client pb.TaskServiceClient, taskID string, result ExecutionResult) {
	req := &pb.ConfirmDoneRequest{
		TaskId:       taskID,
		Success:      result.Success,
		ErrorMessage: result.ErrorMessage,
	}

	var lastErr error
	for attempt := 1; attempt <= confirmMaxRetries; attempt++ {
		_, err := client.ConfirmDone(ctx, req)
		if err == nil {
			return // acknowledged, done
		}
		lastErr = err
		log.Printf("confirm: attempt %d/%d failed for task %s: %v", attempt, confirmMaxRetries, taskID, err)
		if attempt < confirmMaxRetries {
			time.Sleep(confirmRetryDelay)
		}
	}
	log.Printf("confirm: giving up on task %s after %d attempts: %v", taskID, confirmMaxRetries, lastErr)
}