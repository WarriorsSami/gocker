package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	builtBinOnce sync.Once
	builtBinPath string
	builtBinErr  error
	runReadyOnce sync.Once
	runReadyErr  error
)

func moduleRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func buildTestBinary() (string, error) {
	builtBinOnce.Do(func() {
		goBin, err := exec.LookPath("go")
		if err != nil {
			builtBinErr = fmt.Errorf("go binary not found in PATH")
			return
		}

		outDir, err := os.MkdirTemp("", "gocker-integration-*")
		if err != nil {
			builtBinErr = fmt.Errorf("failed to create temp dir for gocker build: %w", err)
			return
		}

		builtBinPath = filepath.Join(outDir, "gocker")
		cmd := exec.Command(goBin, "build", "-o", builtBinPath, ".")
		cmd.Dir = moduleRoot()
		if out, err := cmd.CombinedOutput(); err != nil {
			builtBinErr = fmt.Errorf("failed to build gocker test binary: %w\n%s", err, string(out))
			return
		}
	})

	if builtBinErr != nil {
		return "", builtBinErr
	}
	return builtBinPath, nil
}

// binPath locates the gocker binary. It looks for it next to the test binary
// first, then walks up to the module root and checks there.
// If none exists (common in VS Code test runner), it builds one on demand.
func binPath(t *testing.T) string {
	t.Helper()

	if fromEnv := os.Getenv("GOCKER_TEST_BINARY"); fromEnv != "" {
		if _, err := os.Stat(fromEnv); err == nil {
			return fromEnv
		}
	}

	// 1. Alongside the compiled test binary (useful with -c).
	if self, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(self), "gocker")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// 2. Module root — two directories up from this file.
	root := moduleRoot()
	candidate := filepath.Join(root, "gocker")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// 3. Build a throwaway binary for environments that don't run `make build`
	// before invoking tests (e.g. VS Code's Test Explorer).
	built, err := buildTestBinary()
	if err == nil {
		return built
	}

	t.Skipf("gocker binary not available and auto-build failed: %v", err)
	return ""
}

// run executes the gocker binary with the given arguments and returns the
// combined stdout+stderr output and the process exit code.
//
// When the env var GOCKER_TEST_USE_SUDO=1 is set, the binary is invoked via
// "sudo -n" so unprivileged test runners (e.g. VS Code Test Explorer) can
// exercise paths that require CAP_SYS_ADMIN. Requires passwordless sudo for
// the gocker binary (see docs/dev-setup.md or README for instructions).
func run(t *testing.T, args ...string) (out string, code int) {
	t.Helper()
	var buf bytes.Buffer
	var cmd *exec.Cmd
	if os.Getenv("GOCKER_TEST_USE_SUDO") == "1" {
		cmd = exec.Command("sudo", append([]string{"-n", binPath(t)}, args...)...)
	} else {
		cmd = exec.Command(binPath(t), args...)
	}
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

func shouldSkipUTSTest(code int, out string) bool {
	if code == 0 {
		return false
	}
	lower := strings.ToLower(out)
	return strings.Contains(lower, "operation not permitted") ||
		strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "must be root") ||
		strings.Contains(lower, "clone")
}

// requireRunPathReady skips tests that depend on `gocker run` when the host
// kernel/user privileges don't allow creating the UTS namespace.
func requireRunPathReady(t *testing.T) {
	t.Helper()

	runReadyOnce.Do(func() {
		runReadyOut, code := run(t, "run", "--", "true")
		if code == 0 {
			return
		}
		if shouldSkipUTSTest(code, runReadyOut) {
			runReadyErr = fmt.Errorf("%s", strings.TrimSpace(runReadyOut))
			return
		}
		runReadyErr = fmt.Errorf("unexpected run-path failure (code=%d): %s", code, strings.TrimSpace(runReadyOut))
	})

	if runReadyErr == nil {
		return
	}
	if strings.Contains(strings.ToLower(runReadyErr.Error()), "operation not permitted") ||
		strings.Contains(strings.ToLower(runReadyErr.Error()), "permission denied") ||
		strings.Contains(strings.ToLower(runReadyErr.Error()), "must be root") ||
		strings.Contains(strings.ToLower(runReadyErr.Error()), "clone") {
		t.Skipf("skipping test: gocker run is unavailable in this environment: %s", runReadyErr)
	}
	t.Fatalf("cannot execute integration run path: %v", runReadyErr)
}
