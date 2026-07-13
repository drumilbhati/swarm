package decisionengine

import "github.com/drumilbhati/swarm/cmd/internal/worker/telemetry"

type DecisionEngine interface {
	// Called by Telemetry
	UpdateTelemetry(stats telemetry.UsageStats)

	// Called by Connection loop
	CanAcceptWork() bool

	// Called by Executor
	TaskStarted()
	TaskFinished()
}
