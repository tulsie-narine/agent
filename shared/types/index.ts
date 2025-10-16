// Shared TypeScript types for cross-component compatibility

// Telemetry types
export interface TelemetryData {
  timestamp: string
  data: {
    os?: OSInfo
    hardware?: HardwareInfo
    software?: SoftwareInfo[]
    security?: SecurityInfo
  }
}

export interface OSInfo {
  version: string
  build: string
  architecture: 'x86' | 'x64' | 'arm' | 'arm64'
  install_date?: string
}

export interface HardwareInfo {
  cpu?: CPUInfo
  memory?: MemoryInfo
  disks?: DiskInfo[]
  network?: NetworkInfo
}

export interface CPUInfo {
  model?: string
  cores: number
  threads: number
  speed_mhz?: number
  usage_percent?: number
}

export interface MemoryInfo {
  total_gb: number
  available_gb?: number
  usage_percent?: number
}

export interface DiskInfo {
  device: string
  total_gb: number
  free_gb: number
  type?: 'HDD' | 'SSD' | 'NVMe' | 'Unknown'
  usage_percent?: number
}

export interface NetworkInfo {
  interfaces?: NetworkInterface[]
}

export interface NetworkInterface {
  name: string
  mac_address?: string
  ip_addresses?: string[]
  status?: 'up' | 'down' | 'unknown'
}

export interface SoftwareInfo {
  name: string
  version?: string
  publisher?: string
  install_date?: string
  install_location?: string
  size_mb?: number
}

export interface SecurityInfo {
  antivirus?: AntivirusInfo
  firewall?: FirewallInfo
  updates?: UpdateInfo
}

export interface AntivirusInfo {
  enabled?: boolean
  vendor?: string
  version?: string
  last_scan?: string
  threat_count?: number
}

export interface FirewallInfo {
  enabled?: boolean
  profile?: 'Domain' | 'Private' | 'Public'
}

export interface UpdateInfo {
  last_update?: string
  pending_updates?: number
  auto_update?: boolean
}

// Policy types
export interface Policy {
  id?: string
  name: string
  description?: string
  version: string
  enabled?: boolean
  priority?: number
  target_filters?: PolicyFilters
  content: PolicyContent
  metadata?: PolicyMetadata
}

export interface PolicyFilters {
  hostname_pattern?: string
  os_version?: string
  os_build_min?: string
  os_build_max?: string
  architecture?: 'x86' | 'x64' | 'arm' | 'arm64'
  domain?: string
  group_tags?: string[]
  custom_properties?: Record<string, string>
}

export interface PolicyContent {
  telemetry?: TelemetryPolicy
  security?: SecurityPolicy
  compliance?: CompliancePolicy
  monitoring?: MonitoringPolicy
}

export interface TelemetryPolicy {
  enabled?: boolean
  interval_seconds?: number
  collectors?: {
    os?: boolean
    hardware?: boolean
    software?: boolean
    security?: boolean
    performance?: boolean
  }
  retention_days?: number
}

export interface SecurityPolicy {
  antivirus?: AntivirusPolicy
  firewall?: FirewallPolicy
  updates?: UpdatePolicy
  encryption?: EncryptionPolicy
}

export interface AntivirusPolicy {
  enabled?: boolean
  vendor_preference?: string[]
  scan_schedule?: 'daily' | 'weekly' | 'disabled'
  real_time_protection?: boolean
}

export interface FirewallPolicy {
  enabled?: boolean
  profile?: 'domain' | 'private' | 'public' | 'all'
  rules?: FirewallRule[]
}

export interface FirewallRule {
  name: string
  action: 'allow' | 'block'
  direction: 'inbound' | 'outbound'
  protocol?: 'tcp' | 'udp' | 'icmp' | 'any'
  local_port?: string
  remote_address?: string
  enabled?: boolean
}

export interface UpdatePolicy {
  automatic_updates?: boolean
  update_schedule?: {
    day?: string
    hour?: number
  }
  feature_updates?: boolean
  quality_updates?: boolean
}

export interface EncryptionPolicy {
  bitlocker?: boolean
  removable_drives?: boolean
}

