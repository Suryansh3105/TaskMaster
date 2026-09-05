package worker

import (
	"context"
	"os/exec"
	"strings"
)

type ExecutionResult struct {
	Success      bool
	ErrorMessage string
}

func Execute(ctx context.Context, command string) ExecutionResult {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return ExecutionResult{
			Success:      false,
			ErrorMessage: strings.TrimSpace(string(output)) + ": " + err.Error(),
		}
	}
	return ExecutionResult{Success: true}
}
