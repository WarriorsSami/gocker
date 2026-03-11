//go:build linux

// Package integration contains end-to-end tests that exercise the full gocker
// binary. They require a compiled binary at the workspace root; run
// 'go build -o gocker .' before executing these tests.
//
// Run with:
//
//	go test ./tests/integration/ -v
package integration

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// binPath locates the gocker binary. It looks for it next to the test binary
// first, then walks up to the module root and checks there.
func binPath(t *testing.T) string {
	t.Helper()

	// 1. Alongside the compiled test binary (useful with -c).
	if self, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(self), "gocker")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// 2. Module root — two directories up from this file.
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	candidate := filepath.Join(root, "gocker")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	t.Skip("gocker binary not found; run 'go build -o gocker .' at the module root first")
	return ""
}

// run executes the gocker binary with the given arguments and returns the
// combined stdout+stderr output and the process exit code.
func run(t *testing.T, args ...string) (out string, code int) {
	t.Helper()
	var buf bytes.Buffer
	cmd := exec.Command(binPath(t), args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return buf.String(), exitErr.ExitCode()
		}
		t.Fatalf("unexpected exec error: %v", err)
	}
	return buf.String(), 0
}

// TestReexec_SimpleOutput verifies that a command run through the re-exec
// path produces the expected output and exits 0.
func TestReexec_SimpleOutput(t *testing.T) {
	t.Parallel()
	out, code := run(t, "run", "--", "echo", "hello from gocker")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", code, out)
	}
	if !strings.Contains(out, "hello from gocker") {
		t.Fatalf("expected output to contain %q\ngot: %q", "hello from gocker", out)
	}
}

// TestReexec_ExitCodePropagation verifies that the child's exit code is
// forwarded correctly through the re-exec and pipe-sync layer.
func TestReexec_ExitCodePropagation(t *testing.T) {
	t.Parallel()
	cases := []int{0, 1, 2, 42, 127}
	for _, want := range cases {
		t.Run(strconv.Itoa(want), func(t *testing.T) {
			t.Parallel()
			_, got := run(t, "run", "--", "sh", "-c", "exit "+strconv.Itoa(want))
			if got != want {
				t.Errorf("exit code: want %d, got %d", want, got)
			}
		})
	}
}

// TestReexec_MultipleArgsForwarded verifies that all extra arguments reach the
// command unchanged.
func TestReexec_MultipleArgsForwarded(t *testing.T) {
	t.Parallel()
	out, code := run(t, "run", "--", "sh", "-c", "echo $((6*7))")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", code, out)
	}
	if strings.TrimSpace(out) != "42" {
		t.Fatalf("expected output %q, got %q", "42", strings.TrimSpace(out))
	}
}

// TestReexec_CommandNotFound verifies that a missing binary returns a non-zero
// exit code rather than hanging or panicking.
func TestReexec_CommandNotFound(t *testing.T) {
	t.Parallel()
	_, code := run(t, "run", "--", "this-binary-does-not-exist-xyz")
	if code == 0 {
		t.Fatal("expected non-zero exit code for a missing command, got 0")
	}
}

// TestReexec_ChildDirectCallRejected verifies that the hidden child subcommand
// cannot be invoked directly — fd 3 is not open in a plain user invocation.
func TestReexec_ChildDirectCallRejected(t *testing.T) {
	t.Parallel()
	_, code := run(t, "child", "echo", "should not execute")
	if code == 0 {
		t.Fatal("expected non-zero exit code when calling child without fd 3 open, got 0")
	}
}

// TestReexec_ContextCancellation verifies that cancelling the context
// terminates the child process promptly.
func TestReexec_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the process even starts

	cmd := exec.CommandContext(ctx, binPath(t), "run", "--", "sleep", "60")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error after context cancellation, got nil")
	}
}
