# Inventory Agent API

High-performance REST API for the Inventory Agent system built with Go Fiber. Handles device registration, telemetry ingestion, policy distribution, and command execution for thousands of agents.

## Architecture

The API is designed for horizontal scalability and low latency:

- **Framework**: Go Fiber v2 for high-performance HTTP handling
- **Database**: PostgreSQL with connection pooling and native partitioning
- **Message Queue**: NATS JetStream for decoupling ingestion from database writes
- **Authentication**: Bearer token authentication with bcrypt-hashed tokens
- **Background Workers**: Separate goroutines for telemetry processing and command expiration
- **Middleware**: Compression, rate limiting, CORS, logging, and recovery

## Features

- **Device Registration**: Secure agent onboarding with capability negotiation
- **Telemetry Ingestion**: High-throughput ingestion with batch processing and async writes
- **Policy Management**: Dynamic policy distribution with ETag caching
- **Command Execution**: Ad-hoc command issuance and result collection
- **Health Monitoring**: Comprehensive health checks and Prometheus metrics
- **Security**: TLS enforcement, input validation, and audit logging

## Endpoints

### Authentication
All endpoints except `/health` require `Authorization: Bearer <token>` header.

### Core Endpoints

- `POST /v1/agents/register` - Register new agent
- `POST /v1/agents/{id}/inventory` - Ingest telemetry data
- `GET /v1/agents/{id}/policy` - Retrieve effective policy
- `GET /v1/agents/{id}/commands` - Poll for pending commands
- `POST /v1/agents/{id}/commands/{cmd_id}/ack` - Acknowledge command completion

### Management Endpoints (Future)

- `GET /v1/devices` - List devices with filtering
- `GET /v1/devices/{id}` - Get device details
- `POST /v1/policies` - Create/update policies
- `GET /v1/commands` - List commands

### Health & Monitoring

- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

## Configuration

Environment variables:

```bash
DATABASE_URL=postgres://user:pass@localhost:5432/inventory?sslmode=disable
NATS_URL=nats://localhost:4222
API_PORT=8080
JWT_SECRET=your-secure-secret-here
LOG_LEVEL=info
RATE_LIMIT_RPS=100
MAX_BATCH_SIZE=1000
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
```

## Database Schema

Key tables:

- `agents` - Device registry with capabilities and auth tokens
- `telemetry` - Partitioned telemetry data (device_id, collected_at, metrics)
- `telemetry_latest` - Latest metrics per device
- `policies` - Policy definitions with scope hierarchy
- `commands` - Command queue with TTL and status
- `audit_log` - Security and operational events

## Performance

- **Target**: <300ms p95 ingest latency for 10k agents
- **Throughput**: 1000+ requests/second with proper sizing
- **Scaling**: Stateless API servers, horizontal scaling via load balancer
- **Database**: Connection pooling, prepared statements, partition pruning

## Development

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- NATS Server

### Setup

```bash
# Install dependencies
go mod download

# Run migrations
migrate -path internal/database/migrations -database "$DATABASE_URL" up

# Start API server
go run main.go
```

### Testing

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...
```

## Deployment

### Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o api .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api .
CMD ["./api"]
```

### Kubernetes

See `k8s/` directory for deployment manifests with:

- Deployment with rolling updates
- ConfigMap for environment variables
- Secret for TLS certificates
- Service and Ingress
- HorizontalPodAutoscaler
- Network policies

## Security

- **Transport**: TLS 1.2+ enforced
- **Authentication**: Bearer tokens with bcrypt hashing
- **Authorization**: Device-scoped access
- **Input Validation**: Strict JSON schema validation
- **Rate Limiting**: Per-IP and per-device limits
- **Audit Logging**: All policy/command changes logged
- **Data Protection**: Sensitive data encrypted at rest

## Monitoring

### Health Checks

- Database connectivity
- NATS connectivity
- Background worker status

### Metrics

- HTTP request metrics (count, duration, status)
- Database connection pool stats
- Queue depths and processing rates
- Error rates and latency percentiles

### Logging

Structured JSON logging with configurable levels. Includes request IDs, device IDs, and operation context.

## Troubleshooting

### Common Issues

1. **High Latency**: Check database query plans, connection pool settings
2. **Memory Usage**: Monitor goroutine count, connection pools
3. **NATS Connection**: Verify network connectivity, check JetStream status
4. **Rate Limiting**: Adjust RPS limits based on load testing

### Debug Mode

Enable with `LOG_LEVEL=debug` for detailed request/response logging.

## API Evolution

- Versioned endpoints (`/v1/`)
- Backward compatibility maintained
- Deprecation notices in response headers
- Migration guides for breaking changes