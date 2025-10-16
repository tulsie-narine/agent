# Quick Start: Get Windows Inventory Agent Running on Docker Desktop

## Step-by-Step Guide

### Step 1: Install Docker Desktop (if not already installed)

1. Download Docker Desktop for Windows from: https://www.docker.com/products/docker-desktop
2. Run the installer and follow the prompts
3. Restart your computer when prompted
4. Open PowerShell and verify installation:
   ```powershell
   docker --version
   docker-compose --version
   ```
   You should see version numbers for both commands.

---

### Step 2: Clone the Repository

Open PowerShell and run:
```powershell
# Navigate to where you want the project
cd C:\Dev

# Clone the repository
git clone https://github.com/tulsie-narine/agent.git

# Navigate into the project
cd agent
```

---

### Step 3: Create the Environment Configuration File

The project needs a `.env` file with configuration settings:

```powershell
# Copy the example file to create your own
Copy-Item .env.example .env
```

Now open the `.env` file in your text editor and make sure it has these contents:

```bash
# Database Configuration
DATABASE_URL=postgres://inventory:inventory123@postgres:5432/inventory?sslmode=disable

# Message Queue Configuration
NATS_URL=nats://nats:4222

# API Server Configuration
API_PORT=8080

# Security: Generate a secure random string
# Use this command in PowerShell to generate one:
# [Convert]::ToBase64String([System.Security.Cryptography.RNGCryptoServiceProvider]::new().GetBytes(32))
JWT_SECRET=YOUR_GENERATED_SECRET_HERE

# Logging Configuration
LOG_LEVEL=info

# Rate Limiting
RATE_LIMIT_RPS=100
MAX_BATCH_SIZE=1000

# TLS Configuration (empty for local Docker)
TLS_CERT_FILE=
TLS_KEY_FILE=

# Web Console Configuration
NEXT_PUBLIC_API_URL=http://localhost:8080
NODE_ENV=development
```

**Important**: Replace `YOUR_GENERATED_SECRET_HERE` with an actual secure secret. You can generate one using:

```powershell
# Run this in PowerShell to generate a secure JWT_SECRET
$secret = [Convert]::ToBase64String([System.Security.Cryptography.RNGCryptoServiceProvider]::new().GetBytes(32))
Write-Host "Generated JWT_SECRET: $secret"
# Copy the output and paste it into your .env file
```

---

### Step 4: Start All Services

In PowerShell, make sure you're in the `C:\Dev\agent` directory:

```powershell
# Start all services (this will take 2-3 minutes the first time)
docker-compose up -d

# Watch the startup progress
docker-compose logs -f
```

The logs will show services starting. Wait until you see something like:
```
api      | 2025/10/16 17:00:00 Server started on port 8080
web      | > ready - started server on 0.0.0.0:3000
```

Press `Ctrl+C` to exit the logs (services keep running).

---

### Step 5: Verify All Services Are Running

```powershell
# Check status
docker-compose ps

# You should see output like:
# NAME      IMAGE     STATUS           PORTS
# postgres  postgres  Up (healthy)     5432/tcp
# nats      nats      Up (healthy)     4222/tcp
# api       api       Up (healthy)     8080/tcp
# web       web       Up (healthy)     3000/tcp
```

All services should show "Up (healthy)".

---

### Step 6: Access the Web Console

Open your web browser and go to:
```
http://localhost:3000
```

You should see the dashboard page. It will show "No devices" initially (that's normal).

---

### Step 7: Test the API

Open PowerShell and test the API health endpoint:

```powershell
curl http://localhost:8080/health

# You should see output like:
# {"status":"ok"}
```

If you get an error, check that the API container is healthy:
```powershell
docker-compose logs api | tail -20
```

---

### Step 8: (Optional) Build and Test the Agent Locally

If you want to test the agent registration and telemetry:

#### 8a. Build the Agent Binary

```powershell
# Navigate to the agent directory
cd agent

# Build the agent
go build -o ..\bin\inventory-agent.exe -ldflags "-s -w" .

# Go back to project root
cd ..

# Verify binary exists
ls .\bin\inventory-agent.exe
```

#### 8b. Create Agent Configuration

Create a file named `config.json` in the project root:

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

#### 8c. Run the Agent

```powershell
# Start the agent
.\bin\inventory-agent.exe

# You should see logs like:
# Agent started successfully
# Registration successful
```

#### 8d. Check the Web Console

Go to http://localhost:3000 and refresh. You should now see your device listed!

---

## Common Tasks

### View Logs from Services

```powershell
# View logs from all services
docker-compose logs -f

# View logs from specific service
docker-compose logs -f api
docker-compose logs -f web
docker-compose logs -f postgres

# View last 50 lines from API
docker-compose logs --tail=50 api
```

### Stop All Services (Keep Data)

```powershell
docker-compose stop

# Start them again later
docker-compose start
```

### Stop and Remove Everything (Delete Data!)

```powershell
# This will stop and remove all containers and volumes
docker-compose down -v

# Then restart fresh
docker-compose up -d
```

### Restart a Service

```powershell
# Restart the API service
docker-compose restart api

# Restart all services
docker-compose restart
```

### Check Service Status

```powershell
docker-compose ps --format "table {{.Service}}\t{{.Status}}\t{{.Ports}}"
```

---

## Service URLs

Once running, you can access:

| Service | URL | Purpose |
|---------|-----|---------|
| **Web Console** | http://localhost:3000 | Dashboard for viewing devices |
| **API** | http://localhost:8080 | REST API |
| **API Health** | http://localhost:8080/health | Check if API is running |
| **PostgreSQL** | localhost:5432 | Database (use credentials: user=inventory, pass=inventory123) |
| **NATS** | localhost:4222 | Message queue |

---

## Troubleshooting

### Docker Desktop Not Running
**Problem**: You get an error about Docker daemon not running

**Solution**: Start Docker Desktop from your Start menu or system tray

### Containers Keep Crashing
**Problem**: Containers show "Restarting" or "Exited"

**Solution**:
```powershell
# View the logs to see what went wrong
docker-compose logs

# Often caused by port conflicts or insufficient resources
# Try increasing Docker memory allocation
```

### Port Already in Use
**Problem**: Error about "port 3000 already in use"

**Solution**:
```powershell
# Find what's using the port
netstat -ano | findstr :3000

# Either close that application or modify docker-compose.yml
# to use a different port (e.g., change 3000:3000 to 3001:3000)
```

### API Can't Connect to Database
**Problem**: API logs show database connection errors

**Solution**:
```powershell
# Restart services in the correct order
docker-compose restart postgres
docker-compose restart api

# Or do a full restart
docker-compose down
docker-compose up -d
```

### Web Console Blank or Loading Forever
**Problem**: http://localhost:3000 doesn't load or shows blank page

**Solution**:
```powershell
# Check web service logs
docker-compose logs web

# Usually means API_URL is not set correctly
# Edit docker-compose.yml and verify NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

## Next Steps

Once everything is running:

1. **Build the agent** (see Step 8) and test registration
2. **Deploy agents to Windows VMs** for real testing
3. **For production deployment**, see [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) for Railway/Vercel deployment

For more detailed information, see:
- [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md) - Comprehensive Docker guide
- [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) - Cloud deployment guide
- [README.md](README.md) - Project overview

---

**Questions?** Check the troubleshooting section or see the detailed docs linked above.
