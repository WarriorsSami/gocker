package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type RunCmdRequest struct {
	Command string
	Args    []string
}

func RunParent(ctx context.Context, req RunCmdRequest) error {
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create reexec pipe: %w", err)
	}

	// "--" stops cobra from interpreting the target command's flags (e.g. -c)
	// as flags belonging to the "child" subcommand.
	child := exec.CommandContext(ctx, "/proc/self/exe", append([]string{"child", "--", req.Command}, req.Args...)...)
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.ExtraFiles = []*os.File{r} // becomes fd 3 in the child and is also used to sync the parent and child during setup

	if err := child.Start(); err != nil {
		must(r.Close())
		must(w.Close())
		return err
	}

	// Perform any parent-side setup here if needed (e.g. waiting for the child to signal it's ready via the pipe)

	must(r.Close()) // Not needed in the parent, close it to avoid leaks
	must(w.Close()) // Signal the child that setup is done (if the child is waiting for this, it will unblock when we close the write end)

	if err := child.Wait(); err != nil {
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

func RunChild(ctx context.Context, req RunCmdRequest) error {
	// Block until the parent closes the write end of the pipe (fd 3), which signals that setup is complete
	var buf [1]byte
	must(syscall.Read(3, buf[:])) // returns when parent closes w -> EOF
	must(syscall.Close(3))

	// Perform any child-side setup if needed
	
	// Exec the target command, replacing the child process
	path, err := exec.LookPath(req.Command)
	if err != nil {
		return fmt.Errorf("failed to find command %q: %w", req.Command, err)
	}

	return syscall.Exec(path, append([]string{req.Command}, req.Args...), os.Environ())
}

// must panics if the last argument is a non-nil error.
// It accepts any number of preceding values so it can wrap multi-return calls
// directly, e.g. must(syscall.Read(3, buf[:])) or must(r.Close()).
func must(args ...any) {
	last := args[len(args)-1]
	if last == nil {
		return
	}
	if err, ok := last.(error); ok && err != nil {
		panic(err)
	}
}