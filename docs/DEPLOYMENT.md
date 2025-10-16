# Deployment Guide

## Overview

This guide covers the deployment of the Windows Inventory Agent & Cloud Console system across different environments, from development to production.

## Prerequisites

### System Requirements

#### API Server
- **OS**: Linux (Ubuntu 20.04+), Windows Server 2019+, or macOS
- **CPU**: 2+ cores
- **RAM**: 4GB+ minimum, 8GB+ recommended
- **Storage**: 50GB+ SSD
- **Network**: 100Mbps+ connection

#### Database Server
- **OS**: Linux (Ubuntu 20.04+)
- **CPU**: 4+ cores
- **RAM**: 8GB+ minimum, 16GB+ recommended
- **Storage**: 500GB+ SSD (NVMe preferred)
- **Network**: 1Gbps+ connection

#### Web Console
- **OS**: Linux, Windows, or macOS
- **CPU**: 1+ core
- **RAM**: 2GB+ minimum
- **Storage**: 10GB+ available space

### Software Dependencies

#### Required Software
- **Go**: 1.22+ (for building agent and API)
- **Node.js**: 18+ (for web console)
- **PostgreSQL**: 16+
- **NATS Server**: 2.9+
- **Docker**: 20.10+ (optional, for containerized deployment)
- **Docker Compose**: 2.0+ (for development)

#### Optional Software
- **Kubernetes**: 1.24+ (for production orchestration)
- **Helm**: 3.0+ (for Kubernetes deployments)
- **Terraform**: 1.0+ (for infrastructure as code)
- **Prometheus**: For monitoring
- **Grafana**: For dashboards

## Development Environment

### Quick Start with Docker Compose

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourorg/inventory-agent.git
   cd inventory-agent
   ```

2. **Start development environment:**
   ```bash
   docker-compose up -d
   ```

3. **Verify services are running:**
   ```bash
   docker-compose ps
   ```

4. **View logs:**
   ```bash
   docker-compose logs -f
   ```

### Manual Development Setup

#### Database Setup
```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql
```

```sql
CREATE DATABASE inventory_db;
CREATE USER inventory_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE inventory_db TO inventory_user;
\c inventory_db;
GRANT ALL ON SCHEMA public TO inventory_user;
```

#### NATS Setup
```bash
# Download and install NATS
wget https://github.com/nats-io/nats-server/releases/download/v2.9.0/nats-server-v2.9.0-linux-amd64.zip
unzip nats-server-v2.9.0-linux-amd64.zip
sudo mv nats-server-v2.9.0-linux-amd64/nats-server /usr/local/bin/

# Create systemd service
sudo tee /etc/systemd/system/nats.service > /dev/null <<EOF
[Unit]
Description=NATS Server
After=network.target

[Service]
ExecStart=/usr/local/bin/nats-server -c /etc/nats.conf
Restart=always
User=nats

[Install]
WantedBy=multi-user.target
EOF

# Start NATS
sudo systemctl enable nats
sudo systemctl start nats
```

#### API Server Setup
```bash
cd api

# Install dependencies
go mod download

# Build the application
go build -o bin/api ./cmd/api

# Create configuration
cp config.example.yaml config.yaml
# Edit config.yaml with your settings

# Run the server
./bin/api
```

#### Web Console Setup
```bash
cd web

# Install dependencies
npm install

# Copy environment variables
cp .env.example .env.local
# Edit .env.local with your settings

# Start development server
npm run dev
```

## Production Deployment

### Docker Container Deployment

#### Build Images
```bash
# Build API image
cd api
docker build -t inventory-api:latest .

# Build web console image
cd ../web
docker build -t inventory-web:latest .

# Build agent image (for testing)
cd ../agent
docker build -t inventory-agent:latest .
```

#### Production Docker Compose
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: inventory_db
      POSTGRES_USER: inventory_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/init.sql:/docker-entrypoint-initdb.d/init.sql
    networks:
      - inventory
    restart: unless-stopped

  nats:
    image: nats:2.9-alpine
    command: ["-js", "--cluster_name", "inventory"]
    volumes:
      - nats_data:/data
    networks:
      - inventory
    restart: unless-stopped

  api:
    image: inventory-api:latest
    environment:
      - DATABASE_URL=postgres://inventory_user:${DB_PASSWORD}@postgres:5432/inventory_db?sslmode=disable
      - NATS_URL=nats://nats:4222
      - SERVER_PORT=8080
      - JWT_SECRET=${JWT_SECRET}
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - nats
    networks:
      - inventory
    restart: unless-stopped

  web:
    image: inventory-web:latest
    environment:
      - NEXT_PUBLIC_API_URL=http://api:8080
    ports:
      - "3000:3000"
    depends_on:
      - api
    networks:
      - inventory
    restart: unless-stopped

volumes:
  postgres_data:
  nats_data:

networks:
  inventory:
    driver: bridge
```

### Kubernetes Deployment

