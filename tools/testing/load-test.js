import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const telemetryTrend = new Trend('telemetry_duration');
const policyTrend = new Trend('policy_duration');
const commandTrend = new Trend('command_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up to 10 users
    { duration: '5m', target: 50 },   // Ramp up to 50 users
    { duration: '10m', target: 100 }, // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.1'],    // Error rate should be below 10%
    errors: ['rate<0.1'],             // Custom error rate
  },
};

// Base URL for the API
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';

// Test data
const testDevices = [
  { id: 'device-001', hostname: 'WIN-TEST-001' },
  { id: 'device-002', hostname: 'WIN-TEST-002' },
  { id: 'device-003', hostname: 'WIN-TEST-003' },
  { id: 'device-004', hostname: 'WIN-TEST-004' },
  { id: 'device-005', hostname: 'WIN-TEST-005' },
];

const testPolicies = [
  {
    name: 'Test Security Policy',
    description: 'Load testing security policy',
    version: '1.0.0',
    enabled: true,
    target_filters: {
      hostname_pattern: 'WIN-TEST-*',
    },
    content: {
      security: {
        antivirus: { enabled: true },
        firewall: { enabled: true },
      },
    },
  },
];

const testCommands = [
  {
    command_type: 'collect_inventory',
    command_data: {
      inventory_types: ['os', 'hardware', 'software'],
      force_full_scan: false,
    },
    priority: 1,
    timeout_seconds: 300,
  },
  {
    command_type: 'run_script',
    command_data: {
      script: 'Get-ComputerInfo | ConvertTo-Json',
      script_type: 'powershell',
      run_as_admin: false,
      timeout_seconds: 60,
    },
    priority: 2,
    timeout_seconds: 120,
  },
];

// Generate sample telemetry data
function generateTelemetryData(device) {
  return {
    timestamp: new Date().toISOString(),
    data: {
      os: {
        version: '10.0.19045',
        build: '19045.3086',
        architecture: 'x64',
        install_date: '2023-01-15T10:00:00Z',
      },
      hardware: {
        cpu: {
          model: 'Intel(R) Core(TM) i7-10700K CPU @ 3.80GHz',
          cores: 8,
          threads: 16,
          speed_mhz: 3800,
          usage_percent: Math.random() * 100,
        },
        memory: {
          total_gb: 32,
          available_gb: 16 + Math.random() * 16,
          usage_percent: Math.random() * 100,
        },
        disks: [
          {
            device: 'C:',
            total_gb: 500,
            free_gb: 200 + Math.random() * 200,
            type: 'SSD',
            usage_percent: Math.random() * 100,
          },
        ],
      },
      software: [
        {
          name: 'Google Chrome',
          version: '120.0.6099.109',
          publisher: 'Google LLC',
          install_date: '2023-12-01',
          size_mb: 250,
        },
        {
          name: 'Microsoft Office',
          version: '16.0.17029.20028',
          publisher: 'Microsoft Corporation',
          install_date: '2023-11-15',
          size_mb: 2048,
        },
      ],
      security: {
        antivirus: {
          enabled: true,
          vendor: 'Windows Defender',
          version: '4.18.23110.3',
          last_scan: new Date(Date.now() - Math.random() * 86400000).toISOString(),
          threat_count: 0,
        },
        firewall: {
          enabled: true,
          profile: 'Domain',
        },
        updates: {
          last_update: new Date(Date.now() - Math.random() * 604800000).toISOString(),
          pending_updates: Math.floor(Math.random() * 5),
          auto_update: true,
        },
      },
    },
  };
}

