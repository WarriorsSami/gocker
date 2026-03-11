package runtime

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

const GockerReExecEnv = "GOCKER_REEXEC_TOKEN"
const reExecTokenLen = 16

type RunCmdRequest struct {
	Command string
	Args    []string
}

func RunParent(ctx context.Context, req RunCmdRequest) error {
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create reexec pipe: %w", err)
	}

	// Generate a random one-time token to authenticate the re-exec child.
	var tokenBytes [reExecTokenLen]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		must(r.Close())
		must(w.Close())
		return fmt.Errorf("failed to generate reexec token: %w", err)
	}

	// "--" stops cobra from interpreting the target command's flags (e.g. -c)
	// as flags belonging to the "child" subcommand.
	child := exec.CommandContext(ctx, "/proc/self/exe", append([]string{"child", "--", req.Command}, req.Args...)...)
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.Env = append(os.Environ(), fmt.Sprintf("%s=%s", GockerReExecEnv, hex.EncodeToString(tokenBytes[:])))
	child.ExtraFiles = []*os.File{r} // becomes fd 3 in the child

	if err := child.Start(); err != nil {
		must(r.Close())
		must(w.Close())
		return err
	}

	// Perform any parent-side setup here if needed.

	must(r.Close()) // not needed in the parent
	// Write the token to the pipe; child reads & verifies it, then gets EOF as the "go" signal.
	if _, err := w.Write(tokenBytes[:]); err != nil {
		must(w.Close())
		return fmt.Errorf("failed to write reexec token: %w", err)
	}
	must(w.Close())

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
	// Read the random token from the pipe (blocks until the parent writes it).
	var token [reExecTokenLen]byte
	if _, err := io.ReadFull(os.NewFile(3, "pipe"), token[:]); err != nil {
		return fmt.Errorf("failed to read reexec token: %w", err)
	}
	must(syscall.Close(3))

	// Verify the token matches what the parent placed in the environment.
	expected, err := hex.DecodeString(os.Getenv(GockerReExecEnv))
	if err != nil || subtle.ConstantTimeCompare(token[:], expected) != 1 {
		return fmt.Errorf("invalid reexec token: unauthorized invocation")
	}

	// Don't leak the token into the execed process's environment.
	must(os.Unsetenv(GockerReExecEnv))

	// Perform any child-side setup if needed.

	// Exec the target command, replacing the child process.
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
