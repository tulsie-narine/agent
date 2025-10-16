'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import type { Device } from '../../types/index'
import { apiClient } from '../../lib/api/client'
import { formatDate, formatRelativeTime, getDeviceStatus } from '../../lib/utils/index'

export function DeviceList() {
  const [devices, setDevices] = useState<Device[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchDevices = async () => {
      try {
        const response = await apiClient.getDevices(1, 10)
        setDevices(response.data)
      } catch (err) {
        setError('Failed to load devices')
        console.error('Error fetching devices:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchDevices()
  }, [])

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online': return 'bg-green-100 text-green-800'
      case 'offline': return 'bg-red-100 text-red-800'
      default: return 'bg-yellow-100 text-yellow-800'
    }
  }

  if (loading) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold mb-4">Recent Devices</h2>
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="animate-pulse flex items-center space-x-4">
              <div className="h-10 w-10 bg-gray-200 rounded-full"></div>
              <div className="flex-1 space-y-2">
                <div className="h-4 bg-gray-200 rounded w-3/4"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2"></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold mb-4">Recent Devices</h2>
        <p className="text-red-600">{error}</p>
      </div>
    )
  }

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold">Recent Devices</h2>
        <Link
          href="/devices"
          className="btn btn-secondary text-sm"
        >
          View All
        </Link>
      </div>

      <div className="space-y-4">
        {devices.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No devices registered yet</p>
        ) : (
          devices.map((device) => {
            const status = getDeviceStatus(device.last_seen)
            return (
              <div key={device.id} className="flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:bg-gray-50">
                <div className="flex items-center space-x-4">
                  <div className="h-10 w-10 bg-blue-100 rounded-full flex items-center justify-center">
                    <span className="text-blue-600 font-semibold text-sm">
                      {device.hostname.charAt(0).toUpperCase()}
                    </span>
                  </div>
                  <div>
                    <h3 className="font-medium text-gray-900">{device.hostname}</h3>
                    <p className="text-sm text-gray-600">
                      {device.os_version} • {device.architecture}
                    </p>
                  </div>
                </div>

                <div className="flex items-center space-x-4">
                  <div className="text-right">
                    <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(status)}`}>
                      {status}
                    </span>
                    <p className="text-xs text-gray-500 mt-1">
                      {formatRelativeTime(device.last_seen)}
                    </p>
                  </div>
                  <Link
                    href={`/devices/${device.id}`}
                    className="text-blue-600 hover:text-blue-800 text-sm font-medium"
                  >
                    View →
                  </Link>
                </div>
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}