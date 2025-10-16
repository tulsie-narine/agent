# Security Guide

## Overview

Security is a critical aspect of the Windows Inventory Agent & Cloud Console system. This guide outlines security considerations, best practices, and implementation details for protecting sensitive inventory data and ensuring secure communication between components.

## Security Principles

### Defense in Depth
The system implements multiple layers of security:
- **Network Security**: TLS encryption, firewall rules, network segmentation
- **Application Security**: Input validation, authentication, authorization
- **Data Security**: Encryption at rest, access controls, audit logging
- **Operational Security**: Secure deployment, monitoring, incident response

### Zero Trust Architecture
- No implicit trust between components
- Every request requires authentication and authorization
- Continuous verification of security posture
- Minimal privilege access principles

## Authentication & Authorization

### Agent Authentication

#### API Key Authentication
Agents use API keys for authentication with the cloud API:

```go
// Agent registration generates unique API key
type RegistrationResponse struct {
    ID     string `json:"id"`
    APIKey string `json:"api_key"`
    Config struct {
        TelemetryInterval int `json:"telemetry_interval"`
        CommandPollInterval int `json:"command_poll_interval"`
    } `json:"config"`
}
```

**Security Features:**
- API keys are hashed using bcrypt before storage
- Keys are generated using cryptographically secure random generation
- Keys can be rotated without service interruption
- Failed authentication attempts are rate limited

#### Key Management
```go
// internal/auth/key.go
func GenerateAPIKey() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}

func HashAPIKey(key string) (string, error) {
    return bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
}

func VerifyAPIKey(hashedKey, providedKey string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedKey), []byte(providedKey))
}
```

### User Authentication

#### JWT Token Authentication
Web console users authenticate using JWT tokens:

```typescript
// src/lib/auth.ts
export interface JWTPayload {
  userId: string
  username: string
  role: 'admin' | 'operator' | 'viewer'
  exp: number
  iat: number
}

export function generateJWT(payload: Omit<JWTPayload, 'exp' | 'iat'>): string {
  const secret = process.env.JWT_SECRET
  return jwt.sign(
    {
      ...payload,
      iat: Math.floor(Date.now() / 1000),
      exp: Math.floor(Date.now() / 1000) + (24 * 60 * 60), // 24 hours
    },
    secret,
    { algorithm: 'HS256' }
  )
}
```

**Security Features:**
- Tokens expire after 24 hours
- Refresh tokens for seamless user experience
- Token blacklisting for logout/revocation
- Secure token storage in httpOnly cookies

### Role-Based Access Control (RBAC)

#### User Roles
```sql
-- User roles with hierarchical permissions
CREATE TYPE user_role AS ENUM ('admin', 'operator', 'viewer');

-- Role permissions matrix
CREATE TABLE role_permissions (
    role user_role PRIMARY KEY,
    permissions JSONB NOT NULL
);

INSERT INTO role_permissions VALUES
('viewer', '["devices:read", "telemetry:read"]'),
('operator', '["devices:read", "telemetry:read", "commands:create", "policies:read"]'),
('admin', '["*"]');
```

#### Permission Checking
```go
// internal/auth/middleware.go
func AuthMiddleware(db *pgxpool.Pool) fiber.Handler {
    return func(c *fiber.Ctx) error {
        auth := c.Get("Authorization")
        if auth == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "Missing authorization header",
            })
        }

        token := strings.TrimPrefix(auth, "Bearer ")
        claims, err := ValidateJWT(token)
        if err != nil {
            return c.Status(401).JSON(fiber.Map{
                "error": "Invalid token",
            })
        }

        // Check permissions for the requested resource
        resource := getResourceFromPath(c.Path())
        action := getActionFromMethod(c.Method())

        if !hasPermission(claims.Role, resource, action) {
            return c.Status(403).JSON(fiber.Map{
                "error": "Insufficient permissions",
            })
        }

        c.Locals("user", claims)
        return c.Next()
    }
}
```

## Data Protection

### Encryption at Rest

