package decisionengine

import (
	"sync"

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

func (d *DecisionEngineData) CanAcceptWork() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.ActiveTasks >= d.ConcurrencyLimit {
		return false
	}

	if d.Threshold.ProcessCPUUsage > 0 && d.Stats.ProcessCPUUsage > d.Threshold.ProcessCPUUsage {
		return false
	}

	if d.Threshold.ProcessMemoryUsage > 0 && d.Stats.ProcessMemoryUsage > d.Threshold.ProcessMemoryUsage {
		return false
	}

	if d.Threshold.SystemCPUUsage > 0 && d.Stats.SystemCPUUsage > d.Threshold.SystemCPUUsage {
		return false
	}

	if d.Threshold.SystemMemoryUsage > 0 && d.Stats.SystemMemoryUsage > d.Threshold.SystemMemoryUsage {
		return false
	}

	return true
}

func (d *DecisionEngineData) TaskStarted() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveTasks++
}

func (d *DecisionEngineData) TaskFinished() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveTasks--
}
