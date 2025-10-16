package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nats-io/nats.go"
	"github.com/yourorg/inventory-agent/api/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	db *pgxpool.Pool
	nc *nats.Conn
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Database  string    `json:"database"`
	NATS      string    `json:"nats"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
	Timestamp time.Time `json:"timestamp"`
}

func NewHealthHandler(db *pgxpool.Pool, nc *nats.Conn) *HealthHandler {
	return &HealthHandler{db: db, nc: nc}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	resp := HealthResponse{
		Status:    "healthy",
		Version:   "1.0.0",
		Uptime:    "unknown", // TODO: Track actual uptime
		Timestamp: time.Now(),
	}

	// Check database
	if err := database.Ping(h.db); err != nil {
		resp.Status = "unhealthy"
		resp.Database = "error: " + err.Error()
	} else {
		resp.Database = "ok"
	}

	// Check NATS
	if h.nc == nil {
		resp.NATS = "error: not connected"
		resp.Status = "unhealthy"
	} else {
		// Simple connectivity check
		_, err := h.nc.Request("health.check", []byte("ping"), time.Second)
		if err != nil {
			resp.NATS = "error: " + err.Error()
			resp.Status = "unhealthy"
		} else {
			resp.NATS = "ok"
		}
	}

	statusCode := 200
	if resp.Status != "healthy" {
		statusCode = 503
	}

	return c.Status(statusCode).JSON(resp)
}

func (h *HealthHandler) Metrics(c *fiber.Ctx) error {
	// Basic Prometheus-style metrics
	metrics := `# HELP inventory_api_info API information
# TYPE inventory_api_info gauge
inventory_api_info{version="1.0.0"} 1

# HELP inventory_api_uptime_seconds API uptime in seconds
# TYPE inventory_api_uptime_seconds gauge
inventory_api_uptime_seconds 0

# HELP inventory_database_connections_active Active database connections
# TYPE inventory_database_connections_active gauge
inventory_database_connections_active 0

# HELP inventory_nats_connected NATS connection status
# TYPE inventory_nats_connected gauge
inventory_nats_connected{status="connected"} 1
`

	// Add database connection info if available
	if h.db != nil {
		// Note: In a real implementation, you'd use prometheus client library
		// to properly instrument database stats, HTTP requests, etc.
	}

	return c.Type("text/plain").SendString(metrics)
}