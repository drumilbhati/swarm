# Swarm Project Progress Tracker

This document tracks the design, development milestones, and progress of the Swarm resource-aware distributed workload runner.

---

## Today's Achievements (August 17, 2026)
* **Partition Guard 1 (Worker Self-Termination)**: Implemented 3-consecutive-failure tracking in Worker `heartbeatLoop`. Lost network connectivity triggers local context cancellation and container shutdown before coordinator eviction threshold.
* **Task-Worker Assignment & Automatic Rescheduling**: Implemented `workerTasks` map in Coordinator. Evicted workers trigger `RetrieveTasks` to automatically re-enqueue orphaned tasks back into the 2D Quadtree queue (`SubmitTask`) for healthy workers to claim. Verified via `TestTaskRescheduling_OnWorkerEviction`.
* **Docker Container Execution Optimization**: Implemented `IfNotPresent` local image inspection in `DockerExecutor`, reducing per-container job processing latency from `3.53s` down to **`217ms`** ($16.2\times$ speedup, $4.6$ container jobs/sec).
* **Job Dispatch Benchmark Matrix**: Verified HTTP E2E dispatch latencies under varying queue sizes ($18.6\ \mu\text{s}$ at 1,000 jobs, $410\ \mu\text{s}$ at 1,000,000 jobs).

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
* [x] **Worker Heartbeat / Keep-Alive Protocol**
  * [x] Implement periodic background heartbeat sender loop in Worker connection client (`POST /heartbeat`).
  * [x] Add `/heartbeat` REST API handler in Coordinator controller.
* [x] **Coordinator Worker Registry**
  * [x] Implement a thread-safe active worker registry inside the Coordinator.
  * [x] Track `WorkerID` and `LastSeen` timestamp.
* [x] **Background Liveness Sweeper**
  * [x] Spawn a background loop (goroutine) in Coordinator on startup to sweep active workers.
  * [x] Evict workers exceeding the liveness timeout limit (e.g., 10 seconds without a heartbeat).
* [x] **Partition Guard 1: Worker-Side Self-Termination (Local Suicide Guard)**
  * [x] Track consecutive heartbeat failures in Worker connection client.
  * [x] If 3 consecutive heartbeats fail (>9s network loss), automatically trigger local context cancellation and `docker stop` active containers before coordinator eviction.
* [x] **Automatic Task Rescheduling**
  * [x] Extract unfinished tasks assigned to the evicted/dead worker.
  * [x] Re-enqueue the tasks back into the Coordinator's 2D Quadtree queue (`SubmitTask`) for other healthy workers to claim.
  * [x] Add E2E unit/integration tests verifying that crashed workers trigger automatic job recovery.

### Phase 7: Fairness & Reliable Task Delivery
* [ ] **Partition Guard 2: Coordinator-Side Fencing Tokens & Task Leases**
  * [ ] Attach a versioned `LeaseID` / `FencingToken` to assigned tasks upon dispatch.
  * [ ] Increment `LeaseID` whenever a task is evicted and rescheduled to a new worker.
  * [ ] Make `POST /tasks/complete` reject completion acknowledgements from obsolete lease versions, preventing Zombie worker state corruption.
  * [ ] Add explicit `started`, `completed`, and `failed` acknowledgements from workers.
* [ ] **Starvation Prevention and Aging**
  * [ ] Add an age-based priority component so long-waiting tasks gradually outrank newer tasks with similar resource fit.
  * [ ] Periodically scan older pending tasks outside the capped K-nearest candidate set, preserving efficient normal-path matching while preventing indefinite starvation.
  * [ ] Add configurable maximum wait-time alerts and metrics for pending tasks.
* [ ] **Oversized and Rarely-Fitting Tasks**
  * [ ] Detect tasks that cannot fit any registered worker and expose a clear `unschedulable` status instead of silently retrying forever.
  * [ ] Reserve configurable capacity or provide a separate queue for oversized tasks so small jobs cannot consume all available capacity indefinitely.
  * [ ] Add tests covering CPU-heavy, memory-heavy, and temporarily unschedulable tasks.
* [ ] **Multi-Coordinator Fairness**
  * [ ] Add shared or partition-aware fairness metadata across coordinators.
  * [ ] Prevent work stealing from repeatedly favoring one coordinator or starving tasks isolated on another coordinator.
  * [ ] Add end-to-end tests for fairness across multiple coordinators and workers.

---

## Current Status & Next Steps
- **Current Active State**: Phase 6 (Fault Tolerance, Worker Liveness, Partition Guard 1, and Automatic Task Rescheduling) is 100% complete, fully tested, and benchmarked. Real Docker job processing latency optimized to **217 ms/container** ($16.2\times$ speedup).
- **Up Next**: Phase 7: Fairness & Reliable Task Delivery, starting with **Partition Guard 2: Coordinator-Side Fencing Tokens & Task Leases**.
