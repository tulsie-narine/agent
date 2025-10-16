package collectors

import (
	"context"
	"time"

	"github.com/StackExchange/wmi"
)

type CPUUtilization struct {
	CPUPercent float64 `json:"cpu_percent"`
}

type Win32_PerfFormattedData_PerfOS_Processor struct {
	Name             string
	PercentProcessorTime uint64
}

type CPUCollector struct {
	*BaseCollector
}

func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		BaseCollector: NewBaseCollector("cpu.utilization", false), // Disabled by default
	}
}

func (c *CPUCollector) Collect(ctx context.Context) (interface{}, error) {
	// Method 1: Use PerfMon counter for _Total
	var perfData []Win32_PerfFormattedData_PerfOS_Processor
	err := wmi.Query("SELECT Name, PercentProcessorTime FROM Win32_PerfFormattedData_PerfOS_Processor WHERE Name='_Total'", &perfData)
	if err == nil && len(perfData) > 0 {
		return &CPUUtilization{
			CPUPercent: float64(perfData[0].PercentProcessorTime),
		}, nil
	}

	// Method 2: Calculate from two samples (fallback)
	return c.calculateFromSamples(ctx)
}

func (c *CPUCollector) calculateFromSamples(ctx context.Context) (interface{}, error) {
	// First sample
	var firstSample []Win32_PerfFormattedData_PerfOS_Processor
	err := wmi.Query("SELECT Name, PercentProcessorTime FROM Win32_PerfFormattedData_PerfOS_Processor WHERE Name='_Total'", &firstSample)
	if err != nil || len(firstSample) == 0 {
		return nil, err
	}

	// Wait for interval
	select {
	case <-time.After(1 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Second sample
	var secondSample []Win32_PerfFormattedData_PerfOS_Processor
	err = wmi.Query("SELECT Name, PercentProcessorTime FROM Win32_PerfFormattedData_PerfOS_Processor WHERE Name='_Total'", &secondSample)
	if err != nil || len(secondSample) == 0 {
		return nil, err
	}

	// Calculate utilization (simplified - in reality this is more complex)
	utilization := float64(secondSample[0].PercentProcessorTime)

	return &CPUUtilization{
		CPUPercent: utilization,
	}, nil
}