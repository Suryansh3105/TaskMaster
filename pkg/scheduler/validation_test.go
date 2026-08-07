package scheduler

import (
	"testing"
	"time"
)

func TestValidateCommandRequest(t *testing.T) {
	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	past := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name    string
		req     CommandRequest
		wantErr error
	}{
		{"valid request", CommandRequest{Command: "echo hi", ScheduledAt: future}, nil},
		{"empty command", CommandRequest{Command: "", ScheduledAt: future}, ErrEmptyCommand},
		{"whitespace-only command", CommandRequest{Command: "   ", ScheduledAt: future}, ErrEmptyCommand},
		{"malformed timestamp", CommandRequest{Command: "echo hi", ScheduledAt: "not-a-time"}, ErrInvalidTime},
		{"scheduled in the past", CommandRequest{Command: "echo hi", ScheduledAt: past}, ErrScheduledInPast},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCommandRequest(tt.req)
			if tt.wantErr == nil && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
