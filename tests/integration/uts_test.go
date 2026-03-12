//go:build linux

package integration

import (
	"os"
	"strings"
	"testing"
)

// TestUTS_HostnameIsContainerLocal verifies the run path is in a separate UTS
// namespace with a hostname value distinct from the host hostname.
func TestUTS_HostnameIsContainerLocal(t *testing.T) {
	t.Parallel()
	requireRunPathReady(t)

	hostName, err := os.Hostname()
	if err != nil {
		t.Fatalf("failed to read host hostname: %v", err)
	}

	out, code := run(t, "run", "--", "hostname")
	if shouldSkipUTSTest(code, out) {
		t.Skipf("UTS namespace operations are not permitted in this environment: %s", strings.TrimSpace(out))
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", code, out)
	}

	containerName := strings.TrimSpace(out)
	if containerName == "" {
		t.Fatal("expected non-empty hostname output from container")
	}
	if containerName == hostName {
		t.Fatalf("expected container hostname to differ from host hostname %q", hostName)
	}
}

// TestUTS_HostnameChangeDoesNotLeakToHost verifies hostname writes inside the
// container do not mutate the host hostname.
func TestUTS_HostnameChangeDoesNotLeakToHost(t *testing.T) {
	t.Parallel()
	requireRunPathReady(t)

	hostBefore, err := os.Hostname()
	if err != nil {
		t.Fatalf("failed to read host hostname before run: %v", err)
	}

	out, code := run(t, "run", "--", "sh", "-c", "hostname stage3-isolation-check && hostname")
	if shouldSkipUTSTest(code, out) {
		t.Skipf("UTS namespace operations are not permitted in this environment: %s", strings.TrimSpace(out))
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d\noutput: %s", code, out)
	}

	hostAfter, err := os.Hostname()
	if err != nil {
		t.Fatalf("failed to read host hostname after run: %v", err)
	}
	if hostAfter != hostBefore {
		t.Fatalf("host hostname changed unexpectedly: before=%q after=%q", hostBefore, hostAfter)
	}
}
