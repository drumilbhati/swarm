package connection

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	decisionengine "github.com/drumilbhati/swarm/cmd/internal/worker/decision_engine"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"
)

// MockTelemetry implements telemetry.Telemetry interface
type MockTelemetry struct {
	Stats telemetry.UsageStats
}

func (m *MockTelemetry) GetUsage() (telemetry.UsageStats, error) {
	return m.Stats, nil
}

// MockExecutor implements executor.Executor interface
type MockExecutor struct {
	LastExecutedTask executor.Task
	ExecuteCalled    chan bool
}

func (m *MockExecutor) Execute(ctx context.Context, task executor.Task) error {
	m.LastExecutedTask = task
	m.ExecuteCalled <- true
	return nil
}

func TestConnection_CalculateHeadroom(t *testing.T) {
	dec := &decisionengine.DecisionEngineData{
		Threshold: telemetry.UsageStats{
			SystemCPUUsage:    70.0,
			SystemMemoryUsage: 80.0,
		},
	}

	conn := &Connection{decision: dec}

	// Case 1: System is using 50% CPU and 60% Memory of 16GB Total Memory
	stats := telemetry.UsageStats{
		SystemCPUUsage:    50.0,
		SystemMemoryUsage: 60.0,
		TotalSystemMemory: 16 * 1024 * 1024 * 1024, // 16 GB
	}

	headroom := conn.calculateHeadroom(stats)

	// Available CPU = 70.0 - 50.0 = 20.0
	if headroom.AvailableSystemCPU != 20.0 {
		t.Errorf("Expected AvailableSystemCPU to be 20.0, got %f", headroom.AvailableSystemCPU)
	}

	// Available Memory Percent = 80.0 - 60.0 = 20.0%
	// 20% of 16GB = 3.2GB = 3,435,973,836.8 bytes
	expectedMemory := (16 * 1024 * 1024 * 1024 * 20.0) / 100.0
	if headroom.AvailableSystemMemory != expectedMemory {
		t.Errorf("Expected AvailableSystemMemory to be %f, got %f", expectedMemory, headroom.AvailableSystemMemory)
	}
}

func TestConnection_PollCoordinator_NoContent(t *testing.T) {
	// Start in-memory mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/poll" {
			t.Errorf("Unexpected request path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Unexpected HTTP method: %s", r.Method)
		}

		// Decode the headroom payload sent by the client
		var hr Headroom
		if err := json.NewDecoder(r.Body).Decode(&hr); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Respond with 204 No Content (no tasks fit)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	conn := NewConnection(ts.URL, 100*time.Millisecond, nil, nil, nil)
	headroom := Headroom{
		AvailableSystemCPU:    10.0,
		AvailableSystemMemory: 1024 * 1024,
	}

	task, found, err := conn.pollCoordinator(context.Background(), headroom)
	if err != nil {
		t.Fatalf("pollCoordinator failed: %v", err)
	}
	if found {
		t.Fatal("expected found to be false for StatusNoContent")
	}
	if task.ID != "" {
		t.Errorf("expected empty task, got ID %s", task.ID)
	}
}

func TestConnection_PollCoordinator_TaskFound(t *testing.T) {
	expectedTask := executor.Task{
		ID:    "task-123",
		Image: "ubuntu",
		Cmd:   []string{"echo", "hello"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedTask)
	}))
	defer ts.Close()

	conn := NewConnection(ts.URL, 100*time.Millisecond, nil, nil, nil)
	task, found, err := conn.pollCoordinator(context.Background(), Headroom{})
	if err != nil {
		t.Fatalf("pollCoordinator failed: %v", err)
	}
	if !found {
		t.Fatal("expected found to be true")
	}
	if task.ID != "task-123" || task.Image != "ubuntu" {
		t.Errorf("decoded task mismatch: %+v", task)
	}
}

func TestConnection_PollAndSubmit_Integration(t *testing.T) {
	expectedTask := executor.Task{
		ID:    "task-456",
		Image: "alpine",
		Cmd:   []string{"sleep", "1"},
	}

	// 1. Mock HTTP server returning task
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedTask)
	}))
	defer ts.Close()

	// 2. Mock Telemetry Stats
	mockTel := &MockTelemetry{
		Stats: telemetry.UsageStats{
			SystemCPUUsage:    20.0,
			SystemMemoryUsage: 30.0,
			TotalSystemMemory: 8 * 1024 * 1024 * 1024,
		},
	}

	// 3. Initialize Decision Engine
	dec := &decisionengine.DecisionEngineData{
		ConcurrencyLimit: 2,
		Threshold: telemetry.UsageStats{
			SystemCPUUsage:    80.0,
			SystemMemoryUsage: 80.0,
		},
	}

	// 4. Mock Executor
	mockExec := &MockExecutor{
		ExecuteCalled: make(chan bool, 1),
	}

	// 5. Instantiate Connection Client
	conn := NewConnection(ts.URL, 50*time.Millisecond, mockTel, dec, mockExec)

	// Execute pollAndSubmit once
	conn.pollAndSubmit(context.Background())

	// Assert task was executing in background
	select {
	case <-mockExec.ExecuteCalled:
		// Success: Executor Execute was triggered!
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for mock executor to be called")
	}

	if mockExec.LastExecutedTask.ID != "task-456" {
		t.Errorf("Expected executor to receive task-456, got %s", mockExec.LastExecutedTask.ID)
	}
}
