# Stage 3 — UTS namespace

## Goal

Give the container its own hostname by placing it in a new UTS namespace, so that changes to `hostname` inside the container have no effect on the host.

## Build

- `internal/runtime/run_host.go` — `RunParent`: set `child.SysProcAttr = &syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWUTS}` to ask the kernel to create a fresh UTS namespace for the child process.
- `internal/runtime/run_host.go` — `RunChild`: call `syscall.Sethostname([]byte("gocker"))` after unblocking from the pipe and before `syscall.Exec`, to write a container-local hostname into the new namespace.

```
gocker run -- hostname
    -> parent forks child with CLONE_NEWUTS
    -> child wakes up inside a new UTS namespace
    -> child calls Sethostname("gocker")
    -> child execs /bin/hostname
    -> prints "gocker"        <-- host hostname is unchanged
```

## Verify

```bash
# Container has its own hostname
go run . run -- hostname                                # prints "gocker"
hostname                                                # still prints the host's original name

# Hostname change inside container does not bleed out
go run . run -- sh -c "hostname newname && hostname"   # prints "newname"
hostname                                                # still prints the host's original name
```

### Acceptance criteria

You can call Stage 3 complete when:

- `gocker run -- hostname` prints `gocker`, not the host's hostname.
- Running `hostname` on the host after the container exits confirms no change occurred.
- The existing integration tests for exit codes, stdio forwarding, and the re-exec guard all still pass.

## Learn

- The **UTS namespace** (`CLONE_NEWUTS`) grants a process its own `nodename` and `domainname` fields from `struct utsname`. It is one of the cheapest namespaces to create.
- `SysProcAttr.Cloneflags` is the Go mechanism for passing namespace flags to the `clone(2)` syscall that underlies `exec.Cmd.Start`.
- `syscall.Sethostname` must be called **after** the `clone` happens (i.e., inside the child) and **before** `execve` replaces the process image. The child re-exec model makes this straightforward: the window between unblocking from the pipe and calling `syscall.Exec` is exactly where all such setup belongs.
- Each additional namespace flag (`CLONE_NEWPID`, `CLONE_NEWNS`, …) is simply OR-ed into `Cloneflags`; the pattern established here is reused for every subsequent stage.
- Creating a new UTS namespace requires `CAP_SYS_ADMIN` on the host, so this stage will fail with a permission error if you try to run it as an unprivileged user.
