package coordinator

import (
	"math"
	"time"
)

const (
	baseDelay = 2 * time.Second
	maxDelay  = 5 * time.Minute
)

func NextAttemptDelay(attemptNumber int) time.Duration {
	delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attemptNumber-1)))
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}
