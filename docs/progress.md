# Swarm Project Progress Tracker

This document tracks the design, development milestones, and progress of the Swarm resource-aware distributed workload runner.

---

## Today's Achievements (July 13, 2026)
* **Decision Engine Implementation**: Completed the concrete `DecisionEngineData` structure in `decisionengine.go` with explicit threshold checking.
* **Thread Safety**: Integrated `sync.RWMutex` to secure telemetry updates, task counters, and capacity checks across multiple goroutines.
* **Counter Corrections**: Resolved inverted limits and decrement/increment logic to prevent active task count drift.

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
* [ ] Design mock task profiles (CPU-bound spin, memory-bound allocator, I/O sleeping task).
* [ ] Implement worker-side concurrency slot allocation.
* [ ] Connect the `Executor` to the `DecisionEngine` so that active tasks decrement/increment capacity dynamically.

### Phase 4: Connection & Coordinator (Networking)
* [ ] Build a simple, memory-based HTTP Task Queue (Coordinator server).
* [ ] Implement the worker client polling loop (`Connection` module).
* [ ] Wire up the complete pipeline: `Connection` checks `Decision` -> Pulls Task -> Hands to `Executor` -> Updates `Telemetry`.

---

## Current Status & Next Steps
- **Current Active State**: Telemetry and Decision Engine modules are complete and verified by unit tests.
- **Up Next**: Begin designing the **Task Executor** to manage slot allocation and coordinate concurrent task loops.