export interface CompliancePolicy {
  enabled?: boolean
  check_interval_hours?: number
  grace_period_days?: number
  auto_remediate?: boolean
}

export interface MonitoringPolicy {
  heartbeat_interval_seconds?: number
  log_level?: 'error' | 'warn' | 'info' | 'debug'
  log_retention_days?: number
  performance_monitoring?: boolean
}

export interface PolicyMetadata {
  created_by?: string
  created_at?: string
  updated_by?: string
  updated_at?: string
  tags?: string[]
}

// Command types
export interface Command {
  id?: string
  device_id?: string
  command_type: CommandType
  command_data: CommandData
  status?: CommandStatus
  priority?: number
  timeout_seconds?: number
  expires_at?: string
  created_by?: string
  created_at?: string
  metadata?: CommandMetadata
}

export type CommandType =
  | 'run_script'
  | 'install_software'
  | 'uninstall_software'
  | 'update_policy'
  | 'restart_service'
  | 'shutdown_device'
  | 'run_command'
  | 'collect_inventory'
  | 'update_agent'
  | 'custom_command'

export type CommandStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'expired'
  | 'cancelled'

export type CommandData =
  | RunScriptData
  | InstallSoftwareData
  | UninstallSoftwareData
  | UpdatePolicyData
  | RestartServiceData
  | ShutdownDeviceData
  | RunCommandData
  | CollectInventoryData
  | UpdateAgentData
  | CustomCommandData

export interface RunScriptData {
  script: string
  script_type?: 'powershell' | 'batch' | 'exe'
  parameters?: string[]
  working_directory?: string
  run_as_admin?: boolean
  timeout_seconds?: number
}

export interface InstallSoftwareData {
  software_name: string
  installer_url?: string
  installer_path?: string
  installer_type?: 'msi' | 'exe' | 'msix' | 'appx'
  install_arguments?: string
  uninstall_key?: string
  expected_exit_codes?: number[]
  timeout_seconds?: number
}

export interface UninstallSoftwareData {
  software_name: string
  uninstall_key?: string
  uninstall_command?: string
  force_uninstall?: boolean
  timeout_seconds?: number
}

export interface UpdatePolicyData {
  policy_id: string
  policy_content: Record<string, any>
  force_update?: boolean
}

export interface RestartServiceData {
  service_name: string
  timeout_seconds?: number
}

export interface ShutdownDeviceData {
  shutdown_type?: 'shutdown' | 'restart' | 'hibernate' | 'sleep'
  delay_seconds?: number
  force?: boolean
  message?: string
}

export interface RunCommandData {
  command: string
  arguments?: string[]
  working_directory?: string
  run_as_admin?: boolean
  timeout_seconds?: number
}

export interface CollectInventoryData {
  inventory_types?: ('os' | 'hardware' | 'software' | 'security' | 'all')[]
  force_full_scan?: boolean
  timeout_seconds?: number
}

export interface UpdateAgentData {
  update_url: string
  update_version: string
  force_update?: boolean
  rollback_on_failure?: boolean
  timeout_seconds?: number
}

export interface CustomCommandData {
  custom_type: string
  parameters?: Record<string, any>
  timeout_seconds?: number
}

export interface CommandMetadata {
  correlation_id?: string
  tags?: string[]
  notification_settings?: {
    notify_on_start?: boolean
    notify_on_completion?: boolean
    notify_on_failure?: boolean
  }
}

// API Response types
export interface ApiResponse<T = any> {
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

// Validation result types
export interface ValidationResult {
  valid: boolean
  errors?: ValidationError[]
}

export interface ValidationError {
  field: string
  message: string
  code?: string
}

// Schema validation types
export interface SchemaValidationOptions {
  strict?: boolean
  allowUnknownProperties?: boolean
  removeAdditionalProperties?: boolean
}

export interface SchemaValidator {
  validate(data: any, schema: any, options?: SchemaValidationOptions): ValidationResult
  validateAsync(data: any, schema: any, options?: SchemaValidationOptions): Promise<ValidationResult>
}