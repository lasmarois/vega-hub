import { Bell, Play, Square, MessageSquare, CheckCircle, Snowflake, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import type { ActivityWithRead } from '@/hooks/useActivity'

interface NotificationCenterProps {
  activities: ActivityWithRead[]
  unreadCount: number
  onMarkAsRead: (id: string) => void
  onMarkAllAsRead: () => void
  onGoalClick?: (goalId: string) => void
}

function formatRelativeTime(timestamp: string): string {
  const now = new Date()
  const then = new Date(timestamp)
  const diffMs = now.getTime() - then.getTime()
  const diffSecs = Math.floor(diffMs / 1000)
  const diffMins = Math.floor(diffSecs / 60)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffSecs < 60) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  return then.toLocaleDateString()
}

function getActivityIcon(type: ActivityWithRead['type']) {
  switch (type) {
    case 'executor_started':
      return <Play className="h-4 w-4 text-green-500" />
    case 'executor_stopped':
      return <Square className="h-4 w-4 text-muted-foreground" />
    case 'question':
      return <MessageSquare className="h-4 w-4 text-red-500" />
    case 'answered':
      return <CheckCircle className="h-4 w-4 text-green-500" />
    case 'goal_completed':
      return <CheckCircle className="h-4 w-4 text-green-500" />
    case 'goal_iced':
      return <Snowflake className="h-4 w-4 text-blue-500" />
    case 'goal_updated':
      return <RefreshCw className="h-4 w-4 text-muted-foreground" />
    default:
      return <Bell className="h-4 w-4" />
  }
}

export function NotificationCenter({
  activities,
  unreadCount,
  onMarkAsRead,
  onMarkAllAsRead,
  onGoalClick,
}: NotificationCenterProps) {
  const handleItemClick = (activity: ActivityWithRead) => {
    if (!activity.read) {
      onMarkAsRead(activity.id)
    }
    if (activity.goal_id && onGoalClick) {
      onGoalClick(activity.goal_id)
    }
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="relative">
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <Badge
              variant="destructive"
              className="absolute -top-1 -right-1 h-5 w-5 p-0 flex items-center justify-center text-xs"
            >
              {unreadCount > 9 ? '9+' : unreadCount}
            </Badge>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 p-0">
        <div className="flex items-center justify-between p-3 border-b">
          <h4 className="font-semibold text-sm">Notifications</h4>
          {unreadCount > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="h-auto py-1 px-2 text-xs"
              onClick={onMarkAllAsRead}
            >
              Mark all read
            </Button>
          )}
        </div>
        <ScrollArea className="h-[300px]">
          {activities.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground py-8">
              <Bell className="h-8 w-8 mb-2 opacity-50" />
              <p className="text-sm">No notifications yet</p>
            </div>
          ) : (
            <div className="divide-y">
              {activities.slice(0, 20).map((activity) => (
                <button
                  key={activity.id}
                  className={cn(
                    'w-full flex items-start gap-3 p-3 text-left hover:bg-accent/50 transition-colors',
                    !activity.read && 'bg-accent/30'
                  )}
                  onClick={() => handleItemClick(activity)}
                >
                  <div className="mt-0.5">
                    {getActivityIcon(activity.type)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className={cn(
                      'text-sm truncate',
                      !activity.read && 'font-medium'
                    )}>
                      {activity.message}
                    </p>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground mt-0.5">
                      {activity.goal_id && (
                        <span>#{activity.goal_id}</span>
                      )}
                      <span>{formatRelativeTime(activity.timestamp)}</span>
                    </div>
                  </div>
                  {!activity.read && (
                    <div className="h-2 w-2 rounded-full bg-primary mt-1.5 shrink-0" />
                  )}
                </button>
              ))}
            </div>
          )}
        </ScrollArea>
      </PopoverContent>
    </Popover>
  )
}
