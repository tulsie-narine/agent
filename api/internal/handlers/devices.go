package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceHandler struct {
	db *pgxpool.Pool
}

func NewDeviceHandler(db *pgxpool.Pool) *DeviceHandler {
	return &DeviceHandler{db: db}
}

func (h *DeviceHandler) GetDevices(c *fiber.Ctx) error {
	// Parse query parameters
	limit := 50 // default
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	status := c.Query("status") // active, inactive, offline, or empty for all
	hostname := c.Query("hostname")

	// Build query
	query := `
		SELECT device_id, hostname, status, agent_version, first_seen_at, last_seen_at
		FROM agents
		WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	if status != "" {
		argCount++
		query += ` AND status = $` + strconv.Itoa(argCount)
		args = append(args, status)
	}

	if hostname != "" {
		argCount++
		query += ` AND hostname ILIKE $` + strconv.Itoa(argCount)
		args = append(args, "%"+hostname+"%")
	}

	query += ` ORDER BY last_seen_at DESC LIMIT $` + strconv.Itoa(argCount+1) + ` OFFSET $` + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	// Execute query
	rows, err := h.db.Query(c.Context(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query devices"})
	}
	defer rows.Close()

	var devices []models.Agent
	for rows.Next() {
		var device models.Agent
		err := rows.Scan(&device.DeviceID, &device.Hostname, &device.Status,
			&device.AgentVersion, &device.FirstSeenAt, &device.LastSeenAt)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan device"})
		}
		devices = append(devices, device)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM agents WHERE 1=1`
	countArgs := []interface{}{}

	if status != "" {
		countQuery += ` AND status = $1`
		countArgs = append(countArgs, status)
	}

	if hostname != "" {
		countQuery += ` AND hostname ILIKE $2`
		countArgs = append(countArgs, "%"+hostname+"%")
	}

	var total int
	err = h.db.QueryRow(c.Context(), countQuery, countArgs...).Scan(&total)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get total count"})
	}

	return c.JSON(fiber.Map{
		"devices": devices,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *DeviceHandler) GetDevice(c *fiber.Ctx) error {
	deviceIDStr := c.Params("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
	}

	// Get device info
	var device models.Agent
	err = h.db.QueryRow(c.Context(), `
		SELECT device_id, hostname, status, capabilities, agent_version,
		       first_seen_at, last_seen_at
		FROM agents WHERE device_id = $1`, deviceID).Scan(
		&device.DeviceID, &device.Hostname, &device.Status, &device.Capabilities,
		&device.AgentVersion, &device.FirstSeenAt, &device.LastSeenAt)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}

	// Get latest telemetry
	var telemetry models.Telemetry
	err = h.db.QueryRow(c.Context(), `
		SELECT collected_at, metrics
		FROM telemetry_latest WHERE device_id = $1`, deviceID).Scan(
		&telemetry.CollectedAt, &telemetry.Metrics)
	if err != nil {
		// No telemetry yet, that's ok
		telemetry.Metrics = make(map[string]interface{})
	}

	return c.JSON(fiber.Map{
		"device":    device,
		"telemetry": telemetry,
	})
}

func (h *DeviceHandler) GetDeviceTelemetry(c *fiber.Ctx) error {
	deviceIDStr := c.Params("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
	}

	// Parse time range (default last 24 hours)
	hours := 24
	if h := c.Query("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 && parsed <= 168 { // max 1 week
			hours = parsed
		}
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	rows, err := h.db.Query(c.Context(), `
		SELECT collected_at, metrics
		FROM telemetry
		WHERE device_id = $1 AND collected_at >= $2
		ORDER BY collected_at DESC`, deviceID, since)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query telemetry"})
	}
	defer rows.Close()

	var telemetry []models.Telemetry
	for rows.Next() {
		var t models.Telemetry
		err := rows.Scan(&t.CollectedAt, &t.Metrics)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan telemetry"})
		}
		t.DeviceID = deviceID
		telemetry = append(telemetry, t)
	}

	return c.JSON(telemetry)
}

func (h *DeviceHandler) GetDeviceStats(c *fiber.Ctx) error {
	var stats struct {
		TotalDevices     int64 `json:"total_devices"`
		ActiveDevices    int64 `json:"active_devices"`
		OfflineDevices   int64 `json:"offline_devices"`
		InactiveDevices  int64 `json:"inactive_devices"`
		RecentTelemetry  int64 `json:"recent_telemetry"`
		PendingCommands  int64 `json:"pending_commands"`
	}

	// Get device counts by status
	err := h.db.QueryRow(c.Context(), `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'active') as active,
			COUNT(*) FILTER (WHERE status = 'offline') as offline,
			COUNT(*) FILTER (WHERE status = 'inactive') as inactive
		FROM agents`).Scan(&stats.TotalDevices, &stats.ActiveDevices, &stats.OfflineDevices, &stats.InactiveDevices)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query device stats"})
	}

	// Get recent telemetry count (last 24 hours)
	err = h.db.QueryRow(c.Context(), `
		SELECT COUNT(*) FROM telemetry WHERE collected_at >= NOW() - INTERVAL '24 hours'`,
	).Scan(&stats.RecentTelemetry)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query telemetry stats"})
	}

	// Get pending commands count
	err = h.db.QueryRow(c.Context(), `
		SELECT COUNT(*) FROM commands
		WHERE status = 'pending'
		  AND issued_at + (ttl_seconds || ' seconds')::interval > NOW()`,
	).Scan(&stats.PendingCommands)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query command stats"})
	}

	return c.JSON(fiber.Map{"data": stats})
}