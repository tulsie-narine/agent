package capability

import (
)

type Capability struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func GetCapabilities() []Capability {
	return []Capability{
		{Name: "os.info", Version: "1.0"},
		{Name: "cpu.utilization", Version: "1.0"},
		{Name: "memory.usage", Version: "1.0"},
		{Name: "disk.utilization", Version: "1.0"},
		{Name: "software.inventory", Version: "1.0"},
	}
}

func GetSupportedMetrics() []string {
	caps := GetCapabilities()
	metrics := make([]string, len(caps))
	for i, cap := range caps {
		metrics[i] = cap.Name
	}
	return metrics
}

func IsSupported(metric string) bool {
	for _, cap := range GetCapabilities() {
		if cap.Name == metric {
			return true
		}
	}
	return false
}