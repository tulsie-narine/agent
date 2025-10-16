package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandAdminHandler struct {
	db *pgxpool.Pool
}

func NewCommandAdminHandler(db *pgxpool.Pool) *CommandAdminHandler {
	return &CommandAdminHandler{db: db}
}

func (h *CommandAdminHandler) GetCommands(c *fiber.Ctx) error {
	deviceIDStr := c.Query("device_id")
	var deviceID *uuid.UUID
	if deviceIDStr != "" {
		if id, err := uuid.Parse(deviceIDStr); err == nil {
			deviceID = &id
		} else {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
		}
	}

	query := `
		SELECT command_id, device_id, type, parameters, issued_at, ttl_seconds,
			   status, result, completed_at
		FROM commands`
	args := []interface{}{}
	argCount := 0

	if deviceID != nil {
		argCount++
		query += ` WHERE device_id = $` + fmt.Sprintf("%d", argCount)
		args = append(args, *deviceID)
	}

	query += ` ORDER BY issued_at DESC`

	rows, err := h.db.Query(c.Context(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query commands"})
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		err := rows.Scan(&cmd.CommandID, &cmd.DeviceID, &cmd.Type, &cmd.Parameters,
			&cmd.IssuedAt, &cmd.TTLSeconds, &cmd.Status, &cmd.Result, &cmd.CompletedAt)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan command"})
		}
		commands = append(commands, cmd)
	}

	return c.JSON(fiber.Map{"data": commands})
}

func (h *CommandAdminHandler) CreateCommand(c *fiber.Ctx) error {
	var cmd models.Command
	if err := c.BodyParser(&cmd); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid command data"})
	}

	// Set defaults
	cmd.CommandID = uuid.New()
	cmd.Status = "pending"
	cmd.IssuedAt = time.Now()

	if cmd.TTLSeconds == 0 {
		cmd.TTLSeconds = 3600 // 1 hour default
	}

	if err := cmd.Validate(); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid command: " + err.Error()})
	}

	_, err := h.db.Exec(c.Context(), `
		INSERT INTO commands (command_id, device_id, type, parameters, issued_at, ttl_seconds, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		cmd.CommandID, cmd.DeviceID, cmd.Type, cmd.Parameters, cmd.IssuedAt,
		cmd.TTLSeconds, cmd.Status)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create command"})
	}

	return c.Status(201).JSON(fiber.Map{"data": cmd})
}