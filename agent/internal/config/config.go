package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultConfigPath     = `C:\ProgramData\InventoryAgent\config.json`
	DefaultCollectionInterval = 15 * time.Minute
	DefaultLocalOutputPath = `C:\ProgramData\InventoryAgent\inventory.json`
	DefaultLogLevel       = "info"
	DefaultMaxRetries     = 5
	DefaultBackoffMultiplier = 2.0
	DefaultMaxBackoff     = 5 * time.Minute
)

type RetryConfig struct {
	MaxRetries        int           `json:"max_retries"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	MaxBackoff        time.Duration `json:"max_backoff"`
}

type AgentConfig struct {
	DeviceID           string                 `json:"device_id,omitempty"`
	APIEndpoint        string                 `json:"api_endpoint,omitempty"`
	AuthToken          string                 `json:"auth_token,omitempty"`
	CollectionInterval time.Duration          `json:"collection_interval"`
	EnabledMetrics     map[string]bool        `json:"enabled_metrics"`
	LocalOutputPath    string                 `json:"local_output_path"`
	LogLevel           string                 `json:"log_level"`
	RetryConfig        RetryConfig            `json:"retry_config"`
}

// Load reads configuration from file with fallback to defaults
func Load() (*AgentConfig, error) {
	configPath := os.Getenv("AGENT_CONFIG_PATH")
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	cfg := &AgentConfig{
		CollectionInterval: DefaultCollectionInterval,
		EnabledMetrics: map[string]bool{
			"os.info": true, // Always enabled
		},
		LocalOutputPath: DefaultLocalOutputPath,
		LogLevel:        DefaultLogLevel,
		RetryConfig: RetryConfig{
			MaxRetries:        DefaultMaxRetries,
			BackoffMultiplier: DefaultBackoffMultiplier,
			MaxBackoff:        DefaultMaxBackoff,
		},
	}

	// Try to read existing config
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Generate device ID if not set
	if cfg.DeviceID == "" {
		cfg.DeviceID = uuid.New().String()
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("failed to save generated device ID: %w", err)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Save writes configuration to file
func (c *AgentConfig) Save() error {
	configPath := os.Getenv("AGENT_CONFIG_PATH")
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Atomic write
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	if err := os.Rename(tempPath, configPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	return nil
}

// Validate checks configuration for required fields and valid values
func (c *AgentConfig) Validate() error {
	if c.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}

	if c.CollectionInterval < time.Minute {
		return fmt.Errorf("collection_interval must be at least 1 minute")
	}

	if c.LocalOutputPath == "" {
		return fmt.Errorf("local_output_path is required")
	}

	if c.RetryConfig.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}

	if c.RetryConfig.BackoffMultiplier <= 1.0 {
		return fmt.Errorf("backoff_multiplier must be > 1.0")
	}

	if c.RetryConfig.MaxBackoff < time.Second {
		return fmt.Errorf("max_backoff must be at least 1 second")
	}

	return nil
}