package runtime

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
)

type RunHostRequest struct {
	Command string
	Args    []string
}

func RunHost(ctx context.Context, req RunHostRequest) error {
	if req.Command == "" {
		return errors.New("command is required")
	}

	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			status := exitErr.Sys().(syscall.WaitStatus)
			code := status.ExitStatus()
			if code != 0 {
				// Forward the exit code from the command to the caller
				os.Exit(code)
			}
		}
		return err
	}
	return nil
}