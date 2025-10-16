package collectors

import (
	"context"

	"github.com/StackExchange/wmi"
)

type DiskUtilization struct {
	Name      string `json:"name"`
	TotalBytes int64  `json:"total_bytes"`
	FreeBytes  int64  `json:"free_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
}

type Win32_LogicalDisk struct {
	DeviceID  string
	DriveType uint32
	Size      uint64
	FreeSpace uint64
}

type DiskCollector struct {
	*BaseCollector
}

func NewDiskCollector() *DiskCollector {
	return &DiskCollector{
		BaseCollector: NewBaseCollector("disk.utilization", false), // Disabled by default
	}
}

func (c *DiskCollector) Collect(ctx context.Context) (interface{}, error) {
	var diskData []Win32_LogicalDisk
	// DriveType=3 means local disk
	err := wmi.Query("SELECT DeviceID, DriveType, Size, FreeSpace FROM Win32_LogicalDisk WHERE DriveType=3", &diskData)
	if err != nil {
		return nil, err
	}

	var disks []DiskUtilization
	for _, disk := range diskData {
		// Skip drives with zero size (removable media, etc.)
		if disk.Size == 0 {
			continue
		}

		totalBytes := int64(disk.Size)
		freeBytes := int64(disk.FreeSpace)
		usedBytes := totalBytes - freeBytes

		disks = append(disks, DiskUtilization{
			Name:       disk.DeviceID,
			TotalBytes: totalBytes,
			FreeBytes:  freeBytes,
			UsedBytes:  usedBytes,
		})
	}

	return disks, nil
}