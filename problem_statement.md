# Swarm: Resource-Aware Distributed Workload Runner

## Context
In distributed task execution, workloads are rarely uniform. Some tasks are highly CPU-bound, some are memory-intensive, and others are lightweight or I/O-bound. 

Using a static concurrency limit (e.g., "always run 10 tasks") is inefficient:
* If workers pull multiple heavy tasks, they risk **Out-of-Memory (OOM) crashes** or CPU thrashing.
* If workers pull only light tasks, they leave valuable compute resources idle.

To maximize cost-efficiency and performance, we need a system where workers dynamically adjust their task ingestion rate based on their own real-time local resource utilization.

---

## Core Problem
Design and implement a distributed runner system where worker processes dynamically pull and execute heterogeneous tasks from a centralized queue. The workers must adjust their concurrent execution capacity in real time to maintain a target system resource utilization (e.g., 70% CPU and Memory) without exceeding limits.

---

## Initial Stage Requirements

### 1. Resource-Aware Task Ingestion (The Primary Feature)
* **Local Telemetry**: Each worker process must periodically monitor its own system (or process) CPU and Memory usage.
* **Dynamic Concurrency Control**: 
  * The worker must define a **Target Resource Threshold** (e.g., 70% CPU / 70% Memory limit).
  * If resource usage is below the target, the worker should pull more tasks from the queue.
  * If resource usage is near or above the target, the worker must temporarily stop pulling tasks, allowing currently running tasks to finish and resource usage to normalize.

### 2. Task Heterogeneity
* The task queue must support tasks with different resource profiles. To test the system, define mock tasks that simulate resource usage:
  * **CPU-bound task**: Spins the CPU for $N$ seconds.
  * **Memory-bound task**: Allocates and holds $M$ MB of memory for $N$ seconds.
  * **I/O-bound task**: Sleeps for $N$ seconds (minimal CPU/Memory usage).

### 3. Distributed Process Architecture
* **Decoupled Queue/Coordinator**: Runs as a separate process holding the task queue.
* **Worker Process**: Runs as one or more independent processes that communicate with the queue over a network protocol (e.g., HTTP, TCP, or Unix sockets).
* **Pull-Based**: Workers request tasks from the queue when they determine they have resource headroom.

---

## Non-Requirements for Initial Stage (Deferred Complexity)
* **Task Rescheduling on Worker Crash**: You do not need to handle task recovery if a worker is abruptly killed.
* **Heartbeats & Liveness Tracking**: The coordinator does not need to monitor worker health or disconnects.
* **Persistent State**: The task queue can live entirely in-memory on the coordinator.
* **Security & Authentication**: Communication between workers and the coordinator can be unencrypted and unauthenticated.

---

## Verification Scenario
To validate your implementation:

1. **Workload**: Enqueue a random mix of 100 tasks (some CPU-intensive, some memory-intensive, some sleeping).
2. **Worker Setup**: Start a worker process with a target resource threshold set to **60% CPU** and **60% Memory**.
3. **Execution**: Start the worker and monitor its resource usage.
4. **Assertion**:
   * The worker must process the entire workload without crashing due to OOM.
   * During execution, the worker's CPU and Memory utilization should hover near the 60% target, never triggering system resource exhaustion.
   * Concurrency (number of active goroutines) must fluctuate dynamically (e.g., running many sleep tasks simultaneously, but only a few memory/CPU tasks).