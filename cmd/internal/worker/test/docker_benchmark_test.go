package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
)

func TestRealDockerTask_Execution(t *testing.T) {
	// 1. Instantiate real Docker SDK Executor
	dockerExec, err := executor.NewDockerExecutor()
	if err != nil {
		t.Skipf("Skipping real Docker test: Docker daemon not available or unreachable: %v", err)
	}

	// 2. Define real containerized task with cgroup NanoCPUs & Memory limits
	task := executor.Task{
		ID:    "real-docker-task-1",
		Type:  executor.TaskIO,
		Image: "alpine:latest",
		Cmd:   []string{"echo", "Swarm Real Docker Task Success!"},
		ResourceRequirement: executor.ResourceRequirement{
			RequiredSystemCPU:    0.2,               // 0.2 CPU Cores
			RequiredSystemMemory: 64 * 1024 * 1024, // 64 MB RAM limit
		},
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 3. Execute container via Docker Go SDK
	err = dockerExec.Execute(ctx, task)
	if err != nil {
		t.Fatalf("Real Docker task execution failed: %v", err)
	}

	elapsed := time.Since(start)
	t.Logf("Successfully executed real Docker task (%s) in %v", task.Image, elapsed)
}

func BenchmarkRealDockerTask_Execution(b *testing.B) {
	dockerExec, err := executor.NewDockerExecutor()
	if err != nil {
		b.Skipf("Skipping real Docker benchmark: Docker daemon not available: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task := executor.Task{
			ID:    fmt.Sprintf("bench-docker-task-%d", i),
			Type:  executor.TaskIO,
			Image: "alpine:latest",
			Cmd:   []string{"sh", "-c", "echo benchmark"},
			ResourceRequirement: executor.ResourceRequirement{
				RequiredSystemCPU:    0.1,
				RequiredSystemMemory: 32 * 1024 * 1024,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := dockerExec.Execute(ctx, task); err != nil {
			cancel()
			b.Fatalf("Docker task execution failed at iter %d: %v", i, err)
		}
		cancel()
	}
}
