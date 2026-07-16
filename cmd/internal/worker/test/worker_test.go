package test

import (
	"context"
	"testing"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/worker/decision_engine"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"
)

// MockExecutor allows us to test the Decision Engine -> Executor flow
// without requiring a running Docker daemon in the testing environment.
type MockExecutor struct {
	ExecuteCalled chan bool
}

func (m *MockExecutor) Execute(ctx context.Context, task executor.Task) error {
	m.ExecuteCalled <- true
	time.Sleep(100 * time.Millisecond) // Simulate task execution time
	return nil
}

func TestWorkerPipeline_Integration(t *testing.T) {
	// 1. Initialize the Telemetry Monitor
	monitor, err := telemetry.NewMonitor()
	if err != nil {
		t.Fatalf("Failed to create telemetry monitor: %v", err)
	}

	// 2. Initialize the Decision Engine with thresholds
	engine := &decisionengine.DecisionEngineData{
		ConcurrencyLimit: 2,
		Threshold: telemetry.UsageStats{
			SystemCPUUsage:    99.0, // High threshold to ensure tasks can fit
			SystemMemoryUsage: 99.0,
		},
	}

	// 3. Collect telemetry and update the Decision Engine
	stats, err := monitor.GetUsage()
	if err != nil {
		t.Fatalf("Failed to collect telemetry stats: %v", err)
	}
	engine.UpdateTelemetry(stats)

	// 4. Create a mock task and mock executor
	task := executor.Task{
		ID:    "task-1",
		Type:  executor.TaskIO,
		Image: "alpine",
		Cmd:   []string{"echo", "hello"},
		ResourceRequirement: executor.ResourceRequirement{
			RequiredSystemCPU:    0.1, // 0.1% CPU
			RequiredSystemMemory: 10 * 1024 * 1024, // 10MB RAM in bytes
		},
	}

	mockExec := &MockExecutor{
		ExecuteCalled: make(chan bool, 1),
	}

	// 5. Submit the task through the Decision Engine
	accepted := engine.Submit(task, mockExec)
	if !accepted {
		t.Fatal("Expected task to be accepted by the Decision Engine")
	}

	// Assert that active task counter incremented immediately
	if engine.ActiveTasks != 1 {
		t.Errorf("Expected ActiveTasks to be 1, got %d", engine.ActiveTasks)
	}

	// Wait for the mock executor to receive the task execution signal
	select {
	case <-mockExec.ExecuteCalled:
		// Success: The executor was called asynchronously
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for task execution to start")
	}

	// Wait for task to finish and check counter decrement
	// The mock task sleeps for 100ms, so we poll for a bit
	success := false
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		if engine.ActiveTasks == 0 {
			success = true
			break
		}
	}

	if !success {
		t.Errorf("Expected ActiveTasks to return to 0, got %d", engine.ActiveTasks)
	}
}

func TestWorkerPipeline_Throttling(t *testing.T) {
	// Initialize engine with a strict limit of 1 concurrent task
	engine := &decisionengine.DecisionEngineData{
		ConcurrencyLimit: 1,
	}

	mockExec := &MockExecutor{
		ExecuteCalled: make(chan bool, 2),
	}

	task := executor.Task{ID: "task-1"}

	// 1. First task should be accepted
	if !engine.Submit(task, mockExec) {
		t.Fatal("Expected first task to be accepted")
	}

	// 2. Second task should be rejected immediately because active tasks == limit
	if engine.Submit(task, mockExec) {
		t.Fatal("Expected second task to be rejected due to concurrency limit")
	}

	// Wait for the first task to finish execution
	<-mockExec.ExecuteCalled
	time.Sleep(150 * time.Millisecond) // Wait for task Finished defer to trigger

	// 3. Third task should be accepted again now that the active slot is free
	if !engine.Submit(task, mockExec) {
		t.Fatal("Expected third task to be accepted after first task completed")
	}
}
