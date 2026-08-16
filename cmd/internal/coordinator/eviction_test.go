package coordinator

import (
	"context"
	"testing"
	"time"
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

	// Wait for sweeper cycle to trigger
	time.Sleep(100 * time.Millisecond)

	coord.mu.Lock()
	_, exists := coord.workers["worker-timeout"]
	coord.mu.Unlock()

	if exists {
		t.Fatalf("Expected worker-timeout to be evicted by background Liveness Sweeper")
	}
}
