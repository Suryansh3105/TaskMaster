package coordinator

import (
	"context"
	"log"
	"time"
)

type Coordinator struct {
	repo *Repository
}

func NewCoordinator(repo *Repository) *Coordinator {
	return &Coordinator{repo: repo}
}

func (c *Coordinator) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("coordinator: claim loop stopping")
			return
		case <-ticker.C:
			claimed, err := c.repo.ClaimDueTasks(ctx, 10)
			if err != nil {
				log.Printf("coordinator: claim failed: %v", err)
				continue
			}
			if len(claimed) > 0 {
				log.Printf("coordinator: claimed %d task(s)", len(claimed))
			}
		}
	}
}
