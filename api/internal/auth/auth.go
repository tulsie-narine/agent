package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func AuthMiddleware(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract Bearer token
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(401).JSON(fiber.Map{"error": "Authorization header required"})
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			return c.Status(401).JSON(fiber.Map{"error": "Bearer token required"})
		}

		token := strings.TrimPrefix(auth, prefix)
		if token == "" {
			return c.Status(401).JSON(fiber.Map{"error": "Token cannot be empty"})
		}

		// Get device ID from URL param
		deviceIDStr := c.Params("id")
		if deviceIDStr == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Device ID required"})
		}

		deviceID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
		}

		// Query agent
		var agent models.Agent
		err = db.QueryRow(c.Context(),
			"SELECT device_id, org_id, hostname, status, capabilities, auth_token_hash FROM agents WHERE device_id = $1",
			deviceID).Scan(&agent.DeviceID, &agent.OrgID, &agent.Hostname, &agent.Status,
			&agent.Capabilities, &agent.AuthTokenHash)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Device not found"})
		}

		// Verify token
		if err := bcrypt.CompareHashAndPassword([]byte(agent.AuthTokenHash), []byte(token)); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
		}

		// Check if agent is active
		if agent.Status != "active" {
			return c.Status(403).JSON(fiber.Map{"error": "Device is not active"})
		}

		// Store agent in context
		c.Locals("agent", &agent)

		return c.Next()
	}
}

func GetAgentFromContext(c *fiber.Ctx) (*models.Agent, error) {
	agent, ok := c.Locals("agent").(*models.Agent)
	if !ok {
		return nil, fiber.NewError(500, "Agent not found in context")
	}
	return agent, nil
}

func GenerateToken() string {
	return uuid.New().String()
}

func HashToken(token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	return string(hash), err
}