package worker

import (
	"context"
	"testing"
)

func TestExecute_Success(t *testing.T) {
	result := Execute(context.Background(), "echo hello")
	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}
}

func TestExecute_Failure(t *testing.T) {
	result := Execute(context.Background(), "exit 1")
	if result.Success {
		t.Error("expected failure, got success")
	}
	if result.ErrorMessage == "" {
		t.Error("expected a non-empty error message on failure")
	}
}

func TestExecute_CapturesOutput(t *testing.T) {
	result := Execute(context.Background(), "this-command-does-not-exist")
	if result.Success {
		t.Error("expected failure for a nonexistent command")
	}
	if result.ErrorMessage == "" {
		t.Error("expected an error message describing the failure")
	}
}
