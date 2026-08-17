package coordinator

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/drumilbhati/swarm/cmd/internal/worker/connection"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/quadtree"
)

type OrbTask struct {
	Task        executor.Task
	SubmittedAt time.Time
}

func (ot OrbTask) Point() orb.Point {
	return orb.Point{
		ot.Task.ResourceRequirement.RequiredSystemCPU,
		ot.Task.ResourceRequirement.RequiredSystemMemory,
	}
}

type Coordinator struct {
	mu          sync.Mutex
	tree        *quadtree.Quadtree
	workers     map[string]time.Time
	workerTasks map[string][]executor.Task
}

func NewCoordinator() *Coordinator {
	maxCPU := 128.0
	maxMemory := 1024.0 * 1024.0 * 1024.0

	if val, err := strconv.ParseFloat(os.Getenv("MAX_CPU"), 64); err == nil && val > 0 {
		maxCPU = val
	}
	if val, err := strconv.ParseFloat(os.Getenv("MAX_MEMORY"), 64); err == nil && val > 0 {
		maxMemory = val
	}

	maxBound := orb.Bound{
		Min: orb.Point{0, 0},
		Max: orb.Point{maxCPU, maxMemory},
	}
	return &Coordinator{
		tree:        quadtree.New(maxBound),
		workers:     make(map[string]time.Time),
		workerTasks: make(map[string][]executor.Task),
	}
}

func (c *Coordinator) MatchTask(workerHeadroom connection.Headroom, workerId string) (executor.Task, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	K := 50

	if val, err := strconv.Atoi(os.Getenv("MAX_TASKS")); err == nil && val > 0 {
		K = val
	}

	queryPoint := orb.Point{workerHeadroom.AvailableSystemCPU, workerHeadroom.AvailableSystemMemory}

	// search in quadtree to find valid matchess
	matches := c.tree.KNearestMatching(
		nil,
		queryPoint,
		K,
		func(p orb.Pointer) bool {
			ot := p.(OrbTask)
			systemFit := ot.Task.ResourceRequirement.RequiredSystemCPU <= workerHeadroom.AvailableSystemCPU &&
				ot.Task.ResourceRequirement.RequiredSystemMemory <= workerHeadroom.AvailableSystemMemory

			processFit := ot.Task.ResourceRequirement.RequiredProcessCPU <= workerHeadroom.AvailableProcessCPU &&
				ot.Task.ResourceRequirement.RequiredProcessMemory <= workerHeadroom.AvailableProcessMemory

			return systemFit && processFit
		},
	)
	if len(matches) == 0 {
		return executor.Task{}, false
	}

	var bestTask *OrbTask
	for _, ptr := range matches {
		ot := ptr.(OrbTask)
		if ot.Task.ResourceRequirement.RequiredProcessCPU <= workerHeadroom.AvailableProcessCPU &&
			ot.Task.ResourceRequirement.RequiredProcessMemory <= workerHeadroom.AvailableProcessMemory {
			// select the oldest timestamp to prevent starvation
			if bestTask == nil || ot.SubmittedAt.Before(bestTask.SubmittedAt) {
				bestTask = &ot
			}
		}
	}

	if bestTask != nil {
		c.tree.Remove(*bestTask, func(p orb.Pointer) bool {
			return p.(OrbTask).Task.ID == bestTask.Task.ID
		})
	}
	if workerId != "" {
		c.workerTasks[workerId] = append(c.workerTasks[workerId], bestTask.Task)
		return bestTask.Task, true
	}
	return executor.Task{}, false
}

func (c *Coordinator) SubmitTask(task executor.Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.tree.Add(OrbTask{
		Task:        task,
		SubmittedAt: time.Now(),
	})
	return err
}

func (c *Coordinator) EvictDeadWorkers(timeout time.Duration) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var evicted []string
	for workerID, lastSeen := range c.workers {
		if now.Sub(lastSeen) > timeout {
			delete(c.workers, workerID)
			evicted = append(evicted, workerID)
		}
	}
	return evicted
}

func (c *Coordinator) StartLivenessSweeper(ctx context.Context, checkInterval time.Duration, livenessTimeout time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			evicted := c.EvictDeadWorkers(livenessTimeout)
			if len(evicted) > 0 {
				c.RetrieveTasks(evicted)
			}
		}
	}
}

func (c *Coordinator) RetrieveTasks(evicted []string) {
	c.mu.Lock()
	var tasksToReque []executor.Task
	for _, id := range evicted {
		tasksToReque = append(tasksToReque, c.workerTasks[id]...)
		delete(c.workerTasks, id)
	}
	c.mu.Unlock()

	for _, t := range tasksToReque {
		_ = c.SubmitTask(t)
	}
}
