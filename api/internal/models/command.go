package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Command struct {
	CommandID   uuid.UUID              `json:"command_id" db:"command_id"`
	DeviceID    uuid.UUID              `json:"device_id" db:"device_id"`
	Type        string                 `json:"type" db:"type"`
	Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
	IssuedAt    time.Time              `json:"issued_at" db:"issued_at"`
	TTLSeconds  int                    `json:"ttl_seconds" db:"ttl_seconds"`
	Status      string                 `json:"status" db:"status"`
	Result      map[string]interface{} `json:"result" db:"result"`
	CompletedAt *time.Time             `json:"completed_at" db:"completed_at"`
}

func (c *Command) IsExpired() bool {
	expiration := c.IssuedAt.Add(time.Duration(c.TTLSeconds) * time.Second)
	return time.Now().After(expiration)
}

func (c *Command) MarkExecuting() {
	c.Status = "executing"
}

func (c *Command) MarkCompleted(result map[string]interface{}) {
	c.Status = "completed"
	c.Result = result
	now := time.Now()
	c.CompletedAt = &now
}

func (c *Command) MarkFailed(result map[string]interface{}) {
	c.Status = "failed"
	c.Result = result
	now := time.Now()
	c.CompletedAt = &now
}

func (c *Command) MarkExpired() {
	c.Status = "expired"
}

func (c *Command) Validate() error {
	if c.DeviceID == uuid.Nil {
		return fmt.Errorf("device_id is required")
	}

	if c.Type == "" {
		return fmt.Errorf("type is required")
	}

	if c.TTLSeconds <= 0 {
		return fmt.Errorf("ttl_seconds must be positive")
	}

	if c.TTLSeconds > 3600 {
		return fmt.Errorf("ttl_seconds cannot exceed 3600")
	}

	return nil
}