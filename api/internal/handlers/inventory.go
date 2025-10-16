package handlers

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryHandler struct {
	db *pgxpool.Pool
	js nats.JetStream
}

type TelemetryPayload struct {
	DeviceID     string                 `json:"device_id"`
	AgentVersion string                 `json:"agent_version"`
	CollectedAt  time.Time              `json:"collected_at"`
	Metrics      map[string]interface{} `json:"metrics"`
}

func NewInventoryHandler(db *pgxpool.Pool, js nats.JetStream) *InventoryHandler {
	return &InventoryHandler{db: db, js: js}
}

func (h *InventoryHandler) Ingest(c *fiber.Ctx) error {
	deviceIDStr := c.Params("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
	}

	// Authenticate - this is done by middleware, but verify device exists
	var agent models.Agent
	err = h.db.QueryRow(c.Context(),
		"SELECT device_id, status FROM agents WHERE device_id = $1",
		deviceID).Scan(&agent.DeviceID, &agent.Status)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Device not found"})
	}

	if agent.Status != "active" {
		return c.Status(403).JSON(fiber.Map{"error": "Device is not active"})
	}

	// Parse request body (handle gzip)
	var reader io.Reader = c.Request().BodyStream()
	if c.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid gzip content"})
		}
	}

	var payload TelemetryPayload
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid telemetry payload"})
	}

	// Validate payload
	if payload.DeviceID != deviceIDStr {
		return c.Status(400).JSON(fiber.Map{"error": "Device ID mismatch"})
	}

	if payload.CollectedAt.IsZero() {
		return c.Status(400).JSON(fiber.Map{"error": "collected_at is required"})
	}

	// Create telemetry record
	telemetry := &models.Telemetry{
		DeviceID:    deviceID,
		CollectedAt: payload.CollectedAt,
		Metrics:     payload.Metrics,
		Seq:         0, // TODO: Implement sequence numbers
		IngestionID: uuid.New(),
	}

	if err := telemetry.Validate(); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid telemetry data: " + err.Error()})
	}

	// Publish to JetStream for async processing
	data, err := json.Marshal(telemetry)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to serialize telemetry"})
	}

	_, err = h.js.Publish("telemetry.ingest", data)
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "Message queue unavailable"})
	}

	// Update agent's last seen
	_, err = h.db.Exec(c.Context(),
		"UPDATE agents SET last_seen_at = $1 WHERE device_id = $2",
		time.Now(), deviceID)
	if err != nil {
		// Log error but don't fail the request
	}

	return c.Status(202).JSON(fiber.Map{
		"ingestion_id": telemetry.IngestionID.String(),
		"status":       "accepted",
	})
}