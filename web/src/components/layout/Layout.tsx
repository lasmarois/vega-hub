import { Outlet } from 'react-router-dom'
import { WifiOff, User as UserIcon } from 'lucide-react'
import { Header } from './Header'
import { Sidebar } from './Sidebar'
import { BottomNav } from './BottomNav'
import { NotificationCenter } from '@/components/shared/NotificationCenter'
import type { ActivityWithRead } from '@/hooks/useActivity'
import type { User } from '@/hooks/useUser'

interface LayoutProps {
  connected: boolean
  pendingQuestions: number
  activities: ActivityWithRead[]
  unreadCount: number
  onMarkAsRead: (id: string) => void
  onMarkAllAsRead: () => void
  onGoalClick?: (goalId: string) => void
  user?: User | null
}

export function Layout({
  connected,
  pendingQuestions,
  activities,
  unreadCount,
  onMarkAsRead,
  onMarkAllAsRead,
  onGoalClick,
  user,
}: LayoutProps) {
  return (
    <div className="min-h-screen bg-background">
      {/* Disconnection Banner - Full width, above everything */}
      {!connected && (
        <div className="sticky top-0 z-50 bg-destructive text-destructive-foreground px-4 py-2 flex items-center justify-center gap-2 text-sm">
          <WifiOff className="h-4 w-4" />
          <span>Connection lost. Reconnecting to vega-hub...</span>
        </div>
      )}

      {/* Desktop Sidebar */}
      <Sidebar pendingQuestions={pendingQuestions} />

      {/* Mobile/Tablet Header */}
      <div className="lg:hidden">
        <Header
          connected={connected}
          pendingQuestions={pendingQuestions}
          activities={activities}
          unreadCount={unreadCount}
          onMarkAsRead={onMarkAsRead}
          onMarkAllAsRead={onMarkAllAsRead}
          onGoalClick={onGoalClick}
          user={user}
        />
      </div>

      {/* Main Content */}
      <main className="lg:pl-60">
        {/* Desktop header with connection status and notifications */}
        <div className="hidden lg:block sticky top-0 z-30 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <div className="flex h-14 items-center justify-end gap-4 px-6">
            {user && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <UserIcon className="h-4 w-4" />
                <span>{user.username}</span>
              </div>
            )}
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>{connected ? 'Connected' : 'Reconnecting...'}</span>
              <div
                className={`h-2 w-2 rounded-full ${
                  connected ? 'bg-green-500' : 'bg-red-500 animate-pulse'
                }`}
              />
            </div>
            <NotificationCenter
              activities={activities}
              unreadCount={unreadCount}
              onMarkAsRead={onMarkAsRead}
              onMarkAllAsRead={onMarkAllAsRead}
              onGoalClick={onGoalClick}
            />
          </div>
        </div>

        {/* Page content with bottom padding for mobile nav */}
        <div className="pb-20 lg:pb-0">
          <Outlet />
        </div>
      </main>

      {/* Mobile/Tablet Bottom Nav */}
      <BottomNav pendingQuestions={pendingQuestions} />
    </div>
  )
}