#### Database Encryption
```sql
-- Enable encryption for sensitive columns
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Encrypted API keys
ALTER TABLE devices ADD COLUMN api_key_encrypted BYTEA;

-- Encryption functions
CREATE OR REPLACE FUNCTION encrypt_api_key(plain_key TEXT)
RETURNS BYTEA AS $$
BEGIN
    RETURN pgp_sym_encrypt(plain_key, current_setting('encryption.key'));
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION decrypt_api_key(encrypted_key BYTEA)
RETURNS TEXT AS $$
BEGIN
    RETURN pgp_sym_decrypt(encrypted_key, current_setting('encryption.key'));
END;
$$ LANGUAGE plpgsql;
```

#### Configuration Encryption
Sensitive configuration values are encrypted:

```go
// internal/config/secure.go
func (c *Config) EncryptSensitiveFields() error {
    if c.DatabasePassword != "" {
        encrypted, err := encryptValue(c.DatabasePassword)
        if err != nil {
            return err
        }
        c.DatabasePassword = encrypted
    }

    if c.JWT.Secret != "" {
        encrypted, err := encryptValue(c.JWT.Secret)
        if err != nil {
            return err
        }
        c.JWT.Secret = encrypted
    }

    return nil
}
```

### Encryption in Transit

#### TLS Configuration
All communications use TLS 1.2+ with strong cipher suites:

```go
// API server TLS configuration
func createTLSConfig() *tls.Config {
    return &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
        PreferServerCipherSuites: true,
        CurvePreferences: []tls.CurveID{
            tls.CurveP256,
            tls.X25519,
        },
    }
}
```

#### Certificate Management
```bash
# Generate self-signed certificate for development
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Production certificates from Let's Encrypt
certbot certonly --webroot -w /var/www/html -d api.yourdomain.com
certbot certonly --webroot -w /var/www/html -d web.yourdomain.com
```

### Data Sanitization

#### Input Validation
All inputs are validated and sanitized:

```go
// internal/models/device.go
func (d *Device) Validate() error {
    if d.Hostname == "" {
        return errors.New("hostname is required")
    }

    // Validate hostname format
    if !hostnameRegex.MatchString(d.Hostname) {
        return errors.New("invalid hostname format")
    }

    // Sanitize string fields
    d.Hostname = sanitizeString(d.Hostname)
    d.OSVersion = sanitizeString(d.OSVersion)

    return nil
}

var hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
```

#### SQL Injection Prevention
Using parameterized queries and prepared statements:

```go
// Safe query execution
func (r *DeviceRepository) GetByHostname(ctx context.Context, hostname string) (*Device, error) {
    query := `
        SELECT id, hostname, os_version, os_build, architecture, status, created_at, updated_at
        FROM devices
        WHERE hostname = $1
    `

    var device Device
    err := r.db.QueryRow(ctx, query, hostname).Scan(
        &device.ID, &device.Hostname, &device.OSVersion, &device.OSBuild,
        &device.Architecture, &device.Status, &device.CreatedAt, &device.UpdatedAt,
    )

    return &device, err
}
```

## Network Security

### Firewall Configuration

#### API Server Firewall
```bash
# UFW rules for API server
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 8080/tcp  # API port
ufw allow 8443/tcp  # HTTPS API port
ufw --force enable
```

#### Database Firewall
```bash
# PostgreSQL pg_hba.conf
# Only allow local connections and specific IPs
local   all             postgres                                peer
host    inventory_db    inventory_user     10.0.0.0/8          md5
host    inventory_db    inventory_user     172.16.0.0/12       md5
host    inventory_db    inventory_user     192.168.0.0/16      md5
```

### Network Segmentation

#### Kubernetes Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-network-policy
  namespace: inventory
spec:
  podSelector:
    matchLabels:
      app: inventory-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: inventory-web
    ports:
    - protocol: TCP
      port: 8080
  - from:
    - ipBlock:
        cidr: 0.0.0.0/0  # Allow external agent connections
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: nats
    ports:
    - protocol: TCP
      port: 4222
```

## Security Monitoring

### Audit Logging

#### Database Audit Triggers
```sql
-- Audit table for sensitive operations
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id UUID,
    device_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT
);

