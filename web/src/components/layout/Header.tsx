import { Badge } from '@/components/ui/badge'
import { NotificationCenter } from '@/components/shared/NotificationCenter'
import { cn } from '@/lib/utils'
import type { ActivityWithRead } from '@/hooks/useActivity'

interface HeaderProps {
  connected: boolean
  pendingQuestions: number
  activities: ActivityWithRead[]
  unreadCount: number
  onMarkAsRead: (id: string) => void
  onMarkAllAsRead: () => void
  onGoalClick?: (goalId: string) => void
}

export function Header({
  connected,
  pendingQuestions,
  activities,
  unreadCount,
  onMarkAsRead,
  onMarkAllAsRead,
  onGoalClick,
}: HeaderProps) {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex h-14 items-center justify-between px-4">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold">vega-hub</h1>
          <div
            className={cn(
              "h-2 w-2 rounded-full",
              connected ? "bg-green-500" : "bg-red-500 animate-pulse"
            )}
            title={connected ? "Connected" : "Disconnected"}
          />
        </div>

        <div className="flex items-center gap-2">
          {pendingQuestions > 0 && (
            <Badge variant="destructive" className="gap-1.5">
              <span className="text-sm">Pending</span>
              <span className="font-bold">{pendingQuestions}</span>
            </Badge>
          )}
          <NotificationCenter
            activities={activities}
            unreadCount={unreadCount}
            onMarkAsRead={onMarkAsRead}
            onMarkAllAsRead={onMarkAllAsRead}
            onGoalClick={onGoalClick}
          />
        </div>
      </div>
    </header>
  )
}
