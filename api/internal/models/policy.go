package models

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Policy struct {
	PolicyID   int64                  `json:"policy_id" db:"policy_id"`
	DeviceID   *uuid.UUID             `json:"device_id,omitempty" db:"device_id"`
	GroupID    *int64                 `json:"group_id,omitempty" db:"group_id"`
	Scope      string                 `json:"scope" db:"scope"`
	Version    int                    `json:"version" db:"version"`
	Config     PolicyConfig           `json:"config" db:"config"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	CreatedBy  string                 `json:"created_by" db:"created_by"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

type PolicyConfig struct {
	IntervalSeconds int                    `json:"interval_seconds"`
	Metrics         map[string]MetricConfig `json:"metrics"`
}

type MetricConfig struct {
	Enabled bool `json:"enabled"`
}

func (p *Policy) Validate() error {
	if p.Scope != "global" && p.Scope != "group" && p.Scope != "device" {
		return fmt.Errorf("invalid scope: %s", p.Scope)
	}

	if p.Scope == "device" && p.DeviceID == nil {
		return fmt.Errorf("device_id required for device scope")
	}

	if p.Scope == "group" && p.GroupID == nil {
		return fmt.Errorf("group_id required for group scope")
	}

	if p.Config.IntervalSeconds < 60 || p.Config.IntervalSeconds > 3600 {
		return fmt.Errorf("interval_seconds must be between 60 and 3600")
	}

	return nil
}

func (p *Policy) GenerateETag() string {
	data := fmt.Sprintf("%d-%s-%d", p.PolicyID, p.Scope, p.Version)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf(`"%x"`, hash)
}

func (p *Policy) MatchesDevice(deviceID uuid.UUID, groupID int64) bool {
	switch p.Scope {
	case "global":
		return true
	case "group":
		return p.GroupID != nil && *p.GroupID == groupID
	case "device":
		return p.DeviceID != nil && *p.DeviceID == deviceID
	default:
		return false
	}
}

// ResolveEffectivePolicy returns the effective policy for a device
// Priority: device > group > global
func ResolveEffectivePolicy(policies []Policy, deviceID uuid.UUID, groupID int64) *Policy {
	var global, group, device *Policy

	for i := range policies {
		p := &policies[i]
		if !p.MatchesDevice(deviceID, groupID) {
			continue
		}

		switch p.Scope {
		case "global":
			if global == nil || p.Version > global.Version {
				global = p
			}
		case "group":
			if group == nil || p.Version > group.Version {
				group = p
			}
		case "device":
			if device == nil || p.Version > device.Version {
				device = p
			}
		}
	}

	// Priority: device > group > global
	if device != nil {
		return device
	}
	if group != nil {
		return group
	}
	return global
}

// FilterByCapabilities removes metrics not supported by the agent
func (p *Policy) FilterByCapabilities(capabilities []Capability) {
	if p.Config.Metrics == nil {
		return
	}

	supported := make(map[string]bool)
	for _, cap := range capabilities {
		supported[cap.Name] = true
	}

	for metric := range p.Config.Metrics {
		if !supported[metric] {
			delete(p.Config.Metrics, metric)
		}
	}
}