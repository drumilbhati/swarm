package executor

import "context"

type Result struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

type Executor interface {
	Execute(ctx context.Context, task Task) (Result, error)
}
