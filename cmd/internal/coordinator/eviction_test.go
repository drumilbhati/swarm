package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/worker/connection"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
)

func TestEvictDeadWorkers(t *testing.T) {
	coord := NewCoordinator()

	now := time.Now()

	// 1. Register active worker (last seen just now)
	coord.workers["active-worker"] = now

	// 2. Register stale worker (last seen 15 seconds ago)
	coord.workers["stale-worker"] = now.Add(-15 * time.Second)

	// Run eviction with 10s timeout threshold
	evicted := coord.EvictDeadWorkers(10 * time.Second)

	// Assert stale worker was evicted
	if len(evicted) != 1 || evicted[0] != "stale-worker" {
		t.Fatalf("Expected stale-worker to be evicted, got %v", evicted)
	}

	// Assert active worker remains in map
	coord.mu.Lock()
	_, activeExists := coord.workers["active-worker"]
	_, staleExists := coord.workers["stale-worker"]
	coord.mu.Unlock()

	if !activeExists {
		t.Errorf("Expected active-worker to remain in registry")
	}
	if staleExists {
		t.Errorf("Expected stale-worker to be removed from registry")
	}
}

func TestStartLivenessSweeper_Integration(t *testing.T) {
	coord := NewCoordinator()

	// Register a worker with timestamp 200ms in the past
	coord.workers["worker-timeout"] = time.Now().Add(-200 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start sweeper checking every 30ms with 100ms timeout limit
	go coord.StartLivenessSweeper(ctx, 30*time.Millisecond, 100*time.Millisecond)

	// Bounded polling for missing worker eviction until deadline
	deadline := time.Now().Add(1 * time.Second)
	evicted := false

	for time.Now().Before(deadline) {
		coord.mu.Lock()
		_, exists := coord.workers["worker-timeout"]
		coord.mu.Unlock()
		if !exists {
			evicted = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !evicted {
		t.Fatalf("Expected worker-timeout to be evicted by background Liveness Sweeper")
	}
}

func TestTaskRescheduling_OnWorkerEviction(t *testing.T) {
	coord := NewCoordinator()

	// 1. Submit a task
	task := executor.Task{
		ID:    "task-reschedule-101",
		Image: "alpine",
		ResourceRequirement: executor.ResourceRequirement{
			RequiredSystemCPU:    1.0,
			RequiredSystemMemory: 100 * 1024 * 1024,
		},
	}
	if err := coord.SubmitTask(task); err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	headroom := connection.Headroom{
		AvailableSystemCPU:     2.0,
		AvailableSystemMemory:  512 * 1024 * 1024,
		AvailableProcessCPU:    1.0,
		AvailableProcessMemory: 200 * 1024 * 1024,
	}

	// 2. Worker-1 claims the task
	claimedTask, found := coord.MatchTask(headroom, "worker-1")
	if !found || claimedTask.ID != "task-reschedule-101" {
		t.Fatalf("Expected worker-1 to claim task-reschedule-101, got found=%v, task=%+v", found, claimedTask)
	}

	// Assert task is tracked in workerTasks["worker-1"]
	coord.mu.Lock()
	assignedTasks := coord.workerTasks["worker-1"]
	coord.mu.Unlock()
	if len(assignedTasks) != 1 || assignedTasks[0].ID != "task-reschedule-101" {
		t.Fatalf("Expected worker-1 to have 1 assigned task, got %v", assignedTasks)
	}

	// 3. Simulate worker-1 timeout and eviction
	coord.workers["worker-1"] = time.Now().Add(-15 * time.Second)
	evicted := coord.EvictDeadWorkers(10 * time.Second)
	if len(evicted) != 1 || evicted[0] != "worker-1" {
		t.Fatalf("Expected worker-1 to be evicted, got %v", evicted)
	}

	// Retrieve tasks for evicted worker
	coord.RetrieveTasks(evicted)

	// 4. Assert task was re-enqueued back to Quadtree and can be claimed by worker-2
	reclaimedTask, found2 := coord.MatchTask(headroom, "worker-2")
	if !found2 || reclaimedTask.ID != "task-reschedule-101" {
		t.Fatalf("Expected worker-2 to re-claim task-reschedule-101 after worker-1 eviction, got found=%v, task=%+v", found2, reclaimedTask)
	}
}
