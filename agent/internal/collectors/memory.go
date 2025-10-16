package collectors

import (
	"context"

	"github.com/StackExchange/wmi"
)

type MemoryUsage struct {
	UsedBytes  int64 `json:"used_bytes"`
	TotalBytes int64 `json:"total_bytes"`
}

type Win32_OperatingSystem_Memory struct {
	TotalVisibleMemorySize uint64
	FreePhysicalMemory     uint64
}

type MemoryCollector struct {
	*BaseCollector
}

func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{
		BaseCollector: NewBaseCollector("memory.usage", false), // Disabled by default
	}
}

func (c *MemoryCollector) Collect(ctx context.Context) (interface{}, error) {
	var memData []Win32_OperatingSystem_Memory
	err := wmi.Query("SELECT TotalVisibleMemorySize, FreePhysicalMemory FROM Win32_OperatingSystem", &memData)
	if err != nil || len(memData) == 0 {
		return nil, err
	}

	data := memData[0]
	totalBytes := int64(data.TotalVisibleMemorySize) * 1024  // Convert KB to bytes
	freeBytes := int64(data.FreePhysicalMemory) * 1024
	usedBytes := totalBytes - freeBytes

	return &MemoryUsage{
		UsedBytes:  usedBytes,
		TotalBytes: totalBytes,
	}, nil
}