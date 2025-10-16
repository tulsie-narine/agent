package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/yourorg/inventory-agent/api/internal/auth"
	"github.com/yourorg/inventory-agent/api/internal/config"
	"github.com/yourorg/inventory-agent/api/internal/database"
	"github.com/yourorg/inventory-agent/api/internal/handlers"
	"github.com/yourorg/inventory-agent/api/internal/workers"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Debug: Log all environment variables
	log.Println("=== All Environment Variables ===")
	for _, env := range os.Environ() {
		if strings.Contains(strings.ToUpper(env), "DATABASE") || strings.Contains(strings.ToUpper(env), "PG") || strings.Contains(strings.ToUpper(env), "HOST") {
			log.Printf("ENV: %s", env)
		}
	}
	log.Println("=== End All Environment Variables ===")

	// Log database URL (mask password for security)
	dbURL := cfg.DatabaseURL
	if strings.Contains(dbURL, "://") && strings.Contains(dbURL, "@") {
		// Parse postgres://user:pass@host:port/db
		protocolEnd := strings.Index(dbURL, "://")
		atIndex := strings.Index(dbURL, "@")
		if protocolEnd != -1 && atIndex != -1 {
			protocol := dbURL[:protocolEnd+3] // "postgres://"
			userPass := dbURL[protocolEnd+3 : atIndex] // "user:pass"
			hostAndRest := dbURL[atIndex+1:] // "host:port/db?params"
			user := strings.Split(userPass, ":")[0]
			log.Printf("Using DATABASE_URL: %s%s:***@%s", protocol, user, hostAndRest)
		} else {
			log.Printf("Using DATABASE_URL: %s", dbURL)
		}
	} else {
		log.Printf("Using DATABASE_URL: %s", dbURL)
	}

	// Initialize database with retries
	var db *pgxpool.Pool
	var dbErr error
	maxRetries := 30
	retryDelay := 2 * time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, dbErr = database.Connect(cfg.DatabaseURL)
		if dbErr == nil {
			log.Printf("Database connected successfully (attempt %d)", attempt)
			break
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...", attempt, maxRetries, dbErr, retryDelay)
		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}
	
	if dbErr != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
		// Don't fatally fail - the server can still work
	}

	// Initialize NATS
	nc, err := connectNATS(cfg.NATSUrl)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Initialize JetStream
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to initialize JetStream: %v", err)
	}

	// Create telemetry stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "TELEMETRY",
		Subjects: []string{"telemetry.ingest"},
		Storage:  nats.FileStorage,
		Replicas: 1,
	})
	if err != nil {
		log.Printf("Warning: Failed to create telemetry stream (may already exist): %v", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://inventory.yourdomain.com,https://app.inventory.yourdomain.com,http://localhost:3000",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Rate limiting middleware
	app.Use(limiter.New(limiter.Config{
		Max:               100, // requests per window
		Expiration:        60 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use IP for public routes, device ID for agent routes
			if strings.HasPrefix(c.Path(), "/v1/agents/") {
				return c.Params("id") // Rate limit by device ID
			}
			return c.IP() // Rate limit by IP for other routes
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Rate limit exceeded",
				"retry_after": "60",
			})
		},
	}))

	// Initialize handlers
	regHandler := handlers.NewRegistrationHandler(db)
	inventoryHandler := handlers.NewInventoryHandler(db, js)
	policyHandler := handlers.NewPolicyHandler(db)
	commandHandler := handlers.NewCommandHandler(db)
	deviceHandler := handlers.NewDeviceHandler(db)
	policyAdminHandler := handlers.NewPolicyAdminHandler(db)
	commandAdminHandler := handlers.NewCommandAdminHandler(db)
	healthHandler := handlers.NewHealthHandler(db, nc)

	// Routes
	v1 := app.Group("/v1")

	// Public routes
	v1.Post("/agents/register", regHandler.Register)

	// Agent routes (device authentication)
	agentRoutes := v1.Group("/agents", auth.AuthMiddleware(db))
	agentRoutes.Post("/:id/inventory", inventoryHandler.Ingest)
	agentRoutes.Get("/:id/policy", policyHandler.GetPolicy)
	agentRoutes.Get("/:id/commands", commandHandler.GetCommands)
	agentRoutes.Post("/:id/commands/:cmdId/ack", commandHandler.AckCommand)

	// Admin routes (admin authentication)
	adminRoutes := v1.Group("", auth.AdminAuthMiddleware())
	adminRoutes.Get("/devices", deviceHandler.GetDevices)
	adminRoutes.Get("/devices/:id", deviceHandler.GetDevice)
	adminRoutes.Get("/devices/:id/telemetry", deviceHandler.GetDeviceTelemetry)
	adminRoutes.Get("/devices/stats", deviceHandler.GetDeviceStats)
	adminRoutes.Get("/policies", policyAdminHandler.GetPolicies)
	adminRoutes.Post("/policies", policyAdminHandler.CreatePolicy)
	adminRoutes.Put("/policies/:id", policyAdminHandler.UpdatePolicy)
	adminRoutes.Delete("/policies/:id", policyAdminHandler.DeletePolicy)
	adminRoutes.Get("/commands", commandAdminHandler.GetCommands)
	adminRoutes.Post("/commands", commandAdminHandler.CreateCommand)

	// Health check (no auth)
	app.Get("/health", healthHandler.Health)
	app.Get("/metrics", healthHandler.Metrics)

	// Health check (no auth)
	app.Get("/health", healthHandler.Health)
	app.Get("/metrics", healthHandler.Metrics)

	// Start background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	telemetryWorker := workers.NewTelemetryWriter(db, js)
	if err := telemetryWorker.Start(ctx); err != nil {
		log.Fatalf("Failed to start telemetry worker: %v", err)
	}

	commandExpirer := workers.NewCommandExpirer(db)
	commandExpirer.Start(ctx)

	partitionManager := workers.NewPartitionManager(db)
	partitionManager.Start(ctx)

	// Start server
	serverAddr := ":" + cfg.ServerPort

	go func() {
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			log.Printf("Starting HTTPS server on %s", serverAddr)
			if err := app.ListenTLS(serverAddr, cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
				log.Fatalf("HTTPS server failed: %v", err)
			}
		} else {
			log.Printf("Starting HTTP server on %s", serverAddr)
			if err := app.Listen(serverAddr); err != nil {
				log.Fatalf("HTTP server failed: %v", err)
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Stop workers
	cancel()

	log.Println("Server exited")
}

func runMigrations(databaseURL string) error {
	log.Println("Running database migrations...")

	// Parse the database URL to get a sql.DB instance
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	// Create migration driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/database/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No new migrations to run")
	} else {
		log.Println("Migrations completed successfully")
	}

	return nil
}

func connectNATS(url string) (*nats.Conn, error) {
	return nats.Connect(url)
}