#### Namespace Setup
```bash
kubectl create namespace inventory
```

#### PostgreSQL StatefulSet
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: inventory
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16-alpine
        env:
        - name: POSTGRES_DB
          value: "inventory_db"
        - name: POSTGRES_USER
          value: "inventory_user"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
```

#### NATS Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nats
  namespace: inventory
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nats
  template:
    metadata:
      labels:
        app: nats
    spec:
      containers:
      - name: nats
        image: nats:2.9-alpine
        command: ["nats-server", "--cluster_name", "inventory", "--cluster", "nats://0.0.0.0:6222"]
        ports:
        - containerPort: 4222
        - containerPort: 6222
        volumeMounts:
        - name: nats-storage
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /
            port: 8222
          initialDelaySeconds: 10
          periodSeconds: 10
      volumes:
      - name: nats-storage
        persistentVolumeClaim:
          claimName: nats-pvc
```

#### API Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inventory-api
  namespace: inventory
spec:
  replicas: 3
  selector:
    matchLabels:
      app: inventory-api
  template:
    metadata:
      labels:
        app: inventory-api
    spec:
      containers:
      - name: api
        image: inventory-api:latest
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: api-secret
              key: database-url
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: SERVER_PORT
          value: "8080"
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

#### Web Console Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inventory-web
  namespace: inventory
spec:
  replicas: 2
  selector:
    matchLabels:
      app: inventory-web
  template:
    metadata:
      labels:
        app: inventory-web
    spec:
      containers:
      - name: web
        image: inventory-web:latest
        env:
        - name: NEXT_PUBLIC_API_URL
          value: "https://api.yourdomain.com"
        ports:
        - containerPort: 3000
        livenessProbe:
          httpGet:
            path: /
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
```

#### Ingress Configuration
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: inventory-ingress
  namespace: inventory
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
  - hosts:
    - api.yourdomain.com
    - web.yourdomain.com
    secretName: inventory-tls
  rules:
  - host: api.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: inventory-api
            port:
              number: 8080
  - host: web.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: inventory-web
            port:
              number: 3000
```

### Load Balancing

#### Nginx Configuration
```nginx
upstream api_backend {
    server api1.yourdomain.com:8080;
    server api2.yourdomain.com:8080;
    server api3.yourdomain.com:8080;
}

upstream web_backend {
    server web1.yourdomain.com:3000;
    server web2.yourdomain.com:3000;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    ssl_certificate /etc/ssl/certs/api.yourdomain.com.crt;
    ssl_certificate_key /etc/ssl/private/api.yourdomain.com.key;

    location / {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 443 ssl http2;
    server_name web.yourdomain.com;

    ssl_certificate /etc/ssl/certs/web.yourdomain.com.crt;
    ssl_certificate_key /etc/ssl/private/web.yourdomain.com.key;

    location / {
        proxy_pass http://web_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Agent Deployment

### Windows MSI Installation

#### Build MSI Package
```bash
# Install WiX Toolset
# Download from https://wixtoolset.org/

# Build MSI
cd agent
candle.exe installer.wxs
light.exe installer.wixobj

# Sign MSI (optional)
signtool.exe sign /f certificate.pfx /p password inventory-agent.msi
```

#### MSI Installation Script
```powershell
# Download and install agent
$msiUrl = "https://yourdomain.com/downloads/inventory-agent.msi"
$installerPath = "$env:TEMP\inventory-agent.msi"

Invoke-WebRequest -Uri $msiUrl -OutFile $installerPath
Start-Process msiexec.exe -ArgumentList "/i $installerPath /quiet /norestart SERVER_URL=https://api.yourdomain.com" -Wait

# Clean up
Remove-Item $installerPath
```

### Group Policy Deployment

#### Create GPO
```powershell
# Group Policy script for agent deployment
$agentPath = "\\domain.com\NETLOGON\inventory-agent.msi"
$installArgs = "/i `"$agentPath`" /quiet /norestart SERVER_URL=https://api.yourdomain.com"

Start-Process msiexec.exe -ArgumentList $installArgs -Wait
```

### SCCM/MEM Deployment

#### SCCM Package Configuration
```
Package Name: Inventory Agent
Source: \\sccm-server\sources\inventory-agent
Command Line: msiexec /i inventory-agent.msi /quiet /norestart SERVER_URL=https://api.yourdomain.com
Detection Method: File exists - C:\Program Files\Inventory Agent\agent.exe
```

## Configuration Management

### Environment Variables

#### API Server
```bash
# Database
DATABASE_URL=postgres://user:password@localhost:5432/inventory_db

# NATS
NATS_URL=nats://localhost:4222

# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Security
JWT_SECRET=your-jwt-secret-key
API_KEY_SALT=your-api-key-salt

# TLS
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Performance
MAX_CONNECTIONS=100
REQUEST_TIMEOUT=30s
RATE_LIMIT_RPS=100
```

