package decisionengine

import (
	"context"
	"log"
	"sync"

	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"
)

type DecisionEngineData struct {
	mu               sync.RWMutex
	Threshold        telemetry.UsageStats
	Stats            telemetry.UsageStats
	ActiveTasks      int64
	ConcurrencyLimit int64
}

func (d *DecisionEngineData) UpdateTelemetry(stats telemetry.UsageStats) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Stats = stats
}

func (d *DecisionEngineData) canFit(task executor.Task) bool {
	if d.ActiveTasks >= d.ConcurrencyLimit {
		return false
	}

	if d.Threshold.ProcessCPUUsage > 0 && d.Stats.ProcessCPUUsage+task.ResourceRequirement.RequiredProcessCPU > d.Threshold.ProcessCPUUsage {
		return false
	}

	if d.Threshold.ProcessMemoryUsage > 0 && d.Stats.TotalSystemMemory > 0 {
		processMemPercent := 100.0 * (task.ResourceRequirement.RequiredProcessMemory) / d.Stats.TotalSystemMemory
		if d.Stats.ProcessMemoryUsage+processMemPercent > d.Threshold.ProcessMemoryUsage {
			return false
		}
	}

	if d.Threshold.SystemCPUUsage > 0 && d.Stats.SystemCPUUsage+task.ResourceRequirement.RequiredSystemCPU > d.Threshold.SystemCPUUsage {
		return false
	}

	if d.Threshold.SystemMemoryUsage > 0 && d.Stats.TotalSystemMemory > 0 {
		systemMemPercent := 100.0 * (task.ResourceRequirement.RequiredSystemMemory) / d.Stats.TotalSystemMemory
		if d.Stats.SystemMemoryUsage+systemMemPercent > d.Threshold.SystemMemoryUsage {
			return false
		}
	}

	return true
}

func (d *DecisionEngineData) Submit(task executor.Task, executor executor.Executor) bool {
	d.mu.Lock()
	if !d.canFit(task) {
		d.mu.Unlock()
		return false
	}
	d.ActiveTasks++
	d.mu.Unlock()

	go func() {
		defer d.taskFinished()

		log.Printf("Starting execution of task %s (image: %s)...", task.ID, task.Image)
		err := executor.Execute(context.Background(), task)
		if err != nil {
			log.Printf("Task %s failed: %v", task.ID, err)
		} else {
			log.Printf("Task %s completed successfully", task.ID)
		}
	}()

	return true
}

func (d *DecisionEngineData) taskFinished() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveTasks--
}
