'use client'

import { useEffect, useState } from 'react'
import { DeviceStats } from '@/types'
import { apiClient } from '@/lib/api/client'
import { formatNumber } from '@/lib/utils'

export function DashboardStats() {
  const [stats, setStats] = useState<DeviceStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await apiClient.getDeviceStats()
        setStats(response.data)
      } catch (err) {
        setError('Failed to load device statistics')
        console.error('Error fetching device stats:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchStats()
  }, [])

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="card animate-pulse">
            <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
            <div className="h-8 bg-gray-200 rounded w-1/2"></div>
          </div>
        ))}
      </div>
    )
  }

  if (error || !stats) {
    return (
      <div className="card">
        <p className="text-red-600">{error || 'Failed to load statistics'}</p>
      </div>
    )
  }

  const statCards = [
    {
      title: 'Total Devices',
      value: stats.total,
      color: 'text-blue-600',
      bgColor: 'bg-blue-50',
    },
    {
      title: 'Online',
      value: stats.online,
      color: 'text-green-600',
      bgColor: 'bg-green-50',
    },
    {
      title: 'Offline',
      value: stats.offline,
      color: 'text-red-600',
      bgColor: 'bg-red-50',
    },
    {
      title: 'Unknown',
      value: stats.unknown,
      color: 'text-yellow-600',
      bgColor: 'bg-yellow-50',
    },
  ]

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
      {statCards.map((stat) => (
        <div key={stat.title} className={`card ${stat.bgColor}`}>
          <h3 className="text-sm font-medium text-gray-600 mb-2">{stat.title}</h3>
          <p className={`text-3xl font-bold ${stat.color}`}>
            {formatNumber(stat.value)}
          </p>
        </div>
      ))}
    </div>
  )
}