-- Audit trigger function
CREATE OR REPLACE FUNCTION audit_trigger_func()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO audit_log (
        user_id, device_id, action, resource_type, resource_id,
        old_values, new_values, ip_address, user_agent
    ) VALUES (
        current_setting('app.user_id', true)::UUID,
        current_setting('app.device_id', true)::UUID,
        TG_OP,
        TG_TABLE_NAME,
        CASE WHEN TG_OP != 'INSERT' THEN OLD.id ELSE NEW.id END,
        CASE WHEN TG_OP != 'INSERT' THEN row_to_json(OLD) ELSE NULL END,
        CASE WHEN TG_OP != 'DELETE' THEN row_to_json(NEW) ELSE NULL END,
        inet_client_addr(),
        current_setting('app.user_agent', true)
    );

    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

-- Apply audit triggers
CREATE TRIGGER audit_devices_trigger
    AFTER INSERT OR UPDATE OR DELETE ON devices
    FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
```

#### Application Logging
```go
// Structured logging with security events
type SecurityLogger struct {
    logger *zap.Logger
}

func (l *SecurityLogger) LogSecurityEvent(event SecurityEvent) {
    l.logger.Info("Security Event",
        zap.String("event_type", event.Type),
        zap.String("user_id", event.UserID),
        zap.String("ip_address", event.IPAddress),
        zap.Time("timestamp", event.Timestamp),
        zap.Any("details", event.Details),
    )
}

// Security event types
type SecurityEvent struct {
    Type      string
    UserID    string
    IPAddress string
    Timestamp time.Time
    Details   map[string]interface{}
}
```

### Intrusion Detection

#### Fail2Ban Configuration
```ini
# /etc/fail2ban/jail.d/inventory.conf
[inventory-api]
enabled = true
port = 8080
filter = inventory-api
logpath = /var/log/inventory/api.log
maxretry = 3
bantime = 3600
```

#### Rate Limiting
```go
// API rate limiting middleware
func RateLimitMiddleware(rps int) fiber.Handler {
    limiter := tollbooth.NewLimiter(float64(rps), nil)
    limiter.SetIPLookups([]string{"X-Real-IP", "X-Forwarded-For", "CF-Connecting-IP"})
    limiter.SetMethods([]string{"GET", "POST", "PUT", "DELETE"})

    return func(c *fiber.Ctx) error {
        httpError := tollbooth.LimitByRequest(limiter, c.Context())
        if httpError != nil {
            return c.Status(httpError.StatusCode).JSON(fiber.Map{
                "error": "Rate limit exceeded",
            })
        }
        return c.Next()
    }
}
```

## Vulnerability Management

### Dependency Scanning

#### Go Module Vulnerabilities
```bash
# Check for known vulnerabilities
go mod download
govulncheck ./...

# Update dependencies securely
go get -u ./...
go mod tidy

# Check for outdated dependencies
go list -u -m all
```

#### NPM Audit
```bash
cd web

# Audit dependencies
npm audit

# Fix vulnerabilities
npm audit fix

# Update dependencies
npm update
```

### Container Security

#### Docker Image Security
```dockerfile
# Use minimal base image
FROM golang:1.22-alpine AS builder

# Install security updates
RUN apk update && apk upgrade

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Build application
WORKDIR /app
COPY . .
RUN go build -o main .

# Production image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /app/main /main

USER appuser
EXPOSE 8080
CMD ["/main"]
```

#### Image Scanning
```bash
# Scan images for vulnerabilities
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  anchore/grype inventory-api:latest

# Trivy comprehensive scan
trivy image --exit-code 1 --no-progress inventory-api:latest
trivy fs --security-checks vuln,secret,misconfig .
```

## Incident Response

### Security Incident Procedure

#### Detection Phase
1. **Monitor alerts** from security tools
2. **Review logs** for suspicious activity
3. **Assess impact** of potential breach

#### Containment Phase
1. **Isolate affected systems**
2. **Revoke compromised credentials**
3. **Block malicious IP addresses**

#### Recovery Phase
1. **Restore from clean backups**
2. **Update security measures**
3. **Monitor for reoccurrence**

#### Lessons Learned Phase
1. **Document incident details**
2. **Update security procedures**
3. **Implement preventive measures**

### Incident Response Plan
```yaml
# incident_response.yml
version: 1.0
contacts:
  - name: Security Team Lead
    email: security@company.com
    phone: +1-555-0123
  - name: DevOps Lead
    email: devops@company.com
    phone: +1-555-0124

