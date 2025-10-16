package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Telemetry struct {
	DeviceID         uuid.UUID              `json:"device_id" db:"device_id"`
	CollectedAt      time.Time              `json:"collected_at" db:"collected_at"`
	Metrics          map[string]interface{} `json:"metrics" db:"metrics"`
	Tags             map[string]string      `json:"tags" db:"tags"`
	Seq              int64                  `json:"seq" db:"seq"`
	ServerReceivedAt time.Time              `json:"server_received_at" db:"server_received_at"`
	IngestionID      uuid.UUID              `json:"ingestion_id" db:"ingestion_id"`
}

// OSInfo represents OS information metrics
type OSInfo struct {
	Caption   string `json:"caption"`
	Version   string `json:"version"`
	Make      string `json:"make"`
	Model     string `json:"model"`
	Serial    string `json:"serial"`
	Hostname  string `json:"hostname"`
	Domain    string `json:"domain"`
	LastUser  string `json:"last_user"`
}

// CPUUtilization represents CPU usage metrics
type CPUUtilization struct {
	CPUPercent float64 `json:"cpu_percent"`
}

// MemoryUsage represents memory usage metrics
type MemoryUsage struct {
	UsedBytes  int64 `json:"used_bytes"`
	TotalBytes int64 `json:"total_bytes"`
}

// DiskUtilization represents disk usage metrics
type DiskUtilization struct {
	Name      string `json:"name"`
	TotalBytes int64  `json:"total_bytes"`
	FreeBytes  int64  `json:"free_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
}

// SoftwareInventory represents installed software
type SoftwareInventory []SoftwareItem

type SoftwareItem struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
	InstallDate string `json:"install_date"`
}

func (t *Telemetry) Validate() error {
	if t.DeviceID == uuid.Nil {
		return fmt.Errorf("device_id is required")
	}

	if t.CollectedAt.IsZero() {
		return fmt.Errorf("collected_at is required")
	}

	if t.CollectedAt.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("collected_at cannot be in the future")
	}

	if t.Metrics == nil {
		return fmt.Errorf("metrics is required")
	}

	// Validate metric structure
	for metricName, metricData := range t.Metrics {
		if err := t.validateMetric(metricName, metricData); err != nil {
			return fmt.Errorf("invalid metric %s: %w", metricName, err)
		}
	}

	return nil
}

func (t *Telemetry) validateMetric(name string, data interface{}) error {
	switch name {
	case "os.info":
		return t.validateOSInfo(data)
	case "cpu.utilization":
		return t.validateCPUUtilization(data)
	case "memory.usage":
		return t.validateMemoryUsage(data)
	case "disk.utilization":
		return t.validateDiskUtilization(data)
	case "software.inventory":
		return t.validateSoftwareInventory(data)
	default:
		return fmt.Errorf("unknown metric: %s", name)
	}
}

func (t *Telemetry) validateOSInfo(data interface{}) error {
	// Basic validation - could be more strict
	_, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("os.info must be an object")
	}
	return nil
}

func (t *Telemetry) validateCPUUtilization(data interface{}) error {
	cpu, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("cpu.utilization must be an object")
	}

	if percent, exists := cpu["cpu_percent"]; exists {
		if _, ok := percent.(float64); !ok {
			return fmt.Errorf("cpu_percent must be a number")
		}
	}

	return nil
}

func (t *Telemetry) validateMemoryUsage(data interface{}) error {
	mem, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("memory.usage must be an object")
	}

	if used, exists := mem["used_bytes"]; exists {
		if _, ok := used.(float64); !ok {
			return fmt.Errorf("used_bytes must be a number")
		}
	}

	if total, exists := mem["total_bytes"]; exists {
		if _, ok := total.(float64); !ok {
			return fmt.Errorf("total_bytes must be a number")
		}
	}

	return nil
}

func (t *Telemetry) validateDiskUtilization(data interface{}) error {
	// Can be single object or array
	switch disks := data.(type) {
	case []interface{}:
		for i, disk := range disks {
			if _, ok := disk.(map[string]interface{}); !ok {
				return fmt.Errorf("disk %d must be an object", i)
			}
		}
	case map[string]interface{}:
		// Single disk
	default:
		return fmt.Errorf("disk.utilization must be an object or array")
	}
	return nil
}

func (t *Telemetry) validateSoftwareInventory(data interface{}) error {
	items, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("software.inventory must be an array")
	}

	for i, item := range items {
		if _, ok := item.(map[string]interface{}); !ok {
			return fmt.Errorf("software item %d must be an object", i)
		}
	}

	return nil
}