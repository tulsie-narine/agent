package registration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/inventory-agent/agent/internal/capability"
	"github.com/yourorg/inventory-agent/agent/internal/config"
)

type RegistrationRequest struct {
	DeviceID    string                 `json:"device_id"`
	Hostname    string                 `json:"hostname,omitempty"`
	Capabilities []capability.Capability `json:"capabilities"`
	AgentVersion string                 `json:"agent_version"`
}

type RegistrationResponse struct {
	DeviceID   string `json:"device_id"`
	AuthToken  string `json:"auth_token,omitempty"`
	PolicyVersion int   `json:"policy_version"`
}

type Registrar struct {
	config   *config.AgentConfig
	client   *http.Client
	maxRetries int
}

func New(cfg *config.AgentConfig) *Registrar {
	return &Registrar{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: 10,
	}
}

func (r *Registrar) Register(ctx context.Context) error {
	if r.config.APIEndpoint == "" {
		return fmt.Errorf("API endpoint not configured")
	}

	// Generate device ID if not set
	if r.config.DeviceID == "" {
		r.config.DeviceID = uuid.New().String()
		if err := r.config.Save(); err != nil {
			return fmt.Errorf("failed to save device ID: %w", err)
		}
	}

	hostname := "unknown"
	if h, err := os.Hostname(); err == nil {
		hostname = h
	}

	req := RegistrationRequest{
		DeviceID:     r.config.DeviceID,
		Hostname:     hostname,
		Capabilities: capability.GetCapabilities(),
		AgentVersion: "1.0.0",
	}

	var lastErr error
	for attempt := 0; attempt < r.maxRetries; attempt++ {
		if err := r.attemptRegister(ctx, req); err != nil {
			lastErr = err
			log.Printf("Registration attempt %d failed: %v", attempt+1, err)

			// Exponential backoff
			backoff := time.Duration(attempt+1) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		log.Printf("Registration successful for device %s", r.config.DeviceID)
		return nil
	}

	return fmt.Errorf("registration failed after %d attempts: %w", r.maxRetries, lastErr)
}

func (r *Registrar) attemptRegister(ctx context.Context, req RegistrationRequest) error {
	endpoint := fmt.Sprintf("%s/v1/agents/register", r.config.APIEndpoint)

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200, 201:
		var regResp RegistrationResponse
		if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		// Update config with auth token if provided
		if regResp.AuthToken != "" {
			r.config.AuthToken = regResp.AuthToken
			if err := r.config.Save(); err != nil {
				return fmt.Errorf("failed to save auth token: %w", err)
			}
		}

		return nil

	case 409:
		// Device already registered, try to get token via re-registration
		return r.reRegister(ctx, req)

	default:
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}
}

func (r *Registrar) reRegister(ctx context.Context, req RegistrationRequest) error {
	// For re-registration, we might need different logic
	// For now, assume the device is already registered and we have a token
	if r.config.AuthToken != "" {
		return nil // Already have token
	}

	// If no token, this is an error state
	return fmt.Errorf("device appears registered but no auth token available")
}