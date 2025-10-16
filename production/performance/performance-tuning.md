# Windows Inventory Agent Performance Optimization Guide

## 1. System Requirements

### Minimum Requirements
- CPU: 2 cores @ 2.4 GHz
- RAM: 4 GB
- Disk: 20 GB SSD
- Network: 100 Mbps

### Recommended Requirements
- CPU: 4 cores @ 3.0 GHz
- RAM: 8 GB
- Disk: 50 GB SSD
- Network: 1 Gbps

## 2. Database Optimization

### PostgreSQL Configuration
```sql
-- postgresql.conf optimizations
shared_buffers = 256MB                    # 25% of RAM
effective_cache_size = 1GB               # 75% of RAM
work_mem = 4MB                           # Per connection
maintenance_work_mem = 64MB              # For maintenance operations
checkpoint_completion_target = 0.9        # Spread checkpoint I/O
wal_buffers = 16MB                       # WAL buffer size
default_statistics_target = 100          # Statistics target
random_page_cost = 1.1                   # SSD optimization
effective_io_concurrency = 200           # SSD optimization
```

### Connection Pooling
```yaml
database:
  max_open_connections: 25
  max_idle_connections: 5
  connection_max_lifetime: 1h
  connection_max_idle_time: 30m
```

### Indexing Strategy
```sql
-- Core indexes for performance
CREATE INDEX CONCURRENTLY idx_telemetry_device_timestamp
    ON telemetry (device_id, timestamp DESC);

CREATE INDEX CONCURRENTLY idx_telemetry_timestamp
    ON telemetry (timestamp DESC);

CREATE INDEX CONCURRENTLY idx_policies_target_filters
    ON policies USING GIN (target_filters);

CREATE INDEX CONCURRENTLY idx_commands_device_status
    ON commands (device_id, status, created_at DESC);

CREATE INDEX CONCURRENTLY idx_audit_logs_timestamp
    ON audit_logs (timestamp DESC, event_type);
```

## 3. API Performance Tuning

### HTTP Server Configuration
```yaml
http:
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  max_header_bytes: 1048576  # 1MB
  max_concurrent_connections: 1000
```

### Caching Strategy
```yaml
cache:
  redis:
    enabled: true
    ttl:
      policies: 300s    # 5 minutes
      devices: 60s     # 1 minute
      sessions: 3600s  # 1 hour

  in_memory:
    enabled: true
    size: 100MB
    ttl:
      schema_validation: 3600s  # 1 hour
```

### Request Processing
```yaml
processing:
  worker_pool_size: 10
  queue_size: 100
  timeout: 30s
  retry_attempts: 3
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    reset_timeout: 60s
```

## 4. Agent Performance Optimization

### Collection Intervals
```yaml
agent:
  collection:
    telemetry:
      interval: 300s  # 5 minutes
      timeout: 60s
    inventory:
      full_scan_interval: 86400s  # 24 hours
      incremental_scan_interval: 3600s  # 1 hour
      timeout: 300s
```

### Resource Limits
```yaml
agent:
  limits:
    cpu_percent: 10
    memory_mb: 256
    disk_io_mbps: 50
    network_mbps: 10
```

### Batch Processing
```yaml
agent:
  batching:
    telemetry_batch_size: 100
    telemetry_flush_interval: 30s
    command_batch_size: 10
    retry_batch_delay: 5s
```

## 5. Network Optimization

### Connection Pooling
```yaml
network:
  http_client:
    max_idle_connections: 100
    max_idle_connections_per_host: 10
    idle_connection_timeout: 90s
    tls_handshake_timeout: 10s
    expect_continue_timeout: 1s
```

### Compression
```yaml
compression:
  enabled: true
  algorithms:
    - gzip
    - deflate
  min_size: 1024  # 1KB
  level: 6
```

## 6. Monitoring and Alerting

### Performance Metrics
```yaml
metrics:
  collection_interval: 15s
  retention_period: 30d
  alerting:
    cpu_threshold: 80
    memory_threshold: 85
    disk_threshold: 90
    latency_threshold_p95: 500ms
    error_rate_threshold: 0.05
```

### Profiling
```yaml
profiling:
  enabled: true
  cpu_profile_duration: 30s
  memory_profile_interval: 5m
  goroutine_dump_interval: 10m
```

## 7. Scaling Guidelines

### Vertical Scaling
- Increase CPU cores for better parallel processing
- Add RAM for larger caches and connection pools
- Use faster storage (NVMe) for database performance

### Horizontal Scaling
- Deploy multiple API instances behind load balancer
- Use database read replicas for query-heavy workloads
- Implement Redis clustering for cache scaling

### Auto-scaling Rules
```yaml
autoscaling:
  cpu_based:
    scale_up_threshold: 70
    scale_down_threshold: 30
    cooldown_period: 300s
  request_based:
    scale_up_threshold: 1000  # requests per minute
    scale_down_threshold: 200
    cooldown_period: 300s
```

## 8. Performance Testing

### Load Testing Scenarios
```yaml
load_tests:
  standard:
    vus: 100
    duration: 10m
    thresholds:
      http_req_duration: ['p(95)<500ms']
      http_req_failed: ['rate<0.05']

  stress:
    vus: 500
    duration: 5m
    thresholds:
      http_req_duration: ['p(95)<2000ms']
      http_req_failed: ['rate<0.1']

  endurance:
    vus: 50
    duration: 1h
    thresholds:
      http_req_duration: ['p(95)<300ms']
      http_req_failed: ['rate<0.01']
```

### Benchmark Commands
```bash
# API benchmarks
ab -n 10000 -c 100 http://localhost:8080/api/v1/telemetry

# Database benchmarks
pgbench -c 10 -j 2 -T 60 inventory_db

# Memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap

# CPU profiling
go tool pprof http://localhost:8080/debug/pprof/profile
```

## 9. Troubleshooting Performance Issues

### High CPU Usage
1. Check goroutine count: `runtime.NumGoroutine()`
2. Profile CPU usage with pprof
3. Review database query performance
4. Check for memory leaks

### High Memory Usage
1. Monitor garbage collection: `runtime.GC()`
2. Check for memory leaks with pprof
3. Review cache sizes and TTL settings
4. Monitor connection pool usage

### High Latency
1. Check database query performance
2. Review network latency between components
3. Monitor queue depths and processing delays
4. Check for resource contention

### High Error Rates
1. Review application logs for error patterns
2. Check database connection issues
3. Monitor external service dependencies
4. Review rate limiting and circuit breaker settings

## 10. Maintenance Tasks

### Regular Maintenance
```bash
# Database maintenance
VACUUM ANALYZE;
REINDEX DATABASE inventory_db;

# Cache maintenance
FLUSHDB ASYNC;

# Log rotation
logrotate /etc/logrotate.d/inventory-agent
```

### Performance Monitoring
- Daily: Check key metrics and alerts
- Weekly: Review performance trends
- Monthly: Run comprehensive load tests
- Quarterly: Review and update performance baselines