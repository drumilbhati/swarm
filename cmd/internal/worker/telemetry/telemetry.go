package telemetry

type UsageStats struct {
	SystemCPUUsage     float64
	SystemMemoryUsage  float64
	ProcessCPUUsage    float64
	ProcessMemoryUsage float64
}

type Telemetry interface {
	GetUsage() (UsageStats, error)
}
