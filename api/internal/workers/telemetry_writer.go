package workers

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelemetryWriter struct {
	db     *pgxpool.Pool
	js     nats.JetStream
	sub    *nats.Subscription
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewTelemetryWriter(db *pgxpool.Pool, js nats.JetStream) *TelemetryWriter {
	return &TelemetryWriter{
		db:     db,
		js:     js,
		stopCh: make(chan struct{}),
	}
}

func (w *TelemetryWriter) Start(ctx context.Context) error {
	// Subscribe to telemetry stream using JetStream
	sub, err := w.js.PullSubscribe("telemetry.ingest", "telemetry-writer")
	if err != nil {
		return err
	}
	w.sub = sub

	w.wg.Add(1)
	go w.run(ctx)

	log.Println("Telemetry writer started with JetStream")
	return nil
}

func (w *TelemetryWriter) Stop() {
	if w.sub != nil {
		w.sub.Unsubscribe()
	}
	close(w.stopCh)
	w.wg.Wait()
	log.Println("Telemetry writer stopped")
}

func (w *TelemetryWriter) run(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			// Fetch messages from JetStream
			msgs, err := w.sub.Fetch(100, nats.MaxWait(5*time.Second))
			if err != nil {
				if err != nats.ErrTimeout {
					log.Printf("Failed to fetch messages: %v", err)
				}
				continue
			}

			// Process messages
			for _, msg := range msgs {
				w.handleMessage(msg)
			}
		}
	}
}

func (w *TelemetryWriter) handleMessage(msg *nats.Msg) {
	var telemetry models.Telemetry
	if err := json.Unmarshal(msg.Data, &telemetry); err != nil {
		log.Printf("Failed to unmarshal telemetry: %v", err)
		msg.Nak()
		return
	}

	// For now, process immediately (could batch here too)
	if err := w.writeTelemetry(&telemetry); err != nil {
		log.Printf("Failed to write telemetry: %v", err)
		msg.Nak()
		return
	}

	msg.Ack()
}

func (w *TelemetryWriter) writeTelemetry(telemetry *models.Telemetry) error {
	ctx := context.Background()

	// Begin transaction
	tx, err := w.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Insert into telemetry table
	_, err = tx.Exec(ctx, `
		INSERT INTO telemetry (device_id, collected_at, metrics, tags, seq, ingestion_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		telemetry.DeviceID, telemetry.CollectedAt, telemetry.Metrics,
		telemetry.Tags, telemetry.Seq, telemetry.IngestionID)
	if err != nil {
		return err
	}

	// Upsert latest telemetry
	_, err = tx.Exec(ctx, `
		INSERT INTO telemetry_latest (device_id, collected_at, metrics, tags, seq, ingestion_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (device_id) DO UPDATE SET
			collected_at = EXCLUDED.collected_at,
			metrics = EXCLUDED.metrics,
			tags = EXCLUDED.tags,
			seq = EXCLUDED.seq,
			server_received_at = NOW()`,
		telemetry.DeviceID, telemetry.CollectedAt, telemetry.Metrics,
		telemetry.Tags, telemetry.Seq, telemetry.IngestionID)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit(ctx)
}

func (w *TelemetryWriter) processBatch(batch []*models.Telemetry) {
	// TODO: Implement batch insert for better performance
	for _, telemetry := range batch {
		if err := w.writeTelemetry(telemetry); err != nil {
			log.Printf("Failed to write telemetry batch item: %v", err)
		}
	}
}