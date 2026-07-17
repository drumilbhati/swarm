package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	decisionengine "github.com/drumilbhati/swarm/cmd/internal/worker/decision_engine"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"
)

type Connection struct {
	coordinatorURLs []string // Changed to slice of URLs for work stealing
	httpClient      *http.Client
	pollInterval    time.Duration

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

func NewConnection(urls []string, poll time.Duration, tel telemetry.Telemetry, dec *decisionengine.DecisionEngineData, exe executor.Executor) *Connection {
	return &Connection{
		coordinatorURLs: urls,
		httpClient:      &http.Client{Timeout: 5 * time.Second},
		pollInterval:    poll,
		telemetry:       tel,
		decision:        dec,
		exec:            exe,
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
		fmt.Printf("Telemetry error: %v\n", err)
		return
	}

	headroom := c.calculateHeadroom(stats)

	task, found, err := c.pollCoordinator(ctx, headroom)
	if err != nil {
		fmt.Printf("Poll error: %v\n", err)
		return
	}
	if !found {
		return
	}

	accepted := c.decision.Submit(task, c.exec)
	if !accepted {
		fmt.Printf("Error: Task submission failed\n")
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

	n := len(c.coordinatorURLs)
	if n == 0 {
		return executor.Task{}, false, fmt.Errorf("no coordinator URLs configured")
	}

	// Choose a random starting offset to distribute initial poll requests evenly
	offset := rand.Intn(n)

	// Work-stealing loop: poll coordinators starting from a random index
	for i := 0; i < n; i++ {
		targetIdx := (offset + i) % n
		coordURL := c.coordinatorURLs[targetIdx]

		url := fmt.Sprintf("%s/tasks/poll", coordURL)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Resilient fallback: log warning and try stealing from next coordinator
			fmt.Printf("Warning: coordinator %s unreachable, attempting steal...\n", coordURL)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNoContent {
			// No tasks fit on this coordinator: try next coordinator
			continue
		}

		if resp.StatusCode != http.StatusOK {
			continue
		}

		var task executor.Task
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			continue
		}

		// Task successfully claimed!
		return task, true, nil
	}

	// Checked all coordinators, none had matching pending tasks
	return executor.Task{}, false, nil
}
