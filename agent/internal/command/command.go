package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourorg/inventory-agent/agent/internal/config"
	"github.com/yourorg/inventory-agent/agent/internal/scheduler"
)

type Command struct {
	CommandID    string                 `json:"command_id"`
	Type         string                 `json:"type"`
	Parameters   map[string]interface{} `json:"parameters"`
	IssuedAt     time.Time              `json:"issued_at"`
	TTLSeconds   int                    `json:"ttl_seconds"`
	Status       string                 `json:"status"`
	Result       map[string]interface{} `json:"result,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

type CommandPoller struct {
	config      *config.AgentConfig
	scheduler   *scheduler.Scheduler
	client      *http.Client
	stopChan    chan struct{}
	wg          sync.WaitGroup
	semaphore   chan struct{} // Limit concurrent commands
}

func NewCommandPoller(cfg *config.AgentConfig, sched *scheduler.Scheduler) *CommandPoller {
	return &CommandPoller{
		config:    cfg,
		scheduler: sched,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		stopChan:  make(chan struct{}),
		semaphore: make(chan struct{}, 2), // Max 2 concurrent commands
	}
}

func (cp *CommandPoller) Start(ctx context.Context) {
	cp.wg.Add(1)
	go cp.pollLoop(ctx)
}

func (cp *CommandPoller) Stop() {
	close(cp.stopChan)
	cp.wg.Wait()
}

func (cp *CommandPoller) pollLoop(ctx context.Context) {
	defer cp.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cp.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cp.Poll(ctx); err != nil {
				log.Printf("Command poll failed: %v", err)
			}
		}
	}
}

func (cp *CommandPoller) Poll(ctx context.Context) error {
	if cp.config.APIEndpoint == "" || cp.config.AuthToken == "" {
		return nil // Not configured for cloud mode
	}

	endpoint := fmt.Sprintf("%s/v1/agents/%s/commands", cp.config.APIEndpoint, cp.config.DeviceID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cp.config.AuthToken)

	resp, err := cp.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var commands []Command
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		return fmt.Errorf("failed to decode commands: %w", err)
	}

	// Process commands concurrently with limit
	for _, cmd := range commands {
		select {
		case cp.semaphore <- struct{}{}:
			go cp.processCommand(cmd)
		default:
			log.Printf("Command queue full, skipping command %s", cmd.CommandID)
		}
	}

	return nil
}

func (cp *CommandPoller) processCommand(cmd Command) {
	defer func() { <-cp.semaphore }()

	// Check if expired
	if cmd.IssuedAt.Add(time.Duration(cmd.TTLSeconds) * time.Second).Before(time.Now()) {
		log.Printf("Command %s expired", cmd.CommandID)
		cp.ackCommand(cmd.CommandID, map[string]interface{}{"error": "expired"}, nil)
		return
	}

	// Execute command
	result, err := cp.Execute(cmd)
	if err != nil {
		log.Printf("Command %s execution failed: %v", cmd.CommandID, err)
		cp.ackCommand(cmd.CommandID, map[string]interface{}{"error": err.Error()}, err)
		return
	}

	cp.ackCommand(cmd.CommandID, result, nil)
}

func (cp *CommandPoller) Execute(cmd Command) (map[string]interface{}, error) {
	switch cmd.Type {
	case "collect.now":
		return cp.executeCollectNow(cmd)
	default:
		return nil, fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

func (cp *CommandPoller) executeCollectNow(cmd Command) (map[string]interface{}, error) {
	// Parse parameters
	metrics, ok := cmd.Parameters["metrics"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid metrics parameter")
	}

	// Convert to string slice
	metricNames := make([]string, len(metrics))
	for i, m := range metrics {
		metricNames[i] = fmt.Sprintf("%v", m)
	}

	// Trigger collection for specified metrics
	// Note: This is a simplified implementation
	// In practice, you'd need to modify scheduler to support selective collection
	log.Printf("Executing collect.now for metrics: %v", metricNames)

	// For now, trigger full collection
	if err := cp.scheduler.TriggerNow(); err != nil {
		return nil, fmt.Errorf("collection failed: %w", err)
	}

	return map[string]interface{}{
		"status":  "completed",
		"metrics": metricNames,
	}, nil
}

func (cp *CommandPoller) ackCommand(commandID string, result map[string]interface{}, err error) {
	if cp.config.APIEndpoint == "" || cp.config.AuthToken == "" {
		return
	}

	endpoint := fmt.Sprintf("%s/v1/agents/%s/commands/%s/ack", cp.config.APIEndpoint, cp.config.DeviceID, commandID)

	payload := map[string]interface{}{
		"result": result,
	}
	if err != nil {
		payload["error"] = err.Error()
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal ack payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))
	if err != nil {
		log.Printf("Failed to create ack request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+cp.config.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cp.client.Do(req)
	if err != nil {
		log.Printf("Ack request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Ack request returned status %d", resp.StatusCode)
	}
}