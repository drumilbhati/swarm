package coordinator

import (
	"sync"

	"github.com/drumilbhati/swarm/cmd/internal/worker/connection"
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
)

type Coordinator struct {
	mu    sync.Mutex
	queue []executor.Task
}

func NewCoordinator() *Coordinator {
	return &Coordinator{}
}

func (c *Coordinator) MatchTask(workerHeadroom connection.Headroom) (executor.Task, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, task := range c.queue {
		if task.ResourceRequirement.RequiredSystemCPU <= workerHeadroom.AvailableSystemCPU &&
			task.ResourceRequirement.RequiredSystemMemory <= workerHeadroom.AvailableSystemMemory &&
			task.ResourceRequirement.RequiredProcessCPU <= workerHeadroom.AvailableProcessCPU &&
			task.ResourceRequirement.RequiredProcessMemory <= workerHeadroom.AvailableProcessMemory {
			c.queue = append(c.queue[:i], c.queue[i+1:]...)
			return task, true
		}
	}
	return executor.Task{}, false
}

func (c *Coordinator) SubmitTask(task executor.Task) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queue = append(c.queue, task)
}
