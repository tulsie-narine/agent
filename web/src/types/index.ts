// Device types
export interface Device {
  id: string
  hostname: string
  os_version: string
  os_build: string
  architecture: string
  domain: string
  last_seen: string
  status: 'online' | 'offline' | 'unknown'
  created_at: string
  updated_at: string
}

export interface DeviceStats {
  total: number
  online: number
  offline: number
  unknown: number
}

// Telemetry types
export interface TelemetryData {
  id: string
  device_id: string
  timestamp: string
  metric_type: string
  metric_name: string
  value: number | string | boolean
  unit?: string
  metadata?: Record<string, any>
}

export interface TelemetryQuery {
  device_id?: string
  metric_type?: string
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
}

// Policy types
export interface Policy {
  id: string
  name: string
  description: string
  version: string
  content: Record<string, any>
  target_filters: Record<string, any>
  enabled: boolean
  created_at: string
  updated_at: string
}

// Command types
export interface Command {
  id: string
  device_id: string
  command_type: string
  command_data: Record<string, any>
  status: 'pending' | 'running' | 'completed' | 'failed' | 'expired'
  result?: Record<string, any>
  error_message?: string
  created_at: string
  updated_at: string
  expires_at?: string
}

// API Response types
export interface ApiResponse<T> {
  data: T
  success: boolean
  message?: string
  pagination?: {
    page: number
    limit: number
    total: number
    total_pages: number
  }
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
  pagination: {
    page: number
    limit: number
    total: number
    total_pages: number
  }
}

// Form types
export interface LoginForm {
  username: string
  password: string
}

export interface RegisterForm {
  hostname: string
  capabilities: string[]
}

// Chart data types
export interface ChartDataPoint {
  timestamp: string
  value: number
  label?: string
}

export interface MetricChart {
  title: string
  data: ChartDataPoint[]
  type: 'line' | 'bar' | 'area'
  color?: string
}