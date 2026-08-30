package coordinator

import (
	"testing"
	"time"
)

func TestNextAttemptDelay_GrowsAndCaps(t *testing.T) {
	if d := NextAttemptDelay(1); d != 2*time.Second {
		t.Errorf("attempt 1: expected 2s, got %v", d)
	}
	if d := NextAttemptDelay(2); d != 4*time.Second {
		t.Errorf("attempt 2: expected 4s, got %v", d)
	}
	if d := NextAttemptDelay(3); d != 8*time.Second {
		t.Errorf("attempt 3: expected 8s, got %v", d)
	}
	if d := NextAttemptDelay(20); d != maxDelay {
		t.Errorf("attempt 20: expected cap at %v, got %v", maxDelay, d)
	}
}