package models

import (
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	DeviceID       uuid.UUID              `json:"device_id" db:"device_id"`
	OrgID          int64                  `json:"org_id" db:"org_id"`
	Hostname       string                 `json:"hostname" db:"hostname"`
	Status         string                 `json:"status" db:"status"`
	Capabilities   []Capability           `json:"capabilities" db:"capabilities"`
	FirstSeenAt    time.Time              `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt     time.Time              `json:"last_seen_at" db:"last_seen_at"`
	AuthTokenHash  string                 `json:"-" db:"auth_token_hash"`
	AgentVersion   string                 `json:"agent_version" db:"agent_version"`
	Meta           map[string]interface{} `json:"meta" db:"meta"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

type Capability struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (a *Agent) IsActive() bool {
	return a.Status == "active"
}

func (a *Agent) HasCapability(name string) bool {
	for _, cap := range a.Capabilities {
		if cap.Name == name {
			return true
		}
	}
	return false
}