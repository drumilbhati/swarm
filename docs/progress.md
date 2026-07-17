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
* [x] Build a simple, memory-based HTTP Task Queue (Coordinator server).
* [x] Implement capacity-aware polling client (`Connection` module) sending available CPU/RAM headroom.
* [x] Wire up the complete pipeline: Connection checks Decision -> Pulls Task -> Hands to Executor -> Updates Telemetry.

### Phase 5: Algorithmic & Scale Optimizations
* [x] Optimize Coordinator task matching from $O(N)$ linear search to $O(\log N)$ logarithmic complexity using a 2D Spatial Quadtree index.
* [x] Implement horizontal Work Stealing / Multi-Coordinator load balancing (dividing lock contention to achieve $5.6\times$ scalability gains).

### Phase 6: Fault Tolerance (Worker Liveness & Rescheduling)
* [ ] **Worker Heartbeat / Keep-Alive Protocol**
  * [ ] Implement periodic background heartbeat sender loop in Worker connection client (`POST /workers/heartbeat`).
  * [ ] Add `/workers/heartbeat` REST API handler in Coordinator controller.
* [ ] **Coordinator Worker Registry**
  * [ ] Implement a thread-safe active worker registry inside the Coordinator.
  * [ ] Track `WorkerID`, `LastSeen` timestamp, and current active task assignments.
* [ ] **Background Liveness Sweeper**
  * [ ] Spawn a background loop (goroutine) in Coordinator on startup to sweep active workers.
  * [ ] Evict workers exceeding the liveness timeout limit (e.g., 10 seconds without a heartbeat).
* [ ] **Automatic Task Rescheduling**
  * [ ] Extract unfinished tasks assigned to the evicted/dead worker.
  * [ ] Re-enqueue the tasks back into the Coordinator's 2D Quadtree queue (`SubmitTask`) for other healthy workers to claim.
  * [ ] Add E2E unit/integration tests verifying that crashed workers trigger automatic job recovery.

---

## Current Status & Next Steps
- **Current Active State**: Core execution pipelines, spatial Quadtree-based task matching, and multi-coordinator work-stealing are fully operational, tested, and documented.
- **Up Next**: Start Phase 6 by implementing the Worker Heartbeat API and Coordinator active worker registry.
