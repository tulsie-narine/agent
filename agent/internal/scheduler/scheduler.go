package scheduler

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/yourorg/inventory-agent/agent/internal/collectors"
	"github.com/yourorg/inventory-agent/agent/internal/config"
)

type TelemetryPayload struct {
	DeviceID     string                 `json:"device_id"`
	AgentVersion string                 `json:"agent_version"`
	CollectedAt  time.Time              `json:"collected_at"`
	Metrics      map[string]interface{} `json:"metrics"`
}

type Writer interface {
	Write(payload interface{}) error
}

type Scheduler struct {
	config      *config.AgentConfig
	registry    *collectors.CollectorRegistry
	writers     []Writer
	ticker      *time.Ticker
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
}

func New(cfg *config.AgentConfig, writers []Writer) *Scheduler {
	registry := collectors.NewRegistry()

	// Register all collectors
	registry.Register(collectors.NewOSInfoCollector())
	registry.Register(collectors.NewSoftwareCollector())
	registry.Register(collectors.NewCPUCollector())
	registry.Register(collectors.NewMemoryCollector())
	registry.Register(collectors.NewDiskCollector())

	// Apply initial configuration
	for name, enabled := range cfg.EnabledMetrics {
		registry.SetEnabled(name, enabled)
	}

	return &Scheduler{
		config:   cfg,
		registry: registry,
		writers:  writers,
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker != nil {
		return // Already started
	}

	interval := s.config.CollectionInterval
	s.ticker = time.NewTicker(interval)

	s.wg.Add(1)
	go s.run(ctx)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker == nil {
		return // Not started
	}

	close(s.stopChan)
	s.ticker.Stop()
	s.ticker = nil

	s.wg.Wait()
}

func (s *Scheduler) UpdateInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = time.NewTicker(interval)
	}
}

func (s *Scheduler) TriggerNow() error {
	return s.collectAndWrite(context.Background())
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	// Add jitter to avoid thundering herd
	jitter := time.Duration(rand.Int63n(int64(s.config.CollectionInterval / 10)))
	time.Sleep(jitter)

	for {
		select {
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		case <-s.ticker.C:
			if err := s.collectAndWrite(ctx); err != nil {
				log.Printf("Collection failed: %v", err)
			}
		}
	}
}

func (s *Scheduler) collectAndWrite(ctx context.Context) error {
	enabledCollectors := s.registry.Enabled()

	payload := &TelemetryPayload{
		DeviceID:     s.config.DeviceID,
		AgentVersion: "1.0.0", // TODO: inject from build
		CollectedAt:  time.Now().UTC(),
		Metrics:      make(map[string]interface{}),
	}

	// Collect from all enabled collectors
	for _, collector := range enabledCollectors {
		collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		result, err := collector.Collect(collectCtx)
		cancel()

		if err != nil {
			log.Printf("Collector %s failed: %v", collector.Name(), err)
			continue
		}

		payload.Metrics[collector.Name()] = result
	}

	// Write to all configured writers
	for _, writer := range s.writers {
		if err := writer.Write(payload); err != nil {
			log.Printf("Writer failed: %v", err)
			// Continue with other writers
		}
	}

	log.Printf("Collection completed: %d metrics collected", len(payload.Metrics))
	return nil
}

func (s *Scheduler) SetCollectorEnabled(name string, enabled bool) error {
	return s.registry.SetEnabled(name, enabled)
}