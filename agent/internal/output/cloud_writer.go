package output

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/yourorg/inventory-agent/agent/internal/config"
)

type CloudWriter struct {
	config     *config.AgentConfig
	client     *http.Client
	queue      []*queuedPayload
	queueMu    sync.Mutex
	maxQueue   int
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

type queuedPayload struct {
	payload interface{}
	attempts int
	nextAttempt time.Time
}

func NewCloudWriter(cfg *config.AgentConfig) *CloudWriter {
	// Configure HTTP client with timeouts and TLS
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	return &CloudWriter{
		config:   cfg,
		client:   client,
		queue:    make([]*queuedPayload, 0),
		maxQueue: 100, // Max 100 items in queue
		stopChan: make(chan struct{}),
	}
}

func (w *CloudWriter) Write(payload interface{}) error {
	return w.sendPayload(payload)
}

func (w *CloudWriter) sendPayload(payload interface{}) error {
	endpoint := fmt.Sprintf("%s/v1/agents/%s/inventory", w.config.APIEndpoint, w.config.DeviceID)

	// Marshal payload
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Compress if payload > 1KB
	var body io.Reader = bytes.NewReader(data)
	if len(data) > 1024 {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(data); err != nil {
			return fmt.Errorf("failed to compress payload: %w", err)
		}
		gz.Close()
		body = &buf
	}

	// Create request
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+w.config.AuthToken)
	req.Header.Set("Content-Type", "application/json")
	if len(data) > 1024 {
		req.Header.Set("Content-Encoding", "gzip")
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		// Network error - queue for retry
		w.queuePayload(payload)
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	// Handle response
	switch resp.StatusCode {
	case 202:
		// Success
		return nil
	case 401:
		log.Printf("Authentication failed - token may be invalid")
		return fmt.Errorf("authentication failed")
	case 400:
		// Bad request - don't retry
		return fmt.Errorf("bad request")
	case 403:
		// Forbidden - don't retry
		return fmt.Errorf("forbidden")
	default:
		// Server error - queue for retry
		w.queuePayload(payload)
		return fmt.Errorf("server error: %d", resp.StatusCode)
	}
}

func (w *CloudWriter) queuePayload(payload interface{}) {
	w.queueMu.Lock()
	defer w.queueMu.Unlock()

	if len(w.queue) >= w.maxQueue {
		// Remove oldest item
		w.queue = w.queue[1:]
	}

	w.queue = append(w.queue, &queuedPayload{
		payload:     payload,
		attempts:    0,
		nextAttempt: time.Now().Add(w.calculateBackoff(0)),
	})
}

func (w *CloudWriter) calculateBackoff(attempts int) time.Duration {
	backoff := time.Duration(float64(time.Second) * float64(attempts+1) * w.config.RetryConfig.BackoffMultiplier)
	if backoff > w.config.RetryConfig.MaxBackoff {
		backoff = w.config.RetryConfig.MaxBackoff
	}
	return backoff
}

func (w *CloudWriter) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.retryLoop(ctx)
}

func (w *CloudWriter) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

func (w *CloudWriter) retryLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processQueue()
		}
	}
}

func (w *CloudWriter) processQueue() {
	w.queueMu.Lock()
	defer w.queueMu.Unlock()

	now := time.Now()
	var remaining []*queuedPayload

	for _, item := range w.queue {
		if item.nextAttempt.After(now) {
			remaining = append(remaining, item)
			continue
		}

		if item.attempts >= w.config.RetryConfig.MaxRetries {
			log.Printf("Dropping payload after %d attempts", item.attempts)
			continue
		}

		if err := w.sendPayload(item.payload); err != nil {
			item.attempts++
			item.nextAttempt = now.Add(w.calculateBackoff(item.attempts))
			remaining = append(remaining, item)
		}
		// Success - don't add to remaining
	}

	w.queue = remaining
}