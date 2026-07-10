# Swarm: Resource-Aware Distributed Task Execution Engine

Swarm is a lightweight, distributed task execution framework designed for orchestrating diverse workloads across transient, cost-optimized compute environments. 

Unlike traditional distributed queues that rely on static concurrency limits, Swarm agents utilize real-time local telemetry to dynamically throttle or scale execution capacity. This ensures optimal hardware utilization, prevents resource thrashing, and eliminates Out-of-Memory (OOM) errors in high-density multi-tenant environments.

---

## Architectural Overview

The system consists of two primary logical components:

1. **Coordinator**: A centralized control plane that aggregates pending tasks, exposes scheduling endpoints, and maintains the global queue state.
2. **Worker**: An independent execution agent running on a compute node. Each worker is composed of four internal decoupled components:
   * **Connection**: Manages network communication and pulls tasks from the Coordinator when capacity is available.
   * **Telemetry**: Measures real-time resource usage metrics (CPU and Memory) at the system or process level.
   * **Executor**: Controls task concurrency boundaries and runs tasks inside insulated execution units.
   * **Decision Engine**: Resolves telemetry metrics against configured thresholds to govern active backpressure and adjust task polling rates.

---

## Key Features

* **Dynamic Telemetry-Driven Backpressure**: Workers dynamically adjust task intake thresholds to maintain consistent resource utilization targets (e.g., 70% CPU and memory limits).
* **Cost-Optimized Footprint**: Executable agents compile into single self-contained binaries with zero external runtime dependencies.
* **Task Heterogeneity Support**: Capable of dynamically scheduling a mix of CPU-bound, memory-intensive, and lightweight I/O workloads safely on the same node.
* **Pull-Based Topology**: Employs an outbound-only connection pattern from workers to the coordinator, simplifying firewall traversal and network security policies.

---

## Directory Structure

```text
├── cmd
│   └── internal
│       └── worker
│           ├── telemetry       # Telemetry metrics collection (gopsutil)
│           └── resourcemonitor # Hardware monitoring wrappers
└── docs
    ├── progress.md             # Development milestone tracker
    └── problem_statement.md    # Formal engine requirements specification
```

---

## Getting Started

Refer to the documents located in the `docs/` directory for detailed design requirements and implementation status:
* Read the [Problem Statement](docs/problem_statement.md) for execution parameters and verification scenarios.
* Track the current development milestones in the [Progress Tracker](docs/progress.md).
