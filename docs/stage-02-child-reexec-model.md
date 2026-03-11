# Stage 2 — Child re-exec model

## Goal

Empower the parent process to perform necessary setup (e.g., namespace isolation) before executing the target command in the child process using `/proc/self/exe`.

## Build

```
gocker run -- /bin/echo hello
    -> parent process starts
    -> parent re-execs /proc/self/exe as: gocker internal child /bin/echo hello
    -> child process enters internal setup path
    -> child eventually execs the target command
```

## Verify

```bash
go run . run -- echo hello                     # prints "hello", exits 0
go run . run -- /bin/bash                      # interactive shell, exit code propagated
go run . run -- sh -c "exit 42"; echo $status  # prints 42

```

### Acceptance criteria

You can call Stage 2 complete when:

- gocker run -- echo hello still works
- run no longer executes the target directly
- run re-execs the current binary through /proc/self/exe
- internal child receives the forwarded command correctly
- the child executes the target command successfully
- exit codes still propagate correctly

## Learn

This stage lays the groundwork for namespace isolation and further runtime management tasks required for a functional container. 

By leveraging a child re-exec model, we can keep the parent process responsible for setup, coordination and monitoring, much like a side-car supervisor, while the child process can focus on executing the target command within the configured environment. This separation of concerns is crucial for building a robust and extensible container runtime.

By using `/proc/self/exe` to re-exec the binary, we can ensure that the child process starts with a clean slate, allowing us to perform necessary isolation setup (e.g., unsharing namespaces) before executing the target command. This approach also simplifies the overall architecture and makes it easier to reason about the flow of execution within the runtime.

Also, the child process can only be invoked internally by the parent process using a pipe, so we can keep it hidden from users and prevent accidental invocation. This allows us to maintain a clean and user-friendly CLI while still providing the necessary functionality for our runtime.

Here is how the flow of execution looks like in more detail:
1. The user runs `gocker run -- <cmd> [args...]`.
2. The parent process starts and parses the command-line arguments.
3. The parent process re-execs itself using `/proc/self/exe` with a special argument (e.g., `internal child`) to indicate that it should run the child command instead of the normal CLI logic.
4. The parent process sets up a pipe to forward the target command and its arguments to the child process. Keep in mind that the read end of the pipe is passed to the child process as an extra file descriptor (e.g., fd 3) as a sync mechanism to ensure the child can only be invoked by the parent. The write end of the pipe being closed will signal the child to proceed with execution.
5. The parent process starts the child process concurrently using `Start` instead of `Run` to avoid blocking and to allow the parent to perform any necessary monitoring or coordination tasks while the child is running, e.g. setting the cgroups limits.
6. The parent process waits for the child process to finish using `Wait` and extracts the exit code to propagate it back to the user.
7. The child process receives the forwarded command and its arguments through the pipe, performs any necessary isolation setup (e.g., unsharing namespaces), and then execs the target command. We use the `execve` syscall directly in the child process to replace its own image with the target command, ensuring that the target command runs in the same process and inherits any isolation setup performed by the child.
8. The child process waits for the target command to finish and propagates the exit code back to the parent using `os.Exit`.