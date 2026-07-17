package coordinator

import (
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
	mu   sync.Mutex
	tree *quadtree.Quadtree
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
		tree: quadtree.New(maxBound),
	}
}

func (c *Coordinator) MatchTask(workerHeadroom connection.Headroom) (executor.Task, bool) {
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
