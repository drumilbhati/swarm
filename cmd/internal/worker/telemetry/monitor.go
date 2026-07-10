package telemetry

import (
	"os"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

type Monitor struct {
	p *process.Process
}

func NewMonitor() (*Monitor, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}
	return &Monitor{p: p}, nil
}

func (m *Monitor) GetUsage() (UsageStats, error) {
	usage := UsageStats{}

	systemMemory, err := GetSystemMemory()
	if err != nil {
		return UsageStats{}, err
	}
	usage.SystemMemoryUsage = systemMemory.UsedPercent

	systemCPU, err := GetSystemCPU()
	if err != nil {
		return UsageStats{}, err
	}
	usage.SystemCPUUsage = systemCPU[0]

	processMemory, err := GetProcessMemory(m.p)
	if err != nil {
		return UsageStats{}, err
	}
	usage.ProcessMemoryUsage = 100.0 * (float64(processMemory.RSS) / float64(systemMemory.Total))

	processCPU, err := GetProcessCPU(m.p)
	if err != nil {
		return UsageStats{}, err
	}
	usage.ProcessCPUUsage = processCPU

	return usage, nil
}

func GetProcessMemory(p *process.Process) (*process.MemoryInfoStat, error) {
	return p.MemoryInfo()
}

func GetProcessCPU(p *process.Process) (float64, error) {
	return p.CPUPercent()
}

func GetSystemMemory() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

func GetSystemCPU() ([]float64, error) {
	return cpu.Percent(0, false)
}