escalation_matrix:
  severity_1: # Critical - immediate response required
    - Notify all contacts
    - Begin containment procedures
    - Customer communication within 1 hour
  severity_2: # High - response within 4 hours
    - Notify technical leads
    - Assess business impact
  severity_3: # Medium - response within 24 hours
    - Notify relevant teams
    - Monitor situation

communication_template: |
  Subject: Security Incident - {severity} - {incident_id}

  Incident Details:
  - ID: {incident_id}
  - Severity: {severity}
  - Detection Time: {detection_time}
  - Affected Systems: {affected_systems}
  - Current Status: {status}

  Immediate Actions Taken:
  {actions_taken}

  Next Steps:
  {next_steps}
```

## Compliance Considerations

### Data Privacy (GDPR/CCPA)

#### Data Minimization
- Collect only necessary inventory data
- Implement data retention policies
- Provide data deletion capabilities

#### User Rights
```go
// Data export for user access requests
func (h *DevicesHandler) ExportUserData(c *fiber.Ctx) error {
    userID := c.Locals("user").(*User).ID

    // Export all data associated with user
    data := map[string]interface{}{
        "profile": h.getUserProfile(userID),
        "devices": h.getUserDevices(userID),
        "audit_log": h.getUserAuditLog(userID),
    }

    return c.JSON(ApiResponse{
        Data: data,
        Success: true,
    })
}

// Data deletion for right to be forgotten
func (h *DevicesHandler) DeleteUserData(c *fiber.Ctx) error {
    userID := c.Params("userId")

    // Anonymize or delete user data
    err := h.anonymizeUserData(userID)
    if err != nil {
        return err
    }

    return c.JSON(ApiResponse{
        Message: "User data deleted successfully",
        Success: true,
    })
}
```

### Security Standards

#### OWASP Compliance
- **A01:2021-Broken Access Control**: RBAC implementation
- **A02:2021-Cryptographic Failures**: TLS and data encryption
- **A03:2021-Injection**: Parameterized queries and input validation
- **A05:2021-Security Misconfiguration**: Secure defaults and hardening
- **A07:2021-Identification and Authentication Failures**: JWT and API key security

#### CIS Benchmarks
- **Database Security**: PostgreSQL CIS benchmark compliance
- **Container Security**: Docker CIS benchmark implementation
- **System Security**: Linux CIS benchmark adherence

## Security Testing

### Automated Security Testing

#### SAST (Static Application Security Testing)
```yaml
# .github/workflows/security.yml
name: Security Scan

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  security:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Run Gosec Security Scanner
      uses: securecodewarrior/github-action-gosec@master
      with:
        args: './...'

    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'
```

#### Penetration Testing
```bash
# Automated penetration testing
docker run --rm -it \
  -v $(pwd):/zap/wrk \
  owasp/zap2docker-stable zap-full-scan.py \
  -t https://api.yourdomain.com \
  -r penetration_test_report.html \
  -x penetration_test_report.xml
```

### Security Monitoring Dashboard

#### Grafana Security Dashboard
```json
{
  "dashboard": {
    "title": "Security Monitoring",
    "panels": [
      {
        "title": "Failed Authentication Attempts",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(failed_auth_attempts_total[5m])",
            "legendFormat": "Failed Auth"
          }
        ]
      },
      {
        "title": "Suspicious API Calls",
        "type": "table",
        "targets": [
          {
            "expr": "rate(suspicious_api_calls_total[5m])",
            "legendFormat": "Suspicious Calls"
          }
        ]
      }
    ]
  }
}
```

This comprehensive security guide ensures the Windows Inventory Agent & Cloud Console system maintains robust protection against threats while meeting compliance requirements and industry best practices.