# Testing Strategy

## Overview

Comprehensive testing strategy covering unit tests, integration tests, performance tests, and security tests to ensure the Windows Inventory Agent & Cloud Console system meets quality and reliability standards.

## Testing Pyramid

```
End-to-End Tests (E2E)
    ↕️
Integration Tests
    ↕️
Component Tests
    ↕️
Unit Tests
```

## Unit Testing

### Agent Unit Tests

#### Collector Tests
```go
// internal/collectors/os_test.go
func TestOSCollector_Collect(t *testing.T) {
    collector := &OSCollector{}

    data, err := collector.Collect()
    assert.NoError(t, err)
    assert.Contains(t, data, "version")
    assert.Contains(t, data, "build")
    assert.Contains(t, data, "architecture")
}

func TestOSCollector_Collect_InvalidData(t *testing.T) {
    collector := &OSCollector{}
    // Mock Windows API failure
    // Verify error handling
}
```

#### Scheduler Tests
```go
// internal/scheduler/scheduler_test.go
func TestScheduler_Start(t *testing.T) {
    scheduler := NewScheduler(1 * time.Minute)

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    err := scheduler.Start(ctx)
    assert.NoError(t, err)
    // Verify telemetry collection occurred
}
```

### API Unit Tests

#### Handler Tests
```go
// internal/handlers/devices_test.go
func TestDevicesHandler_GetDevices(t *testing.T) {
    // Mock database
    db := &mockDB{}
    handler := NewDevicesHandler(db)

    req := httptest.NewRequest("GET", "/devices?page=1&limit=10", nil)
    w := httptest.NewRecorder()

    err := handler.GetDevices(fiber.New().AcquireCtx(req, w))
    assert.NoError(t, err)

    var response ApiResponse[[]Device]
    err = json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.True(t, response.Success)
}
```

#### Model Tests
```go
// internal/models/device_test.go
func TestDevice_Validate(t *testing.T) {
    tests := []struct {
        name    string
        device  Device
        wantErr bool
    }{
        {
            name: "valid device",
            device: Device{
                Hostname:      "WIN-ABC123",
                OSVersion:     "Windows 10 Pro",
                OSBuild:       "19045.2006",
                Architecture:  "x64",
                Status:        "online",
            },
            wantErr: false,
        },
        {
            name: "missing hostname",
            device: Device{
                OSVersion: "Windows 10 Pro",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.device.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Web Console Unit Tests

#### Component Tests
```typescript
// src/components/dashboard/DashboardStats.test.tsx
import { render, screen } from '@testing-library/react'
import { DashboardStats } from './DashboardStats'

const mockApiClient = {
  getDeviceStats: jest.fn(),
}

jest.mock('@/lib/api/client', () => ({
  apiClient: mockApiClient,
}))

describe('DashboardStats', () => {
  it('displays loading state initially', () => {
    mockApiClient.getDeviceStats.mockResolvedValue({
      data: { total: 0, online: 0, offline: 0, unknown: 0 },
      success: true,
    })

    render(<DashboardStats />)

    expect(screen.getByText('Total Devices')).toBeInTheDocument()
  })

  it('displays stats after loading', async () => {
    mockApiClient.getDeviceStats.mockResolvedValue({
      data: { total: 150, online: 140, offline: 5, unknown: 5 },
      success: true,
    })

    render(<DashboardStats />)

    await waitFor(() => {
      expect(screen.getByText('150')).toBeInTheDocument()
    })
  })
})
```

#### Utility Function Tests
```typescript
// src/lib/utils/index.test.ts
import { formatBytes, formatRelativeTime, getDeviceStatus } from './index'

describe('formatBytes', () => {
  it('formats bytes correctly', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1024)).toBe('1 KB')
    expect(formatBytes(1024 * 1024)).toBe('1 MB')
    expect(formatBytes(1024 * 1024 * 1024)).toBe('1 GB')
  })
})

