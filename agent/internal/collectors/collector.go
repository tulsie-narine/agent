package collectors

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Collector interface {
	Name() string
	Collect(ctx context.Context) (interface{}, error)
	Enabled() bool
}

type CollectorRegistry struct {
	collectors map[string]Collector
	mu         sync.RWMutex
}

func NewRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]Collector),
	}
}

func (r *CollectorRegistry) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[c.Name()] = c
}

func (r *CollectorRegistry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[name]
	return c, ok
}

func (r *CollectorRegistry) All() []Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	collectors := make([]Collector, 0, len(r.collectors))
	for _, c := range r.collectors {
		collectors = append(collectors, c)
	}
	return collectors
}

func (r *CollectorRegistry) Enabled() []Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var enabled []Collector
	for _, c := range r.collectors {
		if c.Enabled() {
			enabled = append(enabled, c)
		}
	}
	return enabled
}

func (r *CollectorRegistry) SetEnabled(name string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.collectors[name]
	if !ok {
		return fmt.Errorf("collector %s not found", name)
	}
	// Note: This assumes collectors have a SetEnabled method
	// Implementation depends on collector interface
	if setter, ok := c.(interface{ SetEnabled(bool) }); ok {
		setter.SetEnabled(enabled)
	}
	return nil
}

type BaseCollector struct {
	name     string
	enabled  bool
	lastRun  time.Time
	mu       sync.RWMutex
}

func NewBaseCollector(name string, enabled bool) *BaseCollector {
	return &BaseCollector{
		name:    name,
		enabled: enabled,
	}
}

func (c *BaseCollector) Name() string {
	return c.name
}

func (c *BaseCollector) Enabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

func (c *BaseCollector) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

func (c *BaseCollector) LastRun() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRun
}

func (c *BaseCollector) SetLastRun(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastRun = t
}