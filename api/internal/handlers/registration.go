package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourorg/inventory-agent/api/internal/auth"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegistrationHandler struct {
	db *pgxpool.Pool
}

type RegistrationRequest struct {
	DeviceID    string                 `json:"device_id"`
	Hostname    string                 `json:"hostname"`
	Capabilities []models.Capability   `json:"capabilities"`
	AgentVersion string                 `json:"agent_version"`
}

type RegistrationResponse struct {
	DeviceID     string `json:"device_id"`
	AuthToken    string `json:"auth_token,omitempty"`
	PolicyVersion int    `json:"policy_version"`
}

func NewRegistrationHandler(db *pgxpool.Pool) *RegistrationHandler {
	return &RegistrationHandler{db: db}
}

func (h *RegistrationHandler) Register(c *fiber.Ctx) error {
	var req RegistrationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if req.DeviceID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "device_id is required"})
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid device_id format"})
	}

	// Check if agent already exists
	var existingAgent models.Agent
	err = h.db.QueryRow(c.Context(),
		"SELECT device_id, auth_token_hash, status FROM agents WHERE device_id = $1",
		deviceID).Scan(&existingAgent.DeviceID, &existingAgent.AuthTokenHash, &existingAgent.Status)

	isNewAgent := err != nil // pgx.ErrNoRows

	var authToken string
	var authTokenHash string

	if isNewAgent {
		// Generate new token for new agent
		authToken = uuid.New().String()
		authTokenHash, err = auth.HashToken(authToken)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to generate auth token"})
		}

		// Insert new agent
		_, err = h.db.Exec(c.Context(), `
			INSERT INTO agents (device_id, hostname, capabilities, first_seen_at, last_seen_at, auth_token_hash, agent_version, status)
			VALUES ($1, $2, $3, $4, $4, $5, $6, 'active')`,
			deviceID, req.Hostname, req.Capabilities, time.Now(), authTokenHash, req.AgentVersion)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to register agent"})
		}
	} else {
		// Update existing agent
		authTokenHash, err = auth.HashToken(uuid.New().String()) // Generate new token for re-registration
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to generate auth token"})
		}

		authToken = uuid.New().String()
		newHash, err := auth.HashToken(authToken)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to generate auth token"})
		}

		_, err = h.db.Exec(c.Context(), `
			UPDATE agents
			SET hostname = $2, capabilities = $3, last_seen_at = $4, auth_token_hash = $5, agent_version = $6, status = 'active'
			WHERE device_id = $1`,
			deviceID, req.Hostname, req.Capabilities, time.Now(), newHash, req.AgentVersion)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update agent"})
		}
	}

	// Log registration event
	_, err = h.db.Exec(c.Context(), `
		INSERT INTO audit_log (actor, action, resource_type, resource_id, details)
		VALUES ($1, $2, $3, $4, $5)`,
		"agent", "register", "agent", deviceID.String(),
		map[string]interface{}{"hostname": req.Hostname, "agent_version": req.AgentVersion})
	if err != nil {
		// Log error but don't fail registration
		// TODO: Add proper logging
	}

	resp := RegistrationResponse{
		DeviceID:     deviceID.String(),
		AuthToken:    authToken, // Only sent on registration/re-registration
		PolicyVersion: 1,        // TODO: Get actual policy version
	}

	return c.Status(200).JSON(resp)
}