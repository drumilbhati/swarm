package executor

import "time"

type TaskType string

const (
	TaskCPU    TaskType = "CPU"
	TaskMemory TaskType = "MEMORY"
	TaskIO     TaskType = "IO"
)

type Task struct {
	ID                  string
	Type                TaskType
	Image               string
	Cmd                 []string
	Payload             []byte        // how much memory to allocate, or time to run
	Duration            time.Duration // time the task should run
	ResourceRequirement ResourceRequirement
}
