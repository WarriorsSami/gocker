# Gocker

![CI](https://github.com/WarriorsSami/gocker/actions/workflows/ci.yml/badge.svg?branch=master)

A minimal Docker-like container runtime built in Go for learning how containers actually work under the hood.

This project is a hands-on exploration of the Linux primitives behind Docker: namespaces, mounts, `chroot`/`pivot_root`, cgroups, root filesystems, image layers, and container process lifecycle management.

The goal is not full Docker compatibility. The goal is to understand the mechanics that make containers possible.

---

## Why this project exists

Containers often feel magical until you realize they are "just" Linux processes with carefully configured isolation and resource controls.

This project exists to make those mechanics concrete by implementing a small container runtime from scratch in Go.

Through this codebase, the runtime is built progressively from first principles:

- spawning and re-executing processes,
- creating isolated namespaces,
- mounting `/proc`,
- switching the root filesystem,
- applying cgroup limits,
- storing container metadata,
- pulling image layers,
- unpacking and running commands from images.

---

## Learning objectives

This project is designed to help you understand:

- how a container differs from a virtual machine,
- how Linux namespaces isolate what a process can see,
- how cgroups constrain what a process can consume,
- how root filesystems make a process believe it has its own `/`,
- how image manifests and layers are fetched and unpacked,
- how a container runtime orchestrates process startup and cleanup.

By the end, you should be able to explain the core ideas behind Docker without treating it as a black box.

---

## Scope

`gocker` is intentionally a learning runtime, not a production container engine.

### In scope

- running a process through a custom CLI,
- child re-exec model,
- UTS namespace isolation,
- PID namespace isolation,
- mount namespace isolation,
- mounting `/proc`,
- root filesystem switching via `chroot` or `pivot_root`,
- basic cgroup v2 limits,
- local container metadata storage,
- image reference parsing,
- pulling image manifests and layers,
- unpacking layers into a runnable root filesystem.

### Out of scope for the first iteration

- full OCI runtime compatibility,
- rootless containers,
- seccomp and capabilities hardening,
- full overlayfs support,
- advanced networking,
- production-grade security.

---

## Project structure

```text
.
├── .github/
│   └── workflows/
│       └── ci.yml
├── cmd/
│   ├── child.go
│   ├── root.go
│   └── run.go
├── docs/
│   ├── stage-01-basic-process-execution.md
│   └── stage-02-child-reexec-model.md
├── internal/
│   └── runtime/
│       └── run_host.go
├── scripts/
│   └── pre-commit
├── tests/
│   └── integration/
│       └── reexec_test.go
├── .golangci.yml
├── go.mod
├── go.sum
├── main.go
├── Makefile
└── README.md
```

### Package overview

#### `cmd`
CLI entrypoint and command definitions. `root.go` sets up the cobra root command; `run.go` implements `gocker run`.

#### `internal/runtime`
Host-side process execution logic. `run_host.go` spawns the child process, forwards stdio, and propagates the exit code.

#### `scripts`
Development tooling scripts. `pre-commit` mirrors the full CI pipeline locally and is installed into `.git/hooks/` via `make install-hooks`.

#### `tests/integration`
End-to-end tests that exercise the compiled gocker binary. Requires `go build -o gocker .` before running.

---

## Architecture overview

At a high level, `gocker run` follows this flow:

1. Parse CLI arguments.
2. Build a runtime spec.
3. Re-exec the current binary into a child mode.
4. Apply namespace isolation through `SysProcAttr`.
5. Inside the child:
   - set hostname,
   - set up mounts,
   - mount `/proc`,
   - switch into the container root filesystem,
   - apply resource constraints,
   - exec the target command.
6. Track container metadata on disk.

Later stages extend that flow by replacing a manually provided rootfs with one built from pulled image layers.

---

## Core concepts behind the implementation

### 1. Namespaces

Namespaces isolate what a process can see.

Examples used in this project:

- UTS namespace for hostname isolation,
- PID namespace for process tree isolation,
- Mount namespace for filesystem mount isolation.

Optionally later:

- NET namespace for networking,
- IPC namespace for inter-process communication,
- USER namespace for rootless-style work.

### 2. Root filesystem isolation

A process only "lives inside a container" once its root filesystem is changed.

This project starts with a simple rootfs-based execution model using `chroot`, with the option to evolve toward `pivot_root` later.

### 3. Cgroups

Cgroups do not hide resources. They limit resource usage.

This project focuses on cgroup v2 to explore memory and CPU constraints in a modern Linux setup.

### 4. Images and layers

A container image is not a virtual machine disk. It is a layered filesystem description.

This project progressively adds:

- image reference parsing,
- registry API access,
- manifest fetching,
- blob download,
- layer unpacking,
- rootfs construction.

---

## Development roadmap

The project is implemented in stages.

- [x] **Stage 1 — Basic process execution**
  - [x] forward stdio
  - [x] propagate exit codes

- [x] **Stage 2 — Child re-exec model**
  - [x] parent starts child through `/proc/self/exe`
  - [x] child performs isolation setup before exec

- [ ] **Stage 3 — UTS namespace** ← next
  - [ ] isolate hostname
  - [ ] verify container-local hostname changes

- [ ] **Stage 4 — PID namespace**
  - [ ] isolate process tree
  - [ ] observe container-local PIDs

- [ ] **Stage 5 — Mount namespace and `/proc`**
  - [ ] create mount isolation
  - [ ] mount `/proc` inside container

- [ ] **Stage 6 — Rootfs execution**
  - [ ] run commands inside a provided root filesystem

- [ ] **Stage 7 — Cgroups v2**
  - [ ] apply memory and CPU-related limits

- [ ] **Stage 8 — Metadata store**
  - [ ] persist basic information for containers and images

- [ ] **Stage 9 — Image pulling**
  - [ ] parse image references
  - [ ] fetch manifests and layer blobs

- [ ] **Stage 10 — Layer unpacking**
  - [ ] unpack layers in order
  - [ ] build a runnable rootfs

- [ ] **Stage 11 — Run from image**
  - [ ] `gocker run alpine:latest /bin/sh`

- [ ] **Stage 12+ — Stretch goals**
  - [ ] whiteouts
  - [ ] bind mounts
  - [ ] signal forwarding
  - [ ] tiny init process
  - [ ] networking
  - [ ] OCI-aligned improvements

---

## Example commands

### Run with a manually provided root filesystem

```bash
gocker run --rootfs ./rootfs /bin/sh
```

### Run with hostname isolation

```bash
gocker run --hostname gockerbox /bin/hostname
```

### Run with memory limit

```bash
gocker run --rootfs ./rootfs --memory 256m /bin/sh
```

### Pull an image

```bash
gocker pull alpine:latest
```

### Run from a pulled image

```bash
gocker run alpine:latest /bin/echo hello
```

---

## Requirements

This project is Linux-specific.

### Runtime requirements

- Linux kernel with namespace support,
- cgroup v2 support,
- Go installed,
- sufficient privileges for namespace, mount, and cgroup operations.

### Notes

Some stages may require root privileges depending on how isolation is implemented.

This project should be treated as a Linux systems learning exercise.

---

## Running the project

```bash
go run ./cmd/gocker --help
```

Example:

```bash
go run ./cmd/gocker run -- echo hello
```

With a root filesystem:

```bash
sudo go run ./cmd/gocker run --rootfs ./rootfs /bin/sh
```

---

## Testing

This project mixes unit tests and Linux-only integration tests.

### Unit tests

Used for:

- CLI parsing,
- runtime spec construction,
- image reference parsing,
- manifest/config decoding,
- metadata persistence,
- layer application logic.

### Integration tests

Used for:

- namespace behavior,
- `/proc` mounting,
- rootfs execution,
- cgroup configuration,
- process lifecycle behavior.

Run tests with:

```bash
go test ./...
```

Linux-only tests should use build tags where appropriate.

---

## Design principles

The codebase follows a few deliberate rules:

- keep kernel-facing logic small and explicit,
- separate configuration from side effects,
- prefer progressive implementation over premature completeness,
- make each stage observable and testable,
- optimize for understanding rather than abstraction for its own sake.

This is a systems-learning project. If a simplification makes the kernel interaction clearer, that simplification is usually worth it.

---

## What this project is not

This is not:

- a drop-in Docker replacement,
- a secure production runtime,
- a full OCI implementation,
- a container orchestration system.

It is a focused educational runtime for learning the mechanics behind containers.

---

## References and inspiration

This project is inspired by the broader "build your own Docker" learning approach, especially:

- Linux namespace and cgroup fundamentals,
- container-from-scratch style demonstrations in Go,
- image-pulling and layer-unpacking runtime challenges,
- the educational breakdown of container internals popularized by container engineering talks and hands-on systems exercises.

---

## Future improvements

Potential next steps after the core runtime works:

- switch from `chroot` to `pivot_root`,
- add signal forwarding and child reaping,
- support readonly rootfs,
- support bind mounts,
- add simple network namespace setup,
- implement whiteout handling properly,
- introduce a content-addressable image store,
- experiment with user namespaces and rootless execution,
- align the runtime more closely with OCI concepts.

---

## Personal learning notes

Each stage is documented as a small lab note in the [`docs/`](docs/) folder.

| Stage | File |
|-------|------|
| 1 — Basic process execution | [docs/stage-01-basic-process-execution.md](docs/stage-01-basic-process-execution.md) |
| 2 — Child re-exec model | [docs/stage-02-child-reexec-model.md](docs/stage-02-child-reexec-model.md) |
| 3 — UTS namespace | [docs/stage-03-uts-namespace.md](docs/stage-03-uts-namespace.md) |

Every note follows the same four-section format: **Goal**, **Build**, **Verify**, **Learn**.

## License

This project is licensed under the terms of the [Apache License 2.0](LICENSE).
