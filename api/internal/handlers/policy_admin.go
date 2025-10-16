package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourorg/inventory-agent/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PolicyAdminHandler struct {
	db *pgxpool.Pool
}

func NewPolicyAdminHandler(db *pgxpool.Pool) *PolicyAdminHandler {
	return &PolicyAdminHandler{db: db}
}

func (h *PolicyAdminHandler) GetPolicies(c *fiber.Ctx) error {
	rows, err := h.db.Query(c.Context(), `
		SELECT policy_id, scope, version, config, created_by, created_at
		FROM policies
		WHERE scope = 'global'
		ORDER BY created_at DESC`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to query policies"})
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var policy models.Policy
		err := rows.Scan(&policy.PolicyID, &policy.Scope, &policy.Version,
			&policy.Config, &policy.CreatedBy, &policy.CreatedAt)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to scan policy"})
		}
		policies = append(policies, policy)
	}

	return c.JSON(fiber.Map{"data": policies})
}

func (h *PolicyAdminHandler) CreatePolicy(c *fiber.Ctx) error {
	var policy models.Policy
	if err := c.BodyParser(&policy); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid policy data"})
	}

	// Set defaults for global policies
	policy.Scope = "global"
	policy.Version = 1
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	policy.CreatedBy = "admin" // TODO: Get from context

	if err := policy.Validate(); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid policy: " + err.Error()})
	}

	_, err := h.db.Exec(c.Context(), `
		INSERT INTO policies (device_id, group_id, scope, version, config, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		policy.DeviceID, policy.GroupID, policy.Scope, policy.Version,
		policy.Config, policy.CreatedBy, policy.CreatedAt, policy.UpdatedAt)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create policy"})
	}

	return c.Status(201).JSON(fiber.Map{"data": policy})
}

func (h *PolicyAdminHandler) UpdatePolicy(c *fiber.Ctx) error {
	policyIDStr := c.Params("id")
	policyID, err := strconv.ParseInt(policyIDStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid policy ID"})
	}

	var updates models.Policy
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid policy data"})
	}

	updates.UpdatedAt = time.Now()
	updates.CreatedBy = "admin" // TODO: Get from context

	if err := updates.Validate(); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid policy: " + err.Error()})
	}

	_, err = h.db.Exec(c.Context(), `
		UPDATE policies
		SET config = $2, version = version + 1, updated_at = $3
		WHERE policy_id = $1`,
		policyID, updates.Config, updates.UpdatedAt)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update policy"})
	}

	return c.JSON(fiber.Map{"data": updates})
}

func (h *PolicyAdminHandler) DeletePolicy(c *fiber.Ctx) error {
	policyIDStr := c.Params("id")
	policyID, err := strconv.ParseInt(policyIDStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid policy ID"})
	}

	_, err = h.db.Exec(c.Context(), "DELETE FROM policies WHERE policy_id = $1", policyID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete policy"})
	}

	return c.JSON(fiber.Map{"message": "Policy deleted"})
}