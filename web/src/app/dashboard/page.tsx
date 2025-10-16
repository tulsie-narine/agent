import { Metadata } from 'next'
import { DashboardStats } from '@/components/dashboard/DashboardStats'
import { DeviceList } from '@/components/dashboard/DeviceList'
import { RecentActivity } from '@/components/dashboard/RecentActivity'

export const metadata: Metadata = {
  title: 'Dashboard - Inventory Console',
}

export default function DashboardPage() {
  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
          <p className="mt-2 text-gray-600">
            Overview of your Windows inventory system
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-8">
            <DashboardStats />
            <DeviceList />
          </div>
          <div>
            <RecentActivity />
          </div>
        </div>
      </div>
    </div>
  )
}