#### Web Console
```bash
# API
NEXT_PUBLIC_API_URL=https://api.yourdomain.com

# Authentication
NEXTAUTH_URL=https://web.yourdomain.com
NEXTAUTH_SECRET=your-nextauth-secret

# Analytics (optional)
NEXT_PUBLIC_ANALYTICS_ID=GA_MEASUREMENT_ID
```

### Secrets Management

#### Kubernetes Secrets
```bash
# Create secrets
kubectl create secret generic postgres-secret \
  --from-literal=password='your-db-password' \
  --namespace inventory

kubectl create secret generic api-secret \
  --from-literal=database-url='postgres://user:password@postgres:5432/inventory_db' \
  --from-literal=jwt-secret='your-jwt-secret' \
  --namespace inventory
```

#### HashiCorp Vault Integration
```hcl
# Vault policy for inventory system
path "secret/inventory/*" {
  capabilities = ["read"]
}

path "database/creds/inventory" {
  capabilities = ["read"]
}
```

## Monitoring and Logging

### Prometheus Metrics

#### API Server Metrics
```yaml
scrape_configs:
  - job_name: 'inventory-api'
    static_configs:
      - targets: ['api1.yourdomain.com:8080', 'api2.yourdomain.com:8080']
    metrics_path: '/metrics'
```

#### Database Metrics
```yaml
scrape_configs:
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres.yourdomain.com:9187']
```

### Centralized Logging

#### Fluentd Configuration
```xml
<source>
  @type tail
  path /var/log/inventory/*.log
  tag inventory.*
</source>

<match inventory.**>
  @type elasticsearch
  host elasticsearch.yourdomain.com
  port 9200
  logstash_format true
</match>
```

### Health Checks

#### API Health Check
```bash
curl -f https://api.yourdomain.com/health
```

#### Database Health Check
```bash
PGPASSWORD=password pg_isready -h localhost -U inventory_user -d inventory_db
```

## Backup and Recovery

### Database Backup
```bash
# Daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
BACKUP_FILE="$BACKUP_DIR/inventory_$DATE.backup"

pg_dump -h localhost -U inventory_user -d inventory_db -F c -b -v > "$BACKUP_FILE"

# Compress
gzip "$BACKUP_FILE"

# Upload to cloud storage
aws s3 cp "${BACKUP_FILE}.gz" s3://your-backup-bucket/
```

### Configuration Backup
```bash
# Backup configurations
tar -czf /backups/config_$(date +%Y%m%d).tar.gz \
  /etc/inventory/ \
  /opt/inventory/config/
```

### Disaster Recovery

#### Recovery Procedure
1. **Stop all services**
   ```bash
   kubectl scale deployment inventory-api --replicas=0
   kubectl scale deployment inventory-web --replicas=0
   ```

2. **Restore database**
   ```bash
   pg_restore -h localhost -U inventory_user -d inventory_db /backups/latest.backup
   ```

3. **Restore configurations**
   ```bash
   tar -xzf /backups/config_latest.tar.gz -C /
   ```

4. **Restart services**
   ```bash
   kubectl scale deployment inventory-api --replicas=3
   kubectl scale deployment inventory-web --replicas=2
   ```

## Security Hardening

### Network Security
```bash
# Firewall rules
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 8080/tcp
ufw --force enable

# SELinux/AppArmor
sudo setenforce 1
```

### SSL/TLS Configuration
```nginx
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
```

### Security Headers
```nginx
add_header X-Frame-Options DENY;
add_header X-Content-Type-Options nosniff;
add_header X-XSS-Protection "1; mode=block";
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";
```

## Performance Tuning

### Database Optimization
```sql
-- PostgreSQL tuning
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET work_mem = '64MB';
ALTER SYSTEM SET maintenance_work_mem = '256MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
```

### API Server Tuning
```yaml
# Go runtime tuning
GOGC=100
GOMAXPROCS=4

# Application tuning
MAX_CONNECTIONS=1000
REQUEST_TIMEOUT=30s
RATE_LIMIT_RPS=1000
```

### Caching Strategy
```nginx
# Static asset caching
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
    expires 1y;
    add_header Cache-Control "public, immutable";
}

# API response caching
location /api/ {
    proxy_cache inventory_cache;
    proxy_cache_valid 200 10m;
    proxy_cache_valid 404 1m;
}
```

## Scaling Strategies

### Horizontal Scaling
```bash
# Add more API instances
kubectl scale deployment inventory-api --replicas=5

# Add more web instances
kubectl scale deployment inventory-web --replicas=3
```

### Database Scaling
```bash
# Add read replicas
kubectl scale statefulset postgres-replica --replicas=2

# Connection pooling
kubectl apply -f pgbouncer.yaml
```

### Auto-scaling
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: inventory-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: inventory-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

This deployment guide provides comprehensive instructions for setting up the Windows Inventory Agent & Cloud Console system in various environments, from development to production, with considerations for scalability, security, and maintainability.