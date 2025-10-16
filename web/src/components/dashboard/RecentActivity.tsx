'use client'

import { useEffect, useState } from 'react'
import type { Command } from '../../types/index'
import { apiClient } from '../../lib/api/client'
import { formatRelativeTime } from '../../lib/utils/index'

interface ActivityItem {
  id: string
  type: 'command' | 'device' | 'policy'
  title: string
  description: string
  timestamp: string
}

export function RecentActivity() {
  const [activities, setActivities] = useState<ActivityItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchRecentActivity = async () => {
      try {
        // Fetch recent commands as activity
        const commandsResponse = await apiClient.getCommands()
        const commands = commandsResponse.data.slice(0, 5)

        const activityItems: ActivityItem[] = commands.map((cmd: Command) => ({
          id: cmd.id,
          type: 'command',
          title: `Command ${cmd.command_type}`,
          description: `Status: ${cmd.status}`,
          timestamp: cmd.updated_at,
        }))

        // Add some mock device activities for demo
        activityItems.push(
          {
            id: 'mock-device-1',
            type: 'device',
            title: 'New device registered',
            description: 'WIN-ABC123 joined the network',
            timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString(), // 30 minutes ago
          },
          {
            id: 'mock-policy-1',
            type: 'policy',
            title: 'Policy updated',
            description: 'Security policy v2.1 deployed',
            timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString(), // 2 hours ago
          }
        )

        // Sort by timestamp (most recent first)
        activityItems.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())

        setActivities(activityItems.slice(0, 5))
      } catch (err) {
        setError('Failed to load recent activity')
        console.error('Error fetching recent activity:', err)
      } finally {
        setLoading(false)
      }
    }

    fetchRecentActivity()
  }, [])

  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'command': return '⚡'
      case 'device': return '🖥️'
      case 'policy': return '📋'
      default: return '📝'
    }
  }

  if (loading) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold mb-4">Recent Activity</h2>
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="animate-pulse flex items-start space-x-3">
              <div className="h-8 w-8 bg-gray-200 rounded-full"></div>
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
        <h2 className="text-xl font-semibold mb-4">Recent Activity</h2>
        <p className="text-red-600">{error}</p>
      </div>
    )
  }

  return (
    <div className="card">
      <h2 className="text-xl font-semibold mb-6">Recent Activity</h2>

      <div className="space-y-4">
        {activities.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No recent activity</p>
        ) : (
          activities.map((activity) => (
            <div key={activity.id} className="flex items-start space-x-3">
              <div className="text-2xl">{getActivityIcon(activity.type)}</div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900">{activity.title}</p>
                <p className="text-sm text-gray-600">{activity.description}</p>
                <p className="text-xs text-gray-500 mt-1">
                  {formatRelativeTime(activity.timestamp)}
                </p>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}