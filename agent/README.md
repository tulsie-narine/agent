# Inventory Agent

Windows service for collecting system inventory and telemetry data. Designed for low resource usage (<1% CPU, <60 MB RAM) and resilient operation across thousands of devices.

## Architecture

The agent runs as a Windows service with the following components:

- **Service Lifecycle**: Install/start/stop/uninstall using standard Windows service management
- **Collectors**: Modular system for gathering OS info, software inventory, CPU/memory/disk metrics
- **Scheduler**: Interval-based collection with configurable jitter to avoid thundering herd
- **Output Writers**: Local JSON output (Phase 1) and HTTP client for cloud posting (Phase 4+)
- **Policy Engine**: Dynamic reconfiguration based on server-distributed policies (Phase 5+)
- **Command Poller**: Ad-hoc collection and remote command execution (Phase 7)

## Features

- **Secure Registration**: UUID-based device identification with token authentication
- **Store-and-Forward**: Local queuing when network unavailable with exponential backoff
- **Capability Negotiation**: Reports supported collectors for policy validation
- **ETag Caching**: Efficient policy updates with conditional requests
- **Graceful Shutdown**: Proper cleanup on service stop/restart

## Installation

### Build Requirements

- Go 1.22+
- Windows SDK (for CGO compilation)

### Building

```bash
# From project root
make build-agent
# Output: ./dist/agent.exe
```

### MSI Package (Phase 8)

```powershell
# Build MSI installer
make msi-package
# Install silently
msiexec /i inventory-agent-1.0.0.msi /quiet
```

### Manual Installation

```cmd
# Install service
agent.exe --install

# Start service
agent.exe --start

# Check status
sc query InventoryAgent

# Stop service
agent.exe --stop

# Uninstall service
agent.exe --uninstall
```

## Configuration

Configuration file location: `C:\ProgramData\InventoryAgent\config.json`

```json
{
  "device_id": "uuid-generated-on-first-run",
  "api_endpoint": "https://your-api-endpoint.com",
  "auth_token": "token-from-registration",
  "collection_interval": "15m",
  "enabled_metrics": {
    "os.info": true,
    "cpu.utilization": false,
    "memory.usage": false,
    "disk.utilization": false,
    "software.inventory": false
  },
  "local_output_path": "C:\\ProgramData\\InventoryAgent\\inventory.json",
  "log_level": "info",
  "retry_config": {
    "max_retries": 5,
    "backoff_multiplier": 2.0,
    "max_backoff": "5m"
  }
}
```

## Operation

### Service Account

Runs under `LocalSystem` account with minimal privileges. No network access required for local operation.

### Data Collection

- **OS Info**: Caption, version, computer make/model, serial number, hostname, domain, last logged-in user
- **Software Inventory**: Installed programs from Windows registry (HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall)
- **CPU Utilization**: Average processor usage percentage
- **Memory Usage**: Used/total physical memory in bytes
- **Disk Utilization**: Per-drive usage statistics (name, total, free, used bytes)

### Telemetry Payload

```json
{
  "device_id": "uuid",
  "agent_version": "1.0.0",
  "collected_at": "2025-01-01T12:00:00Z",
  "metrics": {
    "os.info": {
      "caption": "Microsoft Windows 11 Pro",
      "version": "10.0.22621",
      "make": "Dell Inc.",
      "model": "Latitude 5420",
      "serial": "ABC123",
      "hostname": "WORKSTATION01",
      "domain": "CORP.LOCAL",
      "last_user": "john.doe"
    },
    "cpu.utilization": {
      "cpu_percent": 15.5
    },
    "memory.usage": {
      "used_bytes": 4294967296,
      "total_bytes": 17179869184
    },
    "disk.utilization": [
      {
        "name": "C:",
        "total_bytes": 1000204886016,
        "free_bytes": 500104243008,
        "used_bytes": 500100642008
      }
    ],
    "software.inventory": [
      {
        "name": "Google Chrome",
        "version": "120.0.6099.109",
        "publisher": "Google LLC",
        "install_date": "2024-12-01"
      }
    ]
  }
}
```

## Logging

Logs are written to Windows Event Log and optionally to file. Log levels: debug, info, warn, error.

## Troubleshooting

### Common Issues

1. **Service won't start**: Check permissions on config directory, verify config.json syntax
2. **Collection fails**: WMI services may be disabled; check `services.msc` for Windows Management Instrumentation
3. **Network errors**: Verify API endpoint accessibility, check proxy settings
4. **High CPU**: Reduce collection interval or disable resource-intensive collectors

### Diagnostic Commands

```cmd
# View service status
sc query InventoryAgent

# View event logs
eventvwr.msc > Windows Logs > Application

# Manual collection (for testing)
agent.exe --config config.json --collect-now
```

## Performance

- **CPU**: <1% average utilization
- **Memory**: <60 MB resident set
- **Disk**: Minimal I/O, atomic writes for reliability
- **Network**: Configurable intervals, gzip compression for large payloads

## Security

- No elevated privileges required
- Registry access limited to uninstall keys
- HTTPS-only communication with server
- Token-based authentication
- Future: Binary signing, integrity checks

## Development

### Testing

```bash
make test-agent
```

### Local Development

```bash
# Run without service installation
make run-agent
```

### Code Structure

```
internal/
├── config/          # Configuration management
├── collectors/      # Metric collection modules
├── scheduler/       # Collection orchestration
├── output/          # Writers for local/cloud output
├── policy/          # Policy management and application
├── capability/      # Capability reporting
├── command/         # Command polling and execution
└── registration/    # Device registration logic
```