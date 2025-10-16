import axios, { AxiosInstance, AxiosResponse } from 'axios'
import { ApiResponse, PaginatedResponse, Device, DeviceStats, TelemetryData, TelemetryQuery, Policy, Command } from '../../types/index'

class ApiClient {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/v1',
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    // Add request interceptor for auth
    this.client.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('auth_token')
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => Promise.reject(error)
    )

    // Add response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          // Handle unauthorized - redirect to login
          localStorage.removeItem('auth_token')
          window.location.href = '/login'
        }
        return Promise.reject(error)
      }
    )
  }

  // Device endpoints
  async getDevices(page = 1, limit = 50, filters?: Record<string, any>): Promise<PaginatedResponse<Device>> {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString(),
      ...filters,
    })

    const response = await this.client.get(`/devices?${params}`)
    return response.data
  }

  async getDevice(id: string): Promise<ApiResponse<Device>> {
    const response = await this.client.get(`/devices/${id}`)
    return response.data
  }

  async getDeviceStats(): Promise<ApiResponse<DeviceStats>> {
    const response = await this.client.get('/devices/stats')
    return response.data
  }

  // Telemetry endpoints
  async getDeviceTelemetry(deviceId: string, query?: TelemetryQuery): Promise<PaginatedResponse<TelemetryData>> {
    const params = new URLSearchParams()
    if (query) {
      Object.entries(query).forEach(([key, value]) => {
        if (value !== undefined) {
          params.append(key, value.toString())
        }
      })
    }

    const response = await this.client.get(`/devices/${deviceId}/telemetry?${params}`)
    return response.data
  }

  // Policy endpoints
  async getPolicies(): Promise<ApiResponse<Policy[]>> {
    const response = await this.client.get('/policies')
    return response.data
  }

  async createPolicy(policy: Omit<Policy, 'id' | 'created_at' | 'updated_at'>): Promise<ApiResponse<Policy>> {
    const response = await this.client.post('/policies', policy)
    return response.data
  }

  async updatePolicy(id: string, policy: Partial<Policy>): Promise<ApiResponse<Policy>> {
    const response = await this.client.put(`/policies/${id}`, policy)
    return response.data
  }

  // Command endpoints
  async getCommands(deviceId?: string): Promise<ApiResponse<Command[]>> {
    const params = deviceId ? `?device_id=${deviceId}` : ''
    const response = await this.client.get(`/commands${params}`)
    return response.data
  }

  async createCommand(command: Omit<Command, 'id' | 'status' | 'created_at' | 'updated_at'>): Promise<ApiResponse<Command>> {
    const response = await this.client.post('/commands', command)
    return response.data
  }

  async ackCommand(deviceId: string, commandId: string): Promise<ApiResponse<void>> {
    const response = await this.client.post(`/agents/${deviceId}/commands/${commandId}/ack`)
    return response.data
  }

  // Health check
  async healthCheck(): Promise<ApiResponse<any>> {
    const response = await this.client.get('/health')
    return response.data
  }
}

export const apiClient = new ApiClient()