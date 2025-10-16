# Windows Inventory Agent Testing Tools

This directory contains automated testing tools for the Windows Inventory Agent system.

## Load Testing with k6

The `load-test.js` script provides comprehensive load testing for the Windows Inventory Agent API using k6.

### Prerequisites

1. Install k6: https://k6.io/docs/get-started/installation/
   ```powershell
   choco install k6
   ```

2. Ensure the API server is running and accessible

### Test Scenarios

- **Standard Load Test**: Gradual ramp-up to 100 concurrent users over 22 minutes
- **Stress Test**: Rapid ramp-up to 200 concurrent users to test system limits
- **Spike Test**: Sudden spike to 500 users to test resilience
- **Endurance Test**: 50 concurrent users for 30 minutes to test stability

### Running Tests

Use the `run-load-tests.ps1` script to execute tests:

```powershell
# Run standard load test
.\run-load-tests.ps1 -TestType standard

# Run stress test
.\run-load-tests.ps1 -TestType stress

# Run all test scenarios
.\run-load-tests.ps1 -TestType all

# Specify custom API URL
.\run-load-tests.ps1 -TestType standard -BaseUrl "https://api.example.com/api/v1"

# Dry run to validate test script
.\run-load-tests.ps1 -TestType standard -DryRun

# Verbose output
.\run-load-tests.ps1 -TestType standard -Verbose
```

### Test Configuration

The `k6-config.yaml` file contains test configuration including:

- Performance thresholds (95% of requests < 500ms, error rate < 5%)
- Test scenarios with different load patterns
- Output configuration for results and metrics
- Environment variables for API endpoints

### Test Results

Results are saved to the `results/` directory:

- `load-test-{scenario}-{timestamp}.json`: Detailed test results
- `summary-{scenario}-{timestamp}.txt`: Test summary and metrics

### Interpreting Results

Key metrics to monitor:

- **http_req_duration**: Response time percentiles
- **http_req_failed**: Error rate
- **telemetry_duration**: Telemetry submission performance
- **policy_duration**: Policy retrieval performance
- **command_duration**: Command operation performance

### Customizing Tests

To modify test scenarios:

1. Edit `k6-config.yaml` to adjust load patterns and thresholds
2. Modify `load-test.js` to change test data or add new scenarios
3. Update the `run-load-tests.ps1` script for additional options

## Performance Benchmarks

Expected performance under normal load:

- Telemetry submission: < 200ms p95
- Policy retrieval: < 100ms p95
- Command operations: < 150ms p95
- Overall error rate: < 5%

## Troubleshooting

### Common Issues

1. **k6 not found**: Ensure k6 is installed and in PATH
2. **Connection refused**: Check that the API server is running
3. **Authentication errors**: Update AUTH_TOKEN in configuration
4. **High error rates**: Check API server logs and resource usage

### Debugging

Use the `-Verbose` flag for detailed output:

```powershell
.\run-load-tests.ps1 -TestType standard -Verbose
```

Check API server logs during test execution for errors.