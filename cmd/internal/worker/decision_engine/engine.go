package decisionengine

import (
	"github.com/drumilbhati/swarm/cmd/internal/worker/executor"
	"github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"
)

type DecisionEngine interface {
	// Called by Telemetry
	UpdateTelemetry(stats telemetry.UsageStats)

	// Called by Connection loop
	canFit(task executor.Task) bool

	// Called by Executor
	Submit(task executor.Task, executor executor.Executor)
	taskFinished()
}
