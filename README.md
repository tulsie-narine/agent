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

## Deploying to Railway

### Prerequisites
- Railway account (free tier available)
- Railway CLI installed (optional): `npm i -g @railway/cli`
- Git repository connected to Railway

### Deployment Methods

#### Method 1: Railway Dashboard (Recommended)
1. **Create New Project:**
   - Log in to Railway dashboard
   - Click "New Project" → "Deploy from GitHub repo"
   - Connect and select the `tulsie-narine/agent` repository

2. **Add PostgreSQL Database:**
   - Click "New" → "Database" → "Add PostgreSQL"
   - Railway automatically provisions a managed PostgreSQL instance
   - The `DATABASE_URL` environment variable is automatically created

3. **Deploy Services:**
   - Railway automatically detects the `railway.toml` configuration
   - It will create three services: `postgres`, `nats`, and `api`
   - Services are deployed in dependency order

4. **Configure Environment Variables:**
   - Navigate to the `api` service settings → "Variables" tab
   - Add `JWT_SECRET`: Generate using `openssl rand -base64 32`
   - Verify `DATABASE_URL` is automatically set
   - Other variables use defaults from `railway.toml`

5. **Deploy:**
   - Railway automatically builds and deploys all services
   - Monitor deployment logs in the dashboard
   - Wait for all health checks to pass

#### Method 2: Railway CLI
```bash
npm i -g @railway/cli
railway login
railway init
railway link
railway up
railway variables set JWT_SECRET=<your-secret>
```

### Service Architecture

The `railway.toml` defines three services:

- **PostgreSQL Service**: Managed database for persistent storage
- **NATS Service**: Message queue with JetStream for telemetry ingestion
- **API Service**: Go-based REST API built from `./api/Dockerfile`

### Environment Variables Reference

| Variable | Required | Default | Description | Source |
|----------|----------|---------|-------------|--------|
| `DATABASE_URL` | Yes | Auto | PostgreSQL connection string | Railway (auto) |
| `NATS_URL` | Yes | `nats://nats:4222` | NATS server URL | railway.toml |
| `API_PORT` | Yes | `8080` | API server port | railway.toml |
| `JWT_SECRET` | Yes | - | JWT signing secret | Manual (dashboard) |
| `LOG_LEVEL` | No | `info` | Logging level | railway.toml |
| `RATE_LIMIT_RPS` | No | `100` | Rate limit per second | railway.toml |
| `MAX_BATCH_SIZE` | No | `1000` | Telemetry batch size | railway.toml |

### Post-Deployment Verification

1. **Check Service Health:**
   - Navigate to each service in Railway dashboard
   - Verify all services show "Healthy" status
   - Check deployment logs for any errors

2. **Test API Endpoint:**
   - Get the public URL from Railway dashboard
   - Test health endpoint: `curl https://your-api-url.railway.app/health`
   - Should return `{"status":"ok"}` or similar

3. **Verify Database Migrations:**
   - Check API logs for "Migrations completed successfully" message
   - This confirms the database schema was created

4. **Test NATS Connection:**
   - Check API logs for successful NATS connection
   - Verify JetStream stream creation ("TELEMETRY" stream)

### Connecting Web Console to Railway API

1. Get the Railway API public URL from the dashboard
2. In Vercel project settings, update `NEXT_PUBLIC_API_URL` environment variable
3. Set it to the Railway API URL (e.g., `https://api-production-xxxx.up.railway.app`)
4. Redeploy the Vercel application
5. The web console will now connect to the Railway-hosted API

### Monitoring and Logs

- Access logs: Railway dashboard → Select service → "Logs" tab
- View metrics: Railway dashboard → Select service → "Metrics" tab
- Set up alerts: Railway dashboard → Project settings → "Notifications"

### Troubleshooting

**Common Issues:**

1. **API fails to start:**
   - Check if PostgreSQL and NATS services are healthy
   - Verify `DATABASE_URL` is correctly set
   - Check API logs for connection errors

2. **Database migration errors:**
   - Ensure PostgreSQL service is fully provisioned
   - Check if migrations directory is included in Docker build
   - Review migration logs in API service

3. **NATS connection failures:**
   - Verify NATS service is running and healthy
   - Check `NATS_URL` is set to `nats://nats:4222`
   - Ensure JetStream is enabled

4. **Rate limiting issues:**
   - Adjust `RATE_LIMIT_RPS` in environment variables
   - Check API logs for rate limit errors
   - Consider scaling the API service in Railway

### Cost Considerations

- Railway free tier includes $5 credit per month
- PostgreSQL and API services consume resources
- Monitor usage in Railway dashboard
- Consider upgrading to paid plan for production workloads

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