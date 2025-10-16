package policy

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourorg/inventory-agent/agent/internal/config"
	"github.com/yourorg/inventory-agent/agent/internal/scheduler"
)

type Policy struct {
	Version        int                    `json:"version"`
	Collect        CollectConfig          `json:"collect"`
}

type CollectConfig struct {
	IntervalSeconds int                    `json:"interval_seconds"`
	Metrics         map[string]MetricConfig `json:"metrics"`
}

type MetricConfig struct {
	Enabled bool `json:"enabled"`
}

type PolicyManager struct {
	config      *config.AgentConfig
	scheduler   *scheduler.Scheduler
	currentPolicy *Policy
	etag         string
	pollInterval time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
}

func NewPolicyManager(cfg *config.AgentConfig, sched *scheduler.Scheduler) *PolicyManager {
	return &PolicyManager{
		config:       cfg,
		scheduler:    sched,
		pollInterval: 60 * time.Second,
		stopChan:     make(chan struct{}),
	}
}

func (pm *PolicyManager) Start(ctx context.Context) {
	pm.wg.Add(1)
	go pm.pollLoop(ctx)
}

func (pm *PolicyManager) Stop() {
	close(pm.stopChan)
	pm.wg.Wait()
}

func (pm *PolicyManager) pollLoop(ctx context.Context) {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := pm.FetchPolicy(ctx); err != nil {
				log.Printf("Policy fetch failed: %v", err)
			}
		}
	}
}

func (pm *PolicyManager) FetchPolicy(ctx context.Context) error {
	if pm.config.APIEndpoint == "" || pm.config.AuthToken == "" {
		return fmt.Errorf("API endpoint or auth token not configured")
	}

	endpoint := fmt.Sprintf("%s/v1/agents/%s/policy", pm.config.APIEndpoint, pm.config.DeviceID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+pm.config.AuthToken)
	if pm.etag != "" {
		req.Header.Set("If-None-Match", pm.etag)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		// New policy
		var policy Policy
		if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
			return fmt.Errorf("failed to decode policy: %w", err)
		}

		// Update ETag
		if etag := resp.Header.Get("ETag"); etag != "" {
			pm.etag = etag
		} else {
			// Generate ETag from policy content
			data, _ := json.Marshal(policy)
			hash := md5.Sum(data)
			pm.etag = `"` + hex.EncodeToString(hash[:]) + `"`
		}

		return pm.ApplyPolicy(&policy)

	case 304:
		// Not modified
		return nil

	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func (pm *PolicyManager) ApplyPolicy(policy *Policy) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Update scheduler interval
	if policy.Collect.IntervalSeconds > 0 {
		interval := time.Duration(policy.Collect.IntervalSeconds) * time.Second
		pm.scheduler.UpdateInterval(interval)

		// Update config
		pm.config.CollectionInterval = interval
		if err := pm.config.Save(); err != nil {
			log.Printf("Failed to save updated config: %v", err)
		}
	}

	// Update collector enabled status
	for metricName, metricConfig := range policy.Collect.Metrics {
		if err := pm.scheduler.SetCollectorEnabled(metricName, metricConfig.Enabled); err != nil {
			log.Printf("Failed to set collector %s enabled=%v: %v", metricName, metricConfig.Enabled, err)
		} else {
			// Update config
			if pm.config.EnabledMetrics == nil {
				pm.config.EnabledMetrics = make(map[string]bool)
			}
			pm.config.EnabledMetrics[metricName] = metricConfig.Enabled
		}
	}

	pm.currentPolicy = policy
	log.Printf("Applied policy version %d", policy.Version)

	return pm.config.Save()
}

func (pm *PolicyManager) GetCurrentPolicy() *Policy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.currentPolicy
}