describe('getDeviceStatus', () => {
  const now = new Date()

  it('returns online for recent activity', () => {
    const recentTime = new Date(now.getTime() - 1000 * 60 * 2) // 2 minutes ago
    expect(getDeviceStatus(recentTime.toISOString())).toBe('online')
  })

  it('returns offline for old activity', () => {
    const oldTime = new Date(now.getTime() - 1000 * 60 * 60 * 24 * 2) // 2 days ago
    expect(getDeviceStatus(oldTime.toISOString())).toBe('offline')
  })
})
```

## Integration Testing

### API Integration Tests

#### Database Integration
```go
// internal/database/integration_test.go
func TestDatabase_Connection(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    db, err := Connect(testDBURL)
    assert.NoError(t, err)
    defer db.Close()

    err = db.Ping(context.Background())
    assert.NoError(t, err)
}

func TestDeviceRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    db, cleanup := setupTestDB(t)
    defer cleanup

    repo := NewDeviceRepository(db)

    device := &Device{
        Hostname:     "TEST-001",
        OSVersion:    "Windows 10 Pro",
        Architecture: "x64",
        Status:       "online",
    }

    err := repo.Create(context.Background(), device)
    assert.NoError(t, err)
    assert.NotEmpty(t, device.ID)

    retrieved, err := repo.GetByID(context.Background(), device.ID)
    assert.NoError(t, err)
    assert.Equal(t, device.Hostname, retrieved.Hostname)
}
```

#### NATS Integration
```go
// internal/workers/telemetry_writer_integration_test.go
func TestTelemetryWriter_NATS_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    natsConn, err := nats.Connect(nats.DefaultURL)
    assert.NoError(t, err)
    defer natsConn.Close()

    db, cleanup := setupTestDB(t)
    defer cleanup

    writer := workers.NewTelemetryWriter(db, natsConn)

    // Publish test telemetry
    testData := `{
        "device_id": "test-device",
        "timestamp": "2024-01-15T10:00:00Z",
        "data": {
            "cpu_usage": 45.5,
            "memory_usage": 67.8
        }
    }`

    err = natsConn.Publish("telemetry", []byte(testData))
    assert.NoError(t, err)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    go writer.Start(ctx)

    // Wait for processing
    time.Sleep(2 * time.Second)

    // Verify data was written to database
    var count int
    err = db.QueryRow(context.Background(),
        "SELECT COUNT(*) FROM telemetry WHERE device_id = $1", "test-device").Scan(&count)
    assert.NoError(t, err)
    assert.Equal(t, 1, count)
}
```

### End-to-End Testing

#### Agent to API E2E Test
```go
// e2e/agent_api_test.go
func TestAgent_API_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test")
    }

    // Start API server
    apiServer := setupTestAPIServer(t)
    defer apiServer.Close()

    // Create test agent
    agent := &Agent{
        Config: &Config{
            ServerURL: apiServer.URL,
            APIKey:    "test-api-key",
        },
    }

    // Register agent
    err := agent.Register()
    assert.NoError(t, err)
    assert.NotEmpty(t, agent.DeviceID)

    // Send inventory
    inventory := map[string]interface{}{
        "os": map[string]interface{}{
            "version": "Windows 10 Pro",
            "build":   "19045.2006",
        },
        "cpu": map[string]interface{}{
            "usage_percent": 25.5,
        },
    }

    err = agent.SendInventory(inventory)
    assert.NoError(t, err)

    // Verify data in database
    db := apiServer.DB()
    var count int
    err = db.QueryRow(context.Background(),
        "SELECT COUNT(*) FROM telemetry WHERE device_id = $1", agent.DeviceID).Scan(&count)
    assert.NoError(t, err)
    assert.True(t, count > 0)
}
```

#### Web Console E2E Test
```typescript
// e2e/dashboard.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // Login if authentication is required
  })

  test('displays dashboard statistics', async ({ page }) => {
    await expect(page.locator('h1').filter({ hasText: 'Dashboard' })).toBeVisible()

    // Check if stats are loaded
    await expect(page.locator('text=Total Devices')).toBeVisible()
    await expect(page.locator('text=Online')).toBeVisible()
    await expect(page.locator('text=Offline')).toBeVisible()
  })

  test('device list loads and paginates', async ({ page }) => {
    await page.click('text=View All')

    await expect(page.locator('text=Devices')).toBeVisible()

    // Check pagination
    const nextButton = page.locator('button').filter({ hasText: 'Next' })
    if (await nextButton.isVisible()) {
      await nextButton.click()
      await expect(page.locator('.device-item')).toHaveCount(10)
    }
  })
})
```

## Performance Testing

### Load Testing with k6

#### API Load Test
```javascript
// tools/k6/api_load_test.js
import http from 'k6/http'
import { check, sleep } from 'k6'

