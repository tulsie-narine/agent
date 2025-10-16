package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"github.com/yourorg/inventory-agent/agent/internal/command"
	"github.com/yourorg/inventory-agent/agent/internal/config"
	"github.com/yourorg/inventory-agent/agent/internal/output"
	"github.com/yourorg/inventory-agent/agent/internal/policy"
	"github.com/yourorg/inventory-agent/agent/internal/registration"
	"github.com/yourorg/inventory-agent/agent/internal/scheduler"
)

type agentService struct {
	config     *config.AgentConfig
	scheduler  *scheduler.Scheduler
	policyMgr  *policy.PolicyManager
	commandPoller *command.CommandPoller
	registrar  *registration.Registrar
}

func (a *agentService) Start(s service.Service) error {
	log.Println("Starting Inventory Agent service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	a.config = cfg

	// Initialize components
	ctx := context.Background()

	// Registration (Phase 2)
	a.registrar = registration.New(a.config)
	if err := a.registrar.Register(ctx); err != nil {
		log.Printf("Registration failed, continuing with local mode: %v", err)
	}

	// Initialize outputs
	var writers []scheduler.Writer
	localWriter := output.NewLocalWriter(a.config.LocalOutputPath)
	writers = append(writers, localWriter)

	if a.config.APIEndpoint != "" {
		cloudWriter := output.NewCloudWriter(a.config)
		writers = append(writers, cloudWriter)
	}

	// Initialize scheduler
	a.scheduler = scheduler.New(a.config, writers)

	// Initialize policy manager (Phase 5)
	a.policyMgr = policy.NewPolicyManager(a.config, a.scheduler)

	// Initialize command poller (Phase 7)
	a.commandPoller = command.NewCommandPoller(a.config, a.scheduler)

	// Start background processes
	go a.scheduler.Start(ctx)
	go a.policyMgr.Start(ctx)
	go a.commandPoller.Start(ctx)

	log.Println("Inventory Agent started successfully")
	return nil
}

func (a *agentService) Stop(s service.Service) error {
	log.Println("Stopping Inventory Agent service")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop components in reverse order
	if a.commandPoller != nil {
		a.commandPoller.Stop()
	}
	if a.policyMgr != nil {
		a.policyMgr.Stop()
	}
	if a.scheduler != nil {
		a.scheduler.Stop()
	}

	// Wait for context cancellation
	<-ctx.Done()

	log.Println("Inventory Agent stopped")
	return nil
}

func main() {
	svcFlag := flag.String("service", "", "Control the system service (install, uninstall, start, stop)")
	configFlag := flag.String("config", "", "Path to configuration file")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Inventory Agent v1.0.0\nBuild: %s\n", "development")
		os.Exit(0)
	}

	// Service configuration
	svcConfig := &service.Config{
		Name:        "InventoryAgent",
		DisplayName: "Inventory Agent",
		Description: "Collects system inventory and telemetry data",
	}

	agentSvc := &agentService{}
	s, err := service.New(agentSvc, svcConfig)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Handle service control commands
	if *svcFlag != "" {
		switch *svcFlag {
		case "install":
			err := s.Install()
			if err != nil {
				log.Fatalf("Failed to install service: %v", err)
			}
			fmt.Println("Service installed successfully")
		case "uninstall":
			err := s.Uninstall()
			if err != nil {
				log.Fatalf("Failed to uninstall service: %v", err)
			}
			fmt.Println("Service uninstalled successfully")
		case "start":
			err := s.Start()
			if err != nil {
				log.Fatalf("Failed to start service: %v", err)
			}
			fmt.Println("Service started successfully")
		case "stop":
			err := s.Stop()
			if err != nil {
				log.Fatalf("Failed to stop service: %v", err)
			}
			fmt.Println("Service stopped successfully")
		default:
			log.Fatalf("Unknown service command: %s", *svcFlag)
		}
		return
	}

	// Override config path if specified
	if *configFlag != "" {
		os.Setenv("AGENT_CONFIG_PATH", *configFlag)
	}

	// Run as service or interactively
	if service.Interactive() {
		// Interactive mode - handle signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		err := s.Run()
		if err != nil {
			log.Fatalf("Service failed: %v", err)
		}

		// Wait for signal
		<-sigChan
	} else {
		// Service mode
		err := s.Run()
		if err != nil {
			log.Fatalf("Service failed: %v", err)
		}
	}
}