package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AdminAuthMiddleware() fiber.Handler {
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

		// TODO: Implement proper admin JWT validation
		// For now, accept any token (implement proper validation later)
		if token == "admin-token" || len(token) > 10 {
			// Set admin user in context
			c.Locals("admin_user", "admin")
			return c.Next()
		}

		return c.Status(401).JSON(fiber.Map{"error": "Invalid admin token"})
	}
}