export let options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 200 },  // Ramp up to 200 users
    { duration: '5m', target: 200 },  // Stay at 200 users
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<300'], // 95% of requests should be below 300ms
    http_req_failed: ['rate<0.1'],    // Error rate should be below 10%
  },
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'

export default function () {
  // Device registration
  let registrationPayload = {
    hostname: `test-device-${__VU}-${Date.now()}`,
    capabilities: ['inventory'],
    os_version: 'Windows 10 Pro',
    os_build: '19045.2006',
    architecture: 'x64',
  }

  let registrationResponse = http.post(`${BASE_URL}/v1/agents/register`, JSON.stringify(registrationPayload), {
    headers: {
      'Content-Type': 'application/json',
    },
  })

  check(registrationResponse, {
    'registration successful': (r) => r.status === 200,
    'has device id': (r) => JSON.parse(r.body).data.id !== undefined,
  })

  let deviceId = JSON.parse(registrationResponse.body).data.id
  let apiKey = JSON.parse(registrationResponse.body).data.api_key

  // Send telemetry
  let telemetryPayload = {
    timestamp: new Date().toISOString(),
    data: {
      cpu_usage: Math.random() * 100,
      memory_usage: Math.random() * 100,
      disk_usage: Math.random() * 100,
    },
  }

  let telemetryResponse = http.post(`${BASE_URL}/v1/agents/${deviceId}/inventory`,
    JSON.stringify(telemetryPayload), {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey,
    },
  })

  check(telemetryResponse, {
    'telemetry accepted': (r) => r.status === 200,
  })

  // Query devices
  let devicesResponse = http.get(`${BASE_URL}/v1/devices?page=1&limit=10`)

  check(devicesResponse, {
    'devices query successful': (r) => r.status === 200,
  })

  sleep(1) // Wait 1 second between iterations
}
```

#### Database Performance Test
```javascript
// tools/k6/database_performance_test.js
import http from 'k6/http'
import { check } from 'k6'

export let options = {
  vus: 50,
  duration: '10m',
  thresholds: {
    http_req_duration: ['p(95)<500'], // Database queries should be fast
  },
}

