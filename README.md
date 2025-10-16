# Inventory Agent & Cloud Console

A comprehensive Windows inventory collection and management system designed for secure, scalable deployment across thousands of devices. Built with Go for performance, PostgreSQL for data storage, and Next.js for the web console.

## Overview

This system consists of three main components:

- **Agent**: Windows service that collects system inventory (OS info, CPU, memory, disk, software) and reports to the cloud backend
- **API**: High-performance REST API built with Go Fiber for handling telemetry ingestion, policy distribution, and command execution
- **Web**: Modern React-based console for device management, policy configuration, and command issuance

## Architecture

The system follows a pull-based architecture with store-and-forward capabilities:

- Agents register with the backend and receive authentication tokens
- Telemetry is collected locally and posted via HTTPS with exponential backoff
- Policies are distributed dynamically with ETag caching for efficiency
- Commands enable ad-hoc collection and remote execution
- NATS JetStream decouples ingestion from database writes for scalability

## Technology Stack

- **Agent & API**: Go 1.22+ (performance, single-binary deployment, excellent Windows service support)
- **Database**: PostgreSQL 16+ with native partitioning for 10k+ devices
- **Message Queue**: NATS JetStream for reliable telemetry ingestion
- **Web Console**: Next.js 14+ with TypeScript and Tailwind CSS
- **Deployment**: Docker Compose for development, Kubernetes for production

## Monorepo Structure

```
/
├── agent/          # Windows agent service (Go)
├── api/            # Backend API server (Go)
├── web/            # Web console (Next.js)
├── docs/           # Documentation
├── shared/         # Common schemas and contracts
├── tools/          # Build and testing utilities
├── docker-compose.yml
├── Makefile
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- PostgreSQL 16+
- Docker & Docker Compose

### Development Setup

1. Clone the repository
2. Copy `.env.example` to `.env` and configure
3. Start services: `make docker-up`
4. Run database migrations: `make db-migrate-up`
5. Build and run API: `make build-api && make run-api`
6. Build and run web: `cd web && npm install && npm run dev`
7. Build agent: `make build-agent`

### Agent Installation

For Windows deployment:

```powershell
# Build MSI package
.\tools\wix\build.ps1 -Version 1.0.0

# Install silently
msiexec /i inventory-agent-1.0.0.msi /quiet
```

## Data Flow

1. **Registration**: Agent generates UUID, registers with API, receives auth token
2. **Telemetry Collection**: Agent collects metrics every 15 minutes (configurable)
3. **Ingestion**: Telemetry posted to API, published to NATS, batched to PostgreSQL
4. **Policy Distribution**: Agent polls for policy updates every 60 seconds
5. **Command Execution**: Admin issues commands via web, agent polls and executes

## Security

- TLS 1.2+ enforced for all communications
- Token-based authentication with bcrypt hashing
- Audit logging for all policy and command changes
- Minimal agent privileges (LocalSystem service)
- Future: mTLS, token rotation, policy signatures

## Performance Targets

- Agent: <1% CPU, <60 MB RAM
- API: <300ms p95 ingest latency for 10k agents
- Web: <500ms p95 page load time

## Phases

The project is implemented in 8 phases:

1. **Agent Core**: Local collection and JSON output
2. **Cloud Backend**: Registration and ingest API
3. **Web Console**: Device list and detail views
4. **Online Mode**: HTTP posting with backoff
5. **Dynamic Policies**: Policy distribution and agent reconfiguration
6. **Capability Negotiation**: Policy validation against agent capabilities
7. **Commands**: Ad-hoc collection and remote execution
8. **Security & Scale**: Hardening, MSI packaging, 10k agent load testing

See [PHASES.md](docs/PHASES.md) for detailed implementation guides.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and component interactions
- [API Reference](docs/API.md) - REST API documentation with OpenAPI spec
- [Database Schema](docs/DATABASE.md) - PostgreSQL schema and partitioning strategy
- [Deployment](docs/DEPLOYMENT.md) - Installation and configuration guides
- [Testing](docs/TESTING.md) - Testing strategy and procedures
- [Security](docs/SECURITY.md) - Security architecture and best practices

## Contributing

1. Follow the phased approach outlined in [PHASES.md](docs/PHASES.md)
2. Implement with comprehensive tests and documentation
3. Ensure all acceptance criteria are met before advancing phases
4. Use the provided tools for building, testing, and deployment

## License

[License information here]