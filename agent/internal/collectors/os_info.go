package collectors

import (
	"context"
	"os"
	"strings"

	"github.com/StackExchange/wmi"
	"golang.org/x/sys/windows/registry"
)

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

type Win32_OperatingSystem struct {
	Caption           string
	Version           string
	TotalVisibleMemorySize uint64
	FreePhysicalMemory     uint64
}

type Win32_ComputerSystem struct {
	Manufacturer string
	Model        string
	UserName     string
	Domain       string
}

type Win32_BIOS struct {
	SerialNumber string
}

type OSInfoCollector struct {
	*BaseCollector
}

func NewOSInfoCollector() *OSInfoCollector {
	return &OSInfoCollector{
		BaseCollector: NewBaseCollector("os.info", true), // Always enabled
	}
}

func (c *OSInfoCollector) Collect(ctx context.Context) (interface{}, error) {
	info := &OSInfo{}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}

	// Query WMI for OS information
	var osInfo []Win32_OperatingSystem
	err = wmi.Query("SELECT Caption, Version FROM Win32_OperatingSystem", &osInfo)
	if err == nil && len(osInfo) > 0 {
		info.Caption = strings.TrimSpace(osInfo[0].Caption)
		info.Version = strings.TrimSpace(osInfo[0].Version)
	}

	// Query WMI for computer system information
	var csInfo []Win32_ComputerSystem
	err = wmi.Query("SELECT Manufacturer, Model, UserName, Domain FROM Win32_ComputerSystem", &csInfo)
	if err == nil && len(csInfo) > 0 {
		info.Make = strings.TrimSpace(csInfo[0].Manufacturer)
		info.Model = strings.TrimSpace(csInfo[0].Model)
		info.Domain = strings.TrimSpace(csInfo[0].Domain)
		if csInfo[0].UserName != "" {
			info.LastUser = strings.TrimSpace(csInfo[0].UserName)
		}
	}

	// Query WMI for BIOS serial number
	var biosInfo []Win32_BIOS
	err = wmi.Query("SELECT SerialNumber FROM Win32_BIOS", &biosInfo)
	if err == nil && len(biosInfo) > 0 {
		info.Serial = strings.TrimSpace(biosInfo[0].SerialNumber)
	}

	// Fallback: try to get last logged in user from registry
	if info.LastUser == "" {
		info.LastUser = getLastLoggedInUser()
	}

	return info, nil
}

func getLastLoggedInUser() string {
	// Try to read from registry
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\LogonUI`,
		registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()

	lastUser, _, err := key.GetStringValue("LastLoggedOnUser")
	if err != nil {
		return ""
	}

	// Extract username from domain\user format
	if idx := strings.LastIndex(lastUser, "\\"); idx >= 0 {
		return lastUser[idx+1:]
	}
	return lastUser
}