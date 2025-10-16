# Local Docker Desktop Deployment Guide

This guide provides comprehensive instructions for deploying the Windows Inventory Agent & Cloud Console system locally using Docker Desktop on Windows.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Service URLs](#service-urls)
4. [Verification Steps](#verification-steps)
5. [Building the Windows Agent](#building-the-windows-agent)
6. [Agent Configuration](#agent-configuration)
7. [Testing Agent Registration](#testing-agent-registration)
8. [Troubleshooting](#troubleshooting)
9. [Development Workflow](#development-workflow)
10. [Stopping and Cleaning Up](#stopping-and-cleaning-up)
11. [Next Steps](#next-steps)

## Prerequisites

### System Requirements
- **Docker Desktop for Windows**: Version 4.0 or later
  - Download from: https://www.docker.com/products/docker-desktop
- **WSL 2 Backend**: Enabled (recommended)
  - Required for better performance on Windows
  - Docker Desktop installer includes WSL 2 setup
- **RAM Allocation**: At least 8GB allocated to Docker
  - Recommended: 10GB+ for comfortable development
  - Configure in Docker Desktop Settings → Resources → Memory
- **Disk Space**: At least 20GB free
  - Required for containers, images, and data volumes
- **Windows Version**: Windows 10 version 20H2 or Windows 11
- **Git**: For cloning the repository
- **Go 1.22+**: For building the agent binary (optional, for local testing)

### Verify Installation
```powershell
# Check Docker is running
docker --version
docker ps

# Check Docker Compose is installed
docker-compose --version
```

## Quick Start

### Step 1: Clone Repository
```powershell
# Clone the repository (if not already done)
git clone https://github.com/tulsie-narine/agent.git
cd agent
```

### Step 2: Create Environment File
```powershell
# Copy the example environment file
Copy-Item .env.example .env

# Edit .env to set a secure JWT_SECRET
# Use this PowerShell command to generate a secure secret:
# [Convert]::ToBase64String([System.Security.Cryptography.RNGCryptoServiceProvider]::new().GetBytes(32))
# Then update the JWT_SECRET value in .env
```

Or manually generate JWT_SECRET using OpenSSL (if available):
```bash
openssl rand -base64 32
```

### Step 3: Start Docker Services
```powershell
# Start all services in detached mode
docker-compose up -d

# Watch the build and startup process
docker-compose logs -f

# Press Ctrl+C to exit logs view (containers keep running)
```

### Step 4: Verify Services
```powershell
# Check status of all services
docker-compose ps

# All services should show "Up" status
# Example output:
# NAME      IMAGE     STATUS           PORTS
# postgres  postgres  Up (healthy)     5432/tcp
# nats      nats      Up (healthy)     4222/tcp, 8222/tcp
# api       api       Up (healthy)     8080/tcp
# web       web       Up (healthy)     3000/tcp
```

## Service URLs

Once all services are running, access them at:

| Service | URL | Purpose |
|---------|-----|---------|
| **Web Console** | http://localhost:3000 | Dashboard for viewing devices and inventory |
| **API** | http://localhost:8080 | REST API for agents and console |
| **API Health** | http://localhost:8080/health | API health check endpoint |
| **PostgreSQL** | localhost:5432 | Database (for direct connection if needed) |
| **NATS** | localhost:4222 | Message queue (for agent telemetry) |
| **NATS HTTP API** | http://localhost:8222 | NATS monitoring endpoint |
| **PgAdmin** | http://localhost:5050 | PostgreSQL web interface (optional) |

### Credentials
- **PostgreSQL**: User: `inventory`, Password: `inventory123`
- **PgAdmin**: Email: `admin@inventory.local`, Password: `admin123`

## Verification Steps

### 1. Test API Health Endpoint
```powershell
# Test the health check endpoint
curl http://localhost:8080/health

# Expected response (200 OK):
# {"status":"ok"}
```

### 2. Access Web Console
1. Open browser and navigate to **http://localhost:3000**
2. You should see the dashboard page load
3. The page should show "No devices" initially (no agents registered yet)

### 3. Check Container Health
```powershell
# View detailed container information
docker-compose ps --format "table {{.Service}}\t{{.Status}}\t{{.Ports}}"

# View container logs for specific service
docker-compose logs postgres
docker-compose logs nats
docker-compose logs api
docker-compose logs web

# View real-time logs from all services
docker-compose logs -f
```

### 4. Verify PostgreSQL Database
```powershell
# Connect to PostgreSQL and check database
docker-compose exec postgres psql -U inventory -d inventory -c "\dt"

# This should show tables like: devices, telemetry, etc.
```

### 5. Check NATS JetStream
```powershell
# Access NATS monitoring API
curl http://localhost:8222/jsz

# Should return JSON with JetStream information including TELEMETRY stream
```

## Building the Windows Agent

### Option 1: Build Using Make
```powershell
# Build the agent binary (requires Go 1.22+)
make build-agent

# Binary will be created at: ./bin/inventory-agent.exe
```

### Option 2: Build Using Go Directly
```powershell
# Navigate to agent directory
cd agent

# Build for Windows
go build -o ..\bin\inventory-agent.exe -ldflags "-s -w" .

# Or for 64-bit Windows specifically
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o ..\bin\inventory-agent.exe -ldflags "-s -w" .

# Return to project root
cd ..

# Verify binary was created
.\bin\inventory-agent.exe -version
```

### Cross-Compilation
```bash
# If building from Linux/macOS for Windows
cd agent
GOOS=windows GOARCH=amd64 go build -o ../bin/inventory-agent.exe -ldflags "-s -w" .
```

## Agent Configuration

### Create Configuration File
Create `config.json` in the project root directory:

```json
{
  "device_id": "",
  "api_endpoint": "http://localhost:8080",
  "auth_token": "",
  "collection_interval": "15m",
  "enabled_metrics": {
    "os.info": true,
    "cpu": true,
    "memory": true,
    "disk": true,
    "software": true
  },
  "local_output_path": "C:\\temp\\inventory.json",
  "log_level": "info",
  "retry_config": {
    "max_retries": 5,
    "backoff_multiplier": 2.0,
    "max_backoff": "5m"
  }
}
```

**Configuration Notes:**
- `device_id`: Leave empty - auto-generated on first run (UUID v4)
- `api_endpoint`: Must be `http://localhost:8080` for Docker Desktop local testing
- `auth_token`: Leave empty - obtained during registration
- `collection_interval`: How often to collect metrics (minimum 1 minute)
- `enabled_metrics`: Which system metrics to collect
- `local_output_path`: Where to save local inventory data as backup
- `log_level`: debug, info, warn, error

## Testing Agent Registration

### Step 1: Start the Agent
```powershell
# Run the agent binary
.\bin\inventory-agent.exe

# You should see startup logs
# Look for: "Agent started successfully"
# And: "Registration successful" or "Registering device..."
```

### Step 2: Monitor Agent Logs
```powershell
# In another PowerShell window, watch the agent logs
Get-Content config.json | ConvertFrom-Json

# Check if device_id and auth_token were populated
```

### Step 3: Verify Registration in Web Console
1. Open http://localhost:3000 in your browser
2. Refresh the page (or wait a moment)
3. You should now see your device listed in the dashboard
4. Click on the device to view collected inventory data

### Step 4: Check Telemetry
1. Look for collected metrics (CPU, Memory, Disk usage)
2. Verify telemetry timestamps are recent
3. Check API logs for telemetry ingestion:
   ```powershell
   docker-compose logs api | Select-String "telemetry" | Tail -20
   ```

## Troubleshooting

### Port Conflicts
**Problem**: Port already in use (e.g., port 3000 already used by another application)

**Solution**:
```powershell
# Find what's using the port (e.g., port 3000)
netstat -ano | findstr :3000

# Either stop the conflicting application or modify docker-compose.yml
# to use different ports:
# Change "3000:3000" to "3001:3000" to use port 3001 instead
```

### Container Health Check Failures
**Problem**: Containers show "Unhealthy" status or keep restarting

**Diagnosis**:
```powershell
# Check container logs
docker-compose logs postgres
docker-compose logs api

# Look for specific error messages
```

**Common Solutions**:
- Ensure previous containers are fully stopped: `docker-compose down`
- Remove old volumes: `docker-compose down -v`
- Increase resource allocation to Docker
- Check firewall rules blocking container communication

### API Connection Errors
**Problem**: Web console can't connect to API

**Diagnosis**:
1. Check API is running: `docker-compose ps`
2. Test API directly: `curl http://localhost:8080/health`
3. Check browser console for network errors (F12 → Console tab)

**Solution**:
- Verify `NEXT_PUBLIC_API_URL` is set to `http://localhost:8080` in docker-compose.yml
- Ensure API service is healthy (not showing "Unhealthy")
- Check firewall allows port 8080

### Database Migration Issues
**Problem**: API crashes with database migration errors

**Diagnosis**:
```powershell
docker-compose logs api | Select-String "migration"
```

**Solution**:
- Ensure PostgreSQL is fully initialized: `docker-compose logs postgres | Select-String "ready to accept"`
- Reset database: `docker-compose down -v && docker-compose up -d`

### Agent Registration Fails
**Problem**: Agent can't connect to API (`http://localhost:8080`)

**Diagnosis**:
1. Verify agent config.json has correct api_endpoint
2. Test connectivity: `Test-NetConnection localhost -Port 8080`
3. Check agent logs for specific error

**Solution**:
- For Windows host testing: Use `http://localhost:8080` (Docker Desktop provides localhost bridge)
- For WSL 2 agent: May need to use `http://host.docker.internal:8080`
- Ensure API container is healthy

### Database Connection String Issues
**Problem**: API shows "Failed to connect to database"

**Diagnosis**:
- Check environment variables in docker-compose.yml
- Verify PostgreSQL container is healthy

**Solution**:
- Ensure `DATABASE_URL` matches the PostgreSQL service configuration
- Default should be: `postgres://inventory:inventory123@postgres:5432/inventory?sslmode=disable`

### Out of Memory Errors
**Problem**: Docker containers crash or system becomes unresponsive

**Solution**:
1. Increase Docker memory allocation:
   - Open Docker Desktop Settings
   - Go to Resources → Memory
   - Increase to 10GB or more
2. Restart Docker Desktop

## Development Workflow

### Making Code Changes

#### Web Console Changes
```powershell
# Changes to web console files are automatically reloaded
# Modify files in ./web/src/
# Next.js will hot-reload the browser

# To rebuild explicitly:
docker-compose up -d --build web
```

#### API Changes
```powershell
# After modifying API code:
docker-compose restart api

# Or rebuild the image:
docker-compose up -d --build api

# View logs:
docker-compose logs -f api
```

### Viewing Logs
```powershell
# Tail logs from all services
docker-compose logs -f

# Logs from specific service
docker-compose logs -f api
docker-compose logs -f web

# View last 100 lines
docker-compose logs --tail=100

# Show timestamps
docker-compose logs -f --timestamps
```

### Database Access

#### Using PgAdmin (Web UI)
1. Open http://localhost:5050
2. Login with `admin@inventory.local` / `admin123`
3. Add new server:
   - Host: `postgres`
   - Username: `inventory`
   - Password: `inventory123`

#### Using psql (Command Line)
```powershell
# Connect to PostgreSQL
docker-compose exec postgres psql -U inventory -d inventory

# List tables
\dt

# Query devices
SELECT * FROM devices;

# Exit
\q
```

### Debugging

#### Enable Debug Logging
```powershell
# Edit .env and set LOG_LEVEL=debug
# Then restart containers:
docker-compose restart api

# View debug logs:
docker-compose logs -f api
```

#### Inspect Container
```powershell
# Open shell in container
docker-compose exec api sh

# Inside container, explore:
ls -la
ps aux
env

# Exit
exit
```

## Stopping and Cleaning Up

### Stop Services (Keep Data)
```powershell
# Stop all containers but keep volumes
docker-compose stop

# Start again later
docker-compose start
```

### Full Shutdown
```powershell
# Stop and remove containers
docker-compose down

# This removes containers but keeps volumes (persistent data)
```

### Complete Cleanup
```powershell
# Remove containers and volumes (WARNING: deletes database!)
docker-compose down -v

# Remove all unused Docker resources
docker system prune -f

# If needed, remove images too
docker system prune -a
```

### Selective Cleanup
```powershell
# Remove only web service
docker-compose rm -f web

# Remove only specific volumes
docker volume rm agent_postgres_data
```

## Next Steps

### After Local Verification
1. **Test agent on other Windows VMs**: Copy agent binary and config to other machines
2. **Performance testing**: Monitor resource usage with multiple agents
3. **Integration testing**: Test full workflow with multiple agents and devices
4. **Production deployment**: Refer to `DEPLOYMENT_GUIDE.md` for Railway/Vercel deployment

### Scaling Considerations
- Docker Desktop is suitable for development and testing
- For production with multiple agents, use Railway/Vercel (see `DEPLOYMENT_GUIDE.md`)
- For on-premises production, consider Kubernetes deployment (see `docs/DEPLOYMENT.md`)

### Additional Resources
- **Full Deployment Guide**: See `DEPLOYMENT_GUIDE.md` for Railway/Vercel cloud deployment
- **Architecture Documentation**: See `docs/ARCHITECTURE.md`
- **API Documentation**: See `docs/API.md`
- **Database Schema**: See `docs/DATABASE.md`
- **Alternative Deployments**: See `docs/DEPLOYMENT.md`

## Makefile Shortcuts

Make common Docker operations easier:

```powershell
# Build Docker images
make docker-build

# Start services with fresh build
make docker-up-build

# View service status
make docker-status

# Tail logs from all services
make docker-logs

# Restart all services
make docker-restart

# Complete cleanup (removes volumes!)
make docker-clean
```

See `Makefile` for all available targets.

---

**Last Updated**: October 2025
**Compatibility**: Windows 10/11 with Docker Desktop 4.0+