// Main test function
export default function () {
  const device = testDevices[Math.floor(Math.random() * testDevices.length)];

  // Scenario 1: Submit telemetry data (most common operation)
  const telemetryData = generateTelemetryData(device);
  const telemetryResponse = http.post(
    `${BASE_URL}/telemetry`,
    JSON.stringify(telemetryData),
    {
      headers: {
        'Content-Type': 'application/json',
        'X-Device-ID': device.id,
      },
    }
  );

  const telemetryCheck = check(telemetryResponse, {
    'telemetry submission successful': (r) => r.status === 200 || r.status === 201,
    'telemetry response time < 200ms': (r) => r.timings.duration < 200,
  });

  telemetryTrend.add(telemetryResponse.timings.duration);
  errorRate.add(!telemetryCheck);

  sleep(Math.random() * 2 + 1); // Random sleep between 1-3 seconds

  // Scenario 2: Get policies for device (less frequent)
  if (Math.random() < 0.3) { // 30% of requests
    const policyResponse = http.get(
      `${BASE_URL}/policies?device_id=${device.id}`,
      {
        headers: {
          'X-Device-ID': device.id,
        },
      }
    );

    const policyCheck = check(policyResponse, {
      'policy retrieval successful': (r) => r.status === 200,
      'policy response time < 100ms': (r) => r.timings.duration < 100,
    });

    policyTrend.add(policyResponse.timings.duration);
    errorRate.add(!policyCheck);

    sleep(Math.random() * 1 + 0.5);
  }

  // Scenario 3: Get pending commands (occasional)
  if (Math.random() < 0.2) { // 20% of requests
    const commandResponse = http.get(
      `${BASE_URL}/commands/pending?device_id=${device.id}`,
      {
        headers: {
          'X-Device-ID': device.id,
        },
      }
    );

    const commandCheck = check(commandResponse, {
      'command retrieval successful': (r) => r.status === 200,
      'command response time < 150ms': (r) => r.timings.duration < 150,
    });

    commandTrend.add(commandResponse.timings.duration);
    errorRate.add(!commandCheck);

    sleep(Math.random() * 1 + 0.5);
  }

  // Scenario 4: Update command status (rare)
  if (Math.random() < 0.1) { // 10% of requests
    const commandId = `cmd-${Math.floor(Math.random() * 1000)}`;
    const statusUpdate = {
      status: 'completed',
      result: { success: true, message: 'Command executed successfully' },
    };

    const updateResponse = http.put(
      `${BASE_URL}/commands/${commandId}/status`,
      JSON.stringify(statusUpdate),
      {
        headers: {
          'Content-Type': 'application/json',
          'X-Device-ID': device.id,
        },
      }
    );

    const updateCheck = check(updateResponse, {
      'command status update successful': (r) => r.status === 200,
      'command update response time < 100ms': (r) => r.timings.duration < 100,
    });

    commandTrend.add(updateResponse.timings.duration);
    errorRate.add(!updateCheck);

    sleep(Math.random() * 1 + 0.5);
  }

  // Random sleep to simulate real-world usage patterns
  sleep(Math.random() * 5 + 2); // Sleep between 2-7 seconds
}

// Setup function - runs before the test starts
export function setup() {
  console.log('Starting load test setup...');

  // Create test policies
  for (const policy of testPolicies) {
    const response = http.post(
      `${BASE_URL}/policies`,
      JSON.stringify(policy),
      {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token', // In real tests, use proper auth
        },
      }
    );

    check(response, {
      'policy creation successful': (r) => r.status === 201,
    });
  }

  // Create test commands
  for (const command of testCommands) {
    const response = http.post(
      `${BASE_URL}/commands`,
      JSON.stringify({
        ...command,
        device_id: testDevices[Math.floor(Math.random() * testDevices.length)].id,
      }),
      {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
        },
      }
    );

    check(response, {
      'command creation successful': (r) => r.status === 201,
    });
  }

  console.log('Load test setup completed');
  return { testDevices, testPolicies, testCommands };
}

// Teardown function - runs after the test completes
export function teardown(data) {
  console.log('Starting load test teardown...');

  // Clean up test data
  console.log(`Test completed. Devices tested: ${data.testDevices.length}`);
  console.log(`Policies created: ${data.testPolicies.length}`);
  console.log(`Commands created: ${data.testCommands.length}`);

  console.log('Load test teardown completed');
}