package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PolicyHandler struct {
	db *pgxpool.Pool
}

func NewPolicyHandler(db *pgxpool.Pool) *PolicyHandler {
	return &PolicyHandler{db: db}
}

func (h *PolicyHandler) GetPolicy(c *fiber.Ctx) error {
	deviceIDStr := c.Params("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid device ID"})
	}

	// Get agent info
	var agent models.Agent
	err = h.db.QueryRow(c.Context(),
		"SELECT device_id, org_id, capabilities FROM agents WHERE device_id = $1",
		deviceID).Scan(&agent.DeviceID, &agent.OrgID, &agent.Capabilities)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Device not found"})
	}

	// Query all applicable policies
	rows, err := h.db.Query(c.Context(), `
		SELECT policy_id, device_id, group_id, scope, version, config
		FROM policies
		WHERE (scope = 'global')
		   OR (scope = 'group' AND group_id = $1)
		   OR (scope = 'device' AND device_id = $2)
		ORDER BY version DESC`,
		agent.OrgID, deviceID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query policies"})
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var policy models.Policy
		err := rows.Scan(&policy.PolicyID, &policy.DeviceID, &policy.GroupID,
			&policy.Scope, &policy.Version, &policy.Config)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan policy"})
		}
		policies = append(policies, policy)
	}

	// Resolve effective policy
	effectivePolicy := models.ResolveEffectivePolicy(policies, deviceID, agent.OrgID)
	if effectivePolicy == nil {
		// Return default policy
		effectivePolicy = &models.Policy{
			Version: 1,
			Config: models.PolicyConfig{
				IntervalSeconds: 900, // 15 minutes
				Metrics:        map[string]models.MetricConfig{},
			},
		}
	}

	// Filter by capabilities
	effectivePolicy.FilterByCapabilities(agent.Capabilities)

	// Check ETag for caching
	etag := effectivePolicy.GenerateETag()
	if ifNoneMatch := c.Get("If-None-Match"); ifNoneMatch != "" && ifNoneMatch == etag {
		return c.Status(304).Send(nil)
	}

	// Set ETag header
	c.Set("ETag", etag)

	return c.JSON(effectivePolicy)
}