export default function () {
  // Complex telemetry query
  let telemetryResponse = http.get('http://localhost:8080/v1/devices/device-001/telemetry?metric_type=cpu&start_time=2024-01-01T00:00:00Z&end_time=2024-01-31T23:59:59Z')

  check(telemetryResponse, {
    'telemetry query successful': (r) => r.status === 200,
    'response time acceptable': (r) => r.timings.duration < 1000,
  })

  // Device search with filters
  let searchResponse = http.get('http://localhost:8080/v1/devices?hostname=test*&status=online&limit=50')

  check(searchResponse, {
    'device search successful': (r) => r.status === 200,
  })
}
```

### Agent Performance Test
```go
// internal/collectors/performance_test.go
func BenchmarkOSCollector_Collect(b *testing.B) {
    collector := &OSCollector{}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := collector.Collect()
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkCPUCollector_Collect(b *testing.B) {
    collector := &CPUCollector{}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := collector.Collect()
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Security Testing

### Penetration Testing

#### API Security Tests
```bash
# OWASP ZAP automated scan
docker run -t owasp/zap2docker-stable zap-baseline.py \
  -t http://localhost:8080 \
  -r api_security_report.html

# SQL injection tests
sqlmap -u "http://localhost:8080/v1/devices?hostname=test" \
  --batch \
  --risk=3 \
  --level=5

# Authentication bypass attempts
# Test JWT token manipulation
# Test API key brute force
```

#### Authentication Tests
```go
// internal/auth/security_test.go
func TestJWT_InvalidTokens(t *testing.T) {
    tests := []struct {
        name  string
        token string
        valid bool
    }{
        {
            name:  "valid token",
            token: generateValidToken(),
            valid: true,
        },
        {
            name:  "expired token",
            token: generateExpiredToken(),
            valid: false,
        },
        {
            name:  "malformed token",
            token: "invalid.jwt.token",
            valid: false,
        },
        {
            name:  "tampered token",
            token: generateTamperedToken(),
            valid: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            claims, err := ValidateJWT(tt.token)
            if tt.valid {
                assert.NoError(t, err)
                assert.NotNil(t, claims)
            } else {
                assert.Error(t, err)
            }
        })
    }
}
```

### Dependency Vulnerability Scanning
```bash
# Go module vulnerabilities
go mod download
govulncheck ./...

# NPM audit
cd web
npm audit

# Container image scanning
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  anchore/grype inventory-api:latest

# Trivy comprehensive scan
trivy image inventory-api:latest
trivy fs --security-checks vuln,secret,misconfig .
```

## Test Automation

### CI/CD Integration

#### GitHub Actions Workflow
```yaml
# .github/workflows/test.yml
name: Test Suite

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      nats:
        image: nats:2.9

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.22'

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Run Go tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html

    - name: Upload coverage reports
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  test-web:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
        cache: 'npm'
        cache-dependency-path: web/package-lock.json

    - name: Install dependencies
      working-directory: ./web
      run: npm ci

    - name: Run tests
      working-directory: ./web
      run: npm test -- --coverage

    - name: Build
      working-directory: ./web
      run: npm run build

  e2e:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
      nats:
        image: nats:2.9

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.22'

    - name: Run E2E tests
      run: go test -v -tags=e2e ./e2e/...
      env:
        DATABASE_URL: postgres://postgres:postgres@localhost:5432/inventory_test
        NATS_URL: nats://localhost:4222

  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Run Trivy vulnerability scanner
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'sarif'
        output: 'trivy-results.sarif'

    - name: Upload Trivy scan results
      uses: github/codecov/codecov-action@v3
      if: always()
      with:
        file: trivy-results.sarif
```

### Test Data Management

#### Test Database Setup
```sql
-- Create test database
CREATE DATABASE inventory_test;

-- Load test data
\COPY devices FROM 'test_data/devices.csv' WITH CSV HEADER;
\COPY telemetry FROM 'test_data/telemetry.csv' WITH CSV HEADER;
```

#### Test Fixtures
```go
// internal/models/fixtures.go
func CreateTestDevice() *Device {
    return &Device{
        Hostname:     "TEST-DEVICE-001",
        OSVersion:    "Windows 10 Pro",
        OSBuild:      "19045.2006",
        Architecture: "x64",
        Status:       "online",
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
}

func CreateTestTelemetry(deviceID string) *TelemetryData {
    return &TelemetryData{
        DeviceID:   deviceID,
        Timestamp:  time.Now(),
        MetricType: "cpu",
        MetricName: "cpu_usage",
        Value:      45.5,
        Unit:       "percent",
    }
}
```

## Test Reporting

### Coverage Reports
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Coverage thresholds
go test -covermode=count -coverprofile=coverage.out ./...
coverage=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
if (( $(echo "$coverage < 80.0" | bc -l) )); then
    echo "Coverage $coverage% is below 80% threshold"
    exit 1
fi
```

### Performance Benchmarks
```go
// benchmarks_test.go
func BenchmarkAPI_Endpoints(b *testing.B) {
    benchmarks := []struct {
        name string
        fn   func(*testing.B)
    }{
        {"GetDevices", benchmarkGetDevices},
        {"GetDevice", benchmarkGetDevice},
        {"SendTelemetry", benchmarkSendTelemetry},
    }

    for _, bm := range benchmarks {
        b.Run(bm.name, bm.fn)
    }
}

func benchmarkGetDevices(b *testing.B) {
    // Setup test server and data
    // Run benchmark
}
```

### Test Results Dashboard
```typescript
// src/pages/testing/index.tsx
export default function TestingDashboard() {
  const [testResults, setTestResults] = useState<TestResults | null>(null)

  useEffect(() => {
    fetchTestResults().then(setTestResults)
  }, [])

  return (
    <div>
      <h1>Test Results Dashboard</h1>

      {testResults && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Card title="Unit Tests" value={`${testResults.unit.passed}/${testResults.unit.total}`} />
          <Card title="Integration Tests" value={`${testResults.integration.passed}/${testResults.integration.total}`} />
          <Card title="Coverage" value={`${testResults.coverage}%`} />
        </div>
      )}
    </div>
  )
}
```

This comprehensive testing strategy ensures the Windows Inventory Agent & Cloud Console system maintains high quality, performance, and security standards throughout development and production deployment.