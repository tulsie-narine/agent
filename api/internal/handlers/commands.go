package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandHandler struct {
	db *pgxpool.Pool
}

type CommandRequest struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	TTLSeconds int                    `json:"ttl_seconds"`
}

func NewCommandHandler(db *pgxpool.Pool) *CommandHandler {
	return &CommandHandler{db: db}
}

func (h *CommandHandler) GetCommands(c *fiber.Ctx) error {
	deviceIDStr := c.Params("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
	}

	// Query pending commands that haven't expired
	rows, err := h.db.Query(c.Context(), `
		SELECT command_id, type, parameters, issued_at, ttl_seconds, status
		FROM commands
		WHERE device_id = $1
		  AND status = 'pending'
		  AND issued_at + (ttl_seconds || ' seconds')::interval > NOW()
		ORDER BY issued_at ASC`,
		deviceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query commands"})
	}
	defer rows.Close()

	var commands []models.Command
	for rows.Next() {
		var cmd models.Command
		err := rows.Scan(&cmd.CommandID, &cmd.Type, &cmd.Parameters,
			&cmd.IssuedAt, &cmd.TTLSeconds, &cmd.Status)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan command"})
		}
		commands = append(commands, cmd)
	}

	// Mark commands as executing
	for _, cmd := range commands {
		_, err = h.db.Exec(c.Context(), `
			UPDATE commands SET status = 'executing' WHERE command_id = $1`,
			cmd.CommandID)
		if err != nil {
			// Log error but continue
		}
	}

	return c.JSON(commands)
}

func (h *CommandHandler) AckCommand(c *fiber.Ctx) error {
	deviceIDStr := c.Params("id")
	commandIDStr := c.Params("cmdId")

	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
	}

	commandID, err := uuid.Parse(commandIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid command ID"})
	}

	var ack struct {
		Result map[string]interface{} `json:"result"`
		Error  string                 `json:"error,omitempty"`
	}

	if err := c.BodyParser(&ack); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Update command
	status := "completed"
	if ack.Error != "" {
		status = "failed"
		ack.Result = map[string]interface{}{"error": ack.Error}
	}

	_, err = h.db.Exec(c.Context(), `
		UPDATE commands
		SET status = $1, result = $2, completed_at = NOW()
		WHERE command_id = $3 AND device_id = $4`,
		status, ack.Result, commandID, deviceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update command"})
	}

	// Log to audit
	_, err = h.db.Exec(c.Context(), `
		INSERT INTO audit_log (actor, action, resource_type, resource_id, details)
		VALUES ($1, $2, $3, $4, $5)`,
		"agent", "ack_command", "command", commandID.String(),
		map[string]interface{}{"status": status})
	if err != nil {
		// Log but don't fail
	}

	return c.SendStatus(200)
}