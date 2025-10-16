package collectors

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type SoftwareItem struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
	InstallDate string `json:"install_date"`
}

type SoftwareCollector struct {
	*BaseCollector
}

func NewSoftwareCollector() *SoftwareCollector {
	return &SoftwareCollector{
		BaseCollector: NewBaseCollector("software.inventory", false), // Disabled by default
	}
}

func (c *SoftwareCollector) Collect(ctx context.Context) (interface{}, error) {
	var software []SoftwareItem

	// Query 64-bit registry
	if items, err := c.queryRegistry(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`); err == nil {
		software = append(software, items...)
	}

	// Query 32-bit registry on 64-bit systems
	if items, err := c.queryRegistry(registry.LOCAL_MACHINE,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`); err == nil {
		software = append(software, items...)
	}

	// Remove duplicates and filter system components
	filtered := c.filterSoftware(software)

	return filtered, nil
}

func (c *SoftwareCollector) queryRegistry(root registry.Key, path string) ([]SoftwareItem, error) {
	key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer key.Close()

	subKeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	var software []SoftwareItem
	for _, subKeyName := range subKeys {
		subKey, err := registry.OpenKey(key, subKeyName, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		item := SoftwareItem{}

		// Read DisplayName
		if name, _, err := subKey.GetStringValue("DisplayName"); err == nil {
			item.Name = strings.TrimSpace(name)
		}

		// Read DisplayVersion
		if version, _, err := subKey.GetStringValue("DisplayVersion"); err == nil {
			item.Version = strings.TrimSpace(version)
		}

		// Read Publisher
		if publisher, _, err := subKey.GetStringValue("Publisher"); err == nil {
			item.Publisher = strings.TrimSpace(publisher)
		}

		// Read InstallDate (format: YYYYMMDD)
		if installDate, _, err := subKey.GetStringValue("InstallDate"); err == nil {
			item.InstallDate = formatInstallDate(installDate)
		}

		subKey.Close()

		// Only include if we have at least a name
		if item.Name != "" {
			software = append(software, item)
		}
	}

	return software, nil
}

func (c *SoftwareCollector) filterSoftware(software []SoftwareItem) []SoftwareItem {
	seen := make(map[string]bool)
	var filtered []SoftwareItem

	systemPrefixes := []string{
		"Microsoft",
		"Windows",
		"Hotfix",
		"Update",
		"Security Update",
		"Service Pack",
	}

	for _, item := range software {
		// Skip empty names
		if item.Name == "" {
			continue
		}

		// Skip system components
		isSystem := false
		for _, prefix := range systemPrefixes {
			if strings.HasPrefix(item.Name, prefix) {
				isSystem = true
				break
			}
		}
		if isSystem {
			continue
		}

		// Deduplicate by name
		key := strings.ToLower(item.Name)
		if seen[key] {
			continue
		}
		seen[key] = true

		filtered = append(filtered, item)
	}

	return filtered
}

func formatInstallDate(dateStr string) string {
	if len(dateStr) != 8 {
		return dateStr
	}

	// Convert YYYYMMDD to YYYY-MM-DD
	return fmt.Sprintf("%s-%s-%s", dateStr[:4], dateStr[4:6], dateStr[6:])
}