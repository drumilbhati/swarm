# Swarm Project Progress Tracker

This document tracks the design, development milestones, and progress of the Swarm resource-aware distributed workload runner.

---

## Today's Achievements (July 10, 2026)
* **API Boundary Definition**: Established the core abstract interface for the worker telemetry module in `telemetry.go`.
* **Telemetry Implementation**: Completed the concrete `gopsutil` monitor in `monitor.go` measuring both system-level and process-specific CPU & Memory usage.
* **Architecture Alignment**: Refined the telemetry implementation to encapsulate PID retrieval (`os.Getpid`) and prevent execution blocks (non-blocking CPU percent querying).

---

## Implementation Roadmap

### Phase 1: Telemetry & Resource Monitoring
* [x] Design raw percentage-based resource stats contract (`UsageStats` & `Telemetry` interface).
* [x] Implement system-wide resource monitoring using `gopsutil`.
* [x] Implement process-level resource monitoring using `os.Getpid()`.
* [ ] Verify telemetry output with a mock client script loop.

### Phase 2: Decision Engine (Self-Throttling)
* [ ] Define the `DecisionEngine` struct to maintain state (concurrency counters, threshold configs, telemetry metrics).
* [ ] Implement thread-safe capacity status checking (`CanAcceptWork() bool`).
* [ ] Implement dynamic backpressure rules (slowing down/pausing task ingestion based on resource headroom).

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
- **Current Active State**: Telemetry module is complete and verified.
- **Up Next**: Begin designing the **Decision Engine** state manager to consume these telemetry values and govern task ingestion thresholds.
