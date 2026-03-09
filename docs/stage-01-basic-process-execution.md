# Stage 1 — Basic process execution

## Goal

Prove that `gocker run` can spawn an arbitrary command, forward stdio, and propagate the exit code — nothing more.

## Build

- `cmd/run.go` — cobra `run` subcommand; accepts `-- <cmd> [args...]`, wires `Stdin`/`Stdout`/`Stderr` to the child.
- `internal/runtime/run_host.go` — `RunHost(ctx, req)`: wraps `exec.CommandContext`, runs the child, extracts `syscall.WaitStatus` to forward the exact exit code via `os.Exit`.

## Verify

```bash
go run . run -- echo hello                     # prints "hello", exits 0
go run . run -- /bin/bash                      # interactive shell, exit code propagated
go run . run -- sh -c "exit 42"; echo $status  # prints 42
```

## Learn

- `exec.ExitError.Sys().(syscall.WaitStatus).ExitStatus()` is the correct way to read the raw exit code in Go.
- Forwarding stdio is trivial but essential — without it interactive commands are unusable.
- The child runs in the **same** namespaces as the host; there is no isolation yet.
- Natural next step: re-exec the binary itself as the child so namespace setup can happen before the target command runs.
