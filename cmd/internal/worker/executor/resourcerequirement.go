package executor

type ResourceRequirement struct {
	RequiredProcessCPU    float64 `json:"required_process_cpu"`
	RequiredProcessMemory float64 `json:"required_process_memory"`
	RequiredSystemCPU     float64 `json:"required_system_cpu"`
	RequiredSystemMemory  float64 `json:"required_system_memory"`
}
