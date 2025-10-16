# API Documentation

## Overview

The Cloud API provides RESTful endpoints for managing Windows inventory agents, policies, commands, and telemetry data. Built with Go and the Fiber framework, it offers high performance and reliability.

## Base URL
```
https://api.yourdomain.com/v1
```

## Authentication

All API requests require authentication using JWT tokens or API keys.

### JWT Authentication
```http
Authorization: Bearer <jwt_token>
```

### API Key Authentication (for agents)
```http
X-API-Key: <api_key>
```

## Response Format

All responses follow a consistent JSON structure:

```json
{
  "data": <response_data>,
  "success": true,
  "message": "Optional message",
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 100,
    "total_pages": 2
  }
}
```

## Error Responses

```json
{
  "data": null,
  "success": false,
  "message": "Error description"
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `422` - Validation Error
- `500` - Internal Server Error

## Endpoints

### Agent Registration

#### Register Agent
```http
POST /agents/register
```

**Request Body:**
```json
{
  "hostname": "WIN-ABC123",
  "capabilities": ["inventory", "commands"],
  "os_version": "Windows 10 Pro",
  "os_build": "19045.2006",
  "architecture": "x64",
  "domain": "CORP"
}
```

**Response:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "api_key": "sk-abc123...",
    "config": {
      "telemetry_interval": 900,
      "command_poll_interval": 60
    }
  },
  "success": true
}
```

### Inventory Management

#### Submit Inventory Data
```http
POST /agents/{id}/inventory
```

**Headers:**
```
X-API-Key: <agent_api_key>
```

**Request Body:**
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "os": {
      "version": "Windows 10 Pro",
      "build": "19045.2006",
      "architecture": "x64"
    },
    "hardware": {
      "cpu": {
        "model": "Intel Core i7-9750H",
        "cores": 6,
        "threads": 12,
        "speed_mhz": 2600
      },
      "memory": {
        "total_gb": 16,
        "available_gb": 8
      },
      "disks": [
        {
          "device": "C:",
          "total_gb": 500,
          "free_gb": 200,
          "type": "SSD"
        }
      ]
    },
    "software": [
      {
        "name": "Google Chrome",
        "version": "120.0.6099.109",
        "publisher": "Google LLC",
        "install_date": "2023-12-01"
      }
    ]
  }
}
```

### Policy Management

#### Get Agent Policy
```http
GET /agents/{id}/policy
```

**Response:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "Standard Security Policy",
    "version": "1.2.0",
    "content": {
      "antivirus": {
        "enabled": true,
        "vendor": "Windows Defender"
      },
      "firewall": {
        "enabled": true,
        "rules": [...]
      },
      "updates": {
        "automatic": true,
        "schedule": "daily"
      }
    },
    "target_filters": {
      "os_version": "Windows 10+",
      "domain": "CORP"
    }
  },
  "success": true
}
```

### Command Management

#### Get Pending Commands
```http
GET /agents/{id}/commands
```

**Response:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "command_type": "run_script",
      "command_data": {
        "script": "Get-Process | Select Name, CPU",
        "timeout": 30
      },
      "created_at": "2024-01-15T10:00:00Z",
      "expires_at": "2024-01-15T11:00:00Z"
    }
  ],
  "success": true
}
```

#### Acknowledge Command
```http
POST /agents/{id}/commands/{cmdId}/ack
```

**Request Body:**
```json
{
  "result": {
    "exit_code": 0,
    "output": "Process output...",
    "error": null
  }
}
```

### Device Management

#### List Devices
```http
GET /devices
```

**Query Parameters:**
- `page` (integer, default: 1) - Page number
- `limit` (integer, default: 50) - Items per page
- `status` (string) - Filter by status: online, offline, unknown
- `hostname` (string) - Filter by hostname pattern
- `domain` (string) - Filter by domain

**Response:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "hostname": "WIN-ABC123",
      "os_version": "Windows 10 Pro",
      "os_build": "19045.2006",
      "architecture": "x64",
      "domain": "CORP",
      "last_seen": "2024-01-15T10:30:00Z",
      "status": "online",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "success": true,
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 150,
    "total_pages": 3
  }
}
```

#### Get Device Details
```http
GET /devices/{id}
```

#### Get Device Telemetry
```http
GET /devices/{id}/telemetry
```

**Query Parameters:**
- `metric_type` (string) - Filter by metric type: cpu, memory, disk, network
- `start_time` (ISO 8601) - Start time for data range
- `end_time` (ISO 8601) - End time for data range
- `limit` (integer, default: 100) - Maximum number of data points

### Health Checks

#### API Health
```http
GET /health
```

**Response:**
```json
{
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "1.0.0",
    "services": {
      "database": "healthy",
      "nats": "healthy",
      "cache": "healthy"
    }
  },
  "success": true
}
```

#### Metrics
```http
GET /metrics
```

Returns Prometheus-compatible metrics for monitoring.

## Rate Limiting

API requests are rate limited based on endpoint and authentication type:

- **Agent endpoints**: 100 requests per minute per agent
- **Management endpoints**: 1000 requests per minute per user
- **Health endpoints**: Unlimited

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642249200
```

## WebSocket Support (Future)

Real-time updates will be available via WebSocket connections:

```javascript
const ws = new WebSocket('wss://api.yourdomain.com/v1/ws');

// Authentication
ws.send(JSON.stringify({
  type: 'auth',
  token: 'jwt_token_here'
}));

// Subscribe to device updates
ws.send(JSON.stringify({
  type: 'subscribe',
  channel: 'devices',
  filter: { status: 'online' }
}));
```

## SDKs and Libraries

### JavaScript/TypeScript Client
```bash
npm install @yourorg/inventory-api-client
```

```typescript
import { InventoryAPI } from '@yourorg/inventory-api-client';

const client = new InventoryAPI({
  baseURL: 'https://api.yourdomain.com/v1',
  token: 'your-jwt-token'
});

const devices = await client.devices.list();
```

### Go Client
```bash
go get github.com/yourorg/inventory-api-go
```

```go
import "github.com/yourorg/inventory-api-go"

client := inventory.NewClient("https://api.yourdomain.com/v1", "jwt-token")
devices, err := client.Devices.List(context.Background(), &inventory.DeviceListOptions{})
```

## Versioning

API versioning follows semantic versioning:

- **v1**: Current stable version
- **Breaking changes**: New major version
- **Additions**: Minor version bump
- **Bug fixes**: Patch version bump

## Changelog

### v1.0.0 (Current)
- Initial release with core inventory management
- Agent registration and authentication
- Policy distribution and command execution
- Device management and telemetry APIs

### Planned Features
- Real-time WebSocket updates
- Advanced filtering and search
- Bulk operations
- API key management
- Audit logging endpoints