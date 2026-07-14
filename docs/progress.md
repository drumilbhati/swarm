# Swarm Project Progress Tracker

This document tracks the design, development milestones, and progress of the Swarm resource-aware distributed workload runner.

---

## Today's Achievements (July 14, 2026)
* **Model 2 Decision Flow**: Aligned the Decision Engine as the central dispatcher. It now accepts tasks, checks resource thresholds, and coordinates async task launch.
* **Deadlock Resolution**: Refined the Mutex implementation by eliminating internal re-entrant locking in the private `canFit` helper method.
* **Docker Task Schema**: Created the decoupled `Task` struct containing metadata, command, image, and resource constraints, plus a nested `ResourceRequirement` struct.
* **Executor Interface**: Defined the abstract `Executor` interface that will wrap the Docker SDK execution handler.
* **Docker Executor**: Implemented the concrete `DockerExecutor` using the Docker Go SDK to pull images, create sandboxed containers with cgroup resource limits, start execution, and safely clean up resources on exit.
* **Integrated Compilation**: Wired the `DockerExecutor` to the `DecisionEngine`'s asynchronous dispatch channel and resolved compiler dependency mismatches.

---

## Implementation Roadmap

### Phase 1: Telemetry & Resource Monitoring
* [x] Design raw percentage-based resource stats contract (`UsageStats` & `Telemetry` interface).
* [x] Implement system-wide resource monitoring using `gopsutil`.
* [x] Implement process-level resource monitoring using `os.Getpid()`.
* [x] Verify telemetry output with a mock client script loop.

### Phase 2: Decision Engine (Self-Throttling)
* [x] Define the `DecisionEngine` struct to maintain state (concurrency counters, threshold configs, telemetry metrics).
* [x] Implement thread-safe capacity status checking (`CanAcceptWork() bool`).
* [x] Implement dynamic backpressure rules (slowing down/pausing task ingestion based on resource headroom).

### Phase 3: Task Executor & Concurrency Control
* [x] Design task profiles and nested resource requirement schema (`task.go` & `resourcerequirement.go`).
* [x] Define the abstract `Executor` interface.
* [x] Install Go Docker SDK and implement the concrete `DockerExecutor`.
* [x] Hook the `DockerExecutor` up to the `DecisionEngine` to manage container execution limits.

### Phase 4: Connection & Coordinator (Networking)
* [ ] Build a simple, memory-based HTTP Task Queue (Coordinator server).
* [ ] Implement the worker client polling loop (`Connection` module).
* [ ] Wire up the complete pipeline: `Connection` checks `Decision` -> Pulls Task -> Hands to `Executor` -> Updates `Telemetry`.

---

## Current Status & Next Steps
- **Current Active State**: Telemetry, Decision Engine, and Executor modules are complete and compile successfully. The worker can execute docker containers under cgroup CPU/Memory limits.
- **Up Next**: Phase 4 (Connection & Coordinator). Build the Coordinator HTTP server queue and the worker polling loop.
