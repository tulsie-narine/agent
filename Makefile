# Inventory Agent Makefile

# Variables
GO_VERSION := 1.22
GOOS := windows
GOARCH := amd64
CGO_ENABLED := 1
LDFLAGS := -s -w
BUILD_DIR := ./dist
AGENT_BINARY := $(BUILD_DIR)/agent.exe
API_BINARY := $(BUILD_DIR)/api.exe

# Go build flags
GO_BUILD_FLAGS := -ldflags "$(LDFLAGS)" -tags netgo

.PHONY: help build-agent build-api build-web test-agent test-api test-web lint docker-up docker-down db-migrate-up db-migrate-down msi-package docker-build docker-up-build docker-logs docker-restart docker-clean docker-status clean

help: ## Show this help message
	@echo "Inventory Agent Build System"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build-agent: ## Build Windows agent binary
	@echo "Building agent for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_FLAGS) -o $(AGENT_BINARY) ./agent
	@echo "Agent built: $(AGENT_BINARY)"

build-api: ## Build API server binary
	@echo "Building API server..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(API_BINARY) ./api
	@echo "API built: $(API_BINARY)"

build-web: ## Build Next.js production bundle
	@echo "Building web console..."
	cd web && npm run build
	@echo "Web built successfully"

test-agent: ## Run agent tests
	@echo "Running agent tests..."
	cd agent && go test -v -race -coverprofile=coverage.out ./...
	@echo "Agent tests completed"

test-api: ## Run API tests
	@echo "Running API tests..."
	cd api && go test -v -race -coverprofile=coverage.out ./...
	@echo "API tests completed"

test-web: ## Run web tests
	@echo "Running web tests..."
	cd web && npm test
	@echo "Web tests completed"

lint: ## Run linters
	@echo "Running Go linter..."
	golangci-lint run ./agent/... ./api/...
	@echo "Running web linter..."
	cd web && npm run lint
	@echo "Linting completed"

docker-up: ## Start Docker Compose services
	@echo "Starting Docker services..."
	docker-compose up -d
	@echo "Services started: API at http://localhost:8080, Web at http://localhost:3000"

docker-build: ## Build all Docker images locally
	@echo "Building Docker images..."
	docker-compose build
	@echo "Docker images built"

docker-up-build: ## Start Docker services with fresh image builds
	@echo "Building and starting Docker services..."
	docker-compose up -d --build
	@echo "Services started with fresh builds"

docker-logs: ## Tail logs from all Docker services
	@echo "Tailing Docker logs (Ctrl+C to exit)..."
	docker-compose logs -f

docker-restart: ## Restart all Docker services
	@echo "Restarting Docker services..."
	docker-compose restart
	@echo "Services restarted"

docker-status: ## Show status of Docker services
	@echo "Docker service status:"
	@docker-compose ps --format "table {{.Service}}\t{{.Status}}\t{{.Ports}}"

docker-clean: ## Stop services and remove containers, volumes, and images
	@echo "Cleaning Docker environment (this will delete data!)..."
	docker-compose down -v
	docker system prune -f
	@echo "Docker environment cleaned"

docker-down: ## Stop Docker Compose services (keeps data)
	@echo "Stopping Docker services..."
	docker-compose down
	@echo "Services stopped. Data preserved in volumes"

db-migrate-up: ## Run database migrations up
	@echo "Running database migrations..."
	@migrate -path api/internal/database/migrations -database "$(DATABASE_URL)" up
	@echo "Migrations completed"

db-migrate-down: ## Run database migrations down
	@echo "Rolling back database migrations..."
	@migrate -path api/internal/database/migrations -database "$(DATABASE_URL)" down 1
	@echo "Migration rolled back"

msi-package: build-agent ## Build MSI installer package
	@echo "Building MSI package..."
	@powershell -ExecutionPolicy Bypass -File tools/deployment/build-msi.ps1
	@echo "MSI package built"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	cd web && rm -rf .next out
	go clean ./...
	@echo "Clean completed"

# Development helpers
run-api: build-api ## Build and run API server
	@echo "Starting API server..."
	./$(API_BINARY)

run-agent: build-agent ## Build and run agent (for testing)
	@echo "Starting agent..."
	./$(AGENT_BINARY) --config ./agent/config.json

dev-web: ## Start web development server
	@echo "Starting web dev server..."
	cd web && npm run dev

install-deps: ## Install development dependencies
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing web dependencies..."
	cd web && npm install
	@echo "Installing tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Version injection (set VERSION variable)
version: ## Show current version
	@echo "Version: $(shell git describe --tags --always --dirty)"

# CI/CD targets
ci-test: test-agent test-api test-web lint ## Run all tests and linting for CI
	@echo "CI tests completed"

ci-build: build-agent build-api build-web ## Build all components for CI
	@echo "CI build completed"