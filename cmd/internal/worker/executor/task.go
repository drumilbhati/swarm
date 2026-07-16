package executor

import "time"

type TaskType string

const (
	TaskCPU    TaskType = "CPU"
	TaskMemory TaskType = "MEMORY"
	TaskIO     TaskType = "IO"
)

type Task struct {
	ID                  string              `json:"id"`
	Type                TaskType            `json:"type"`
	Image               string              `json:"image"`
	Cmd                 []string            `json:"cmd"`
	Payload             []byte              `json:"payload"`
	Duration            time.Duration       `json:"duration"`
	ResourceRequirement ResourceRequirement `json:"resource_requirement"`
}
