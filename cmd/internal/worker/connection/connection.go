package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	decisionengine "github.com/drumilbhati/swarm/cmd/internal/worker/decision_engine"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"
)

type Connection struct {
	coordinatorURL string
	httpClient     *http.Client
	pollInterval   time.Duration

	telemetry telemetry.Telemetry
	decision  *decisionengine.DecisionEngineData
	exec      executor.Executor
}

type Headroom struct {
	AvailableSystemCPU     float64 `json:"available_system_cpu"`
	AvailableSystemMemory  float64 `json:"available_system_memory"` // in bytes
	AvailableProcessCPU    float64 `json:"available_process_cpu"`
	AvailableProcessMemory float64 `json:"available_process_memory"` // in bytes
}

func NewConnection(url string, poll time.Duration, tel telemetry.Telemetry, dec *decisionengine.DecisionEngineData, exe executor.Executor) *Connection {
	return &Connection{
		coordinatorURL: url,
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		pollInterval:   poll,
		telemetry:      tel,
		decision:       dec,
		exec:           exe,
	}
}

func (c *Connection) Start(ctx context.Context) {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.pollAndSubmit(ctx)
		}
	}
}

func (c *Connection) pollAndSubmit(ctx context.Context) {
	stats, err := c.telemetry.GetUsage()
	if err != nil {
		fmt.Errorf("%v\n", err)
		return
	}

	headroom := c.calculateHeadroom(stats)

	task, found, err := c.pollCoordinator(ctx, headroom)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if !found {
		return
	}

	accepted := c.decision.Submit(task, c.exec)
	if !accepted {
		fmt.Printf("Error: No task found")
	}
}

func (c *Connection) calculateHeadroom(stats telemetry.UsageStats) Headroom {
	thresholds := c.decision.Threshold

	var availableCPU float64
	if thresholds.SystemCPUUsage > 0 {
		availableCPU = thresholds.SystemCPUUsage - stats.SystemCPUUsage
	}

	var availableMemory float64
	if thresholds.SystemMemoryUsage > 0 {
		freePercent := thresholds.SystemMemoryUsage - stats.SystemMemoryUsage
		availableMemory = (stats.TotalSystemMemory * freePercent) / 100.0
	}

	var availableProcessCPU float64
	if thresholds.ProcessCPUUsage > 0 {
		availableProcessCPU = thresholds.ProcessCPUUsage - stats.ProcessCPUUsage
	}

	var availableProcessMemory float64
	if thresholds.ProcessMemoryUsage > 0 {
		freePercent := thresholds.ProcessMemoryUsage - stats.ProcessMemoryUsage
		availableProcessMemory = (stats.TotalSystemMemory * freePercent) / 100.0
	}

	return Headroom{
		AvailableSystemCPU:     availableCPU,
		AvailableSystemMemory:  availableMemory,
		AvailableProcessCPU:    availableProcessCPU,
		AvailableProcessMemory: availableProcessMemory,
	}
}

func (c *Connection) pollCoordinator(ctx context.Context, headroom Headroom) (executor.Task, bool, error) {
	jsonBytes, err := json.Marshal(headroom)
	if err != nil {
		return executor.Task{}, false, err
	}

	url := fmt.Sprintf("%s/tasks/poll", c.coordinatorURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return executor.Task{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return executor.Task{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return executor.Task{}, false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return executor.Task{}, false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task executor.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return executor.Task{}, false, err
	}

	return task, true